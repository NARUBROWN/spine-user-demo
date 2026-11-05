//go:build e2e

package e2e

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestHTTPContractsAndTransactionBoundaries(t *testing.T) {
	baseURL := requiredEnv(t, "SPINE_DEMO_E2E_BASE_URL")
	db := openDB(t, requiredEnv(t, "SPINE_DEMO_MYSQL_DSN"))
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	successEmail := "success-" + suffix + "@example.com"
	var createdUser struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	controllerErrorEmail := "controller-error-" + suffix + "@example.com"
	responseErrorEmail := "response-error-" + suffix + "@example.com"
	for _, email := range []string{successEmail, controllerErrorEmail, responseErrorEmail} {
		email := email
		t.Cleanup(func() { _, _ = db.Exec("DELETE FROM users WHERE email = ?", email) })
	}

	t.Run("native Swagger route serves the API document", func(t *testing.T) {
		status, body := requestJSON(t, http.MethodGet, baseURL+"/swagger/doc.json", "")
		assertStatus(t, status, http.StatusOK)
		var document struct {
			Paths map[string]json.RawMessage `json:"paths"`
		}
		if err := json.Unmarshal([]byte(body), &document); err != nil || document.Paths["/users"] == nil {
			t.Fatalf("Swagger user API document: %v, body=%s", err, body)
		}
	})

	t.Run("successful response commits", func(t *testing.T) {
		status, body := postJSON(t, baseURL+"/users", userJSON("Alice", successEmail))
		assertStatus(t, status, http.StatusOK)
		assertUserCount(t, db, successEmail, 1)
		if err := json.Unmarshal([]byte(body), &createdUser); err != nil || createdUser.ID <= 0 || createdUser.Name != "Alice" || createdUser.Email != successEmail {
			t.Fatalf("created user: %#v, error=%v, body=%s", createdUser, err, body)
		}
	})

	t.Run("invalid input returns 400", func(t *testing.T) {
		status, _ := postJSON(t, baseURL+"/users", userJSON("", "not-an-email"))
		assertStatus(t, status, http.StatusBadRequest)
	})

	for _, query := range []string{"", "?id=0", "?id=-1", "?id=abc", "?id=9223372036854775808"} {
		t.Run("invalid user ID "+query, func(t *testing.T) {
			status, _ := requestJSON(t, http.MethodGet, baseURL+"/users"+query, "")
			assertStatus(t, status, http.StatusBadRequest)
		})
	}

	t.Run("missing user returns 404", func(t *testing.T) {
		response, err := http.Get(baseURL + "/users?id=2147483647")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		assertStatus(t, response.StatusCode, http.StatusNotFound)
	})

	t.Run("duplicate email returns 409 and rolls back", func(t *testing.T) {
		status, _ := postJSON(t, baseURL+"/users", userJSON("Duplicate", successEmail))
		assertStatus(t, status, http.StatusConflict)
		assertUserCount(t, db, successEmail, 1)
	})

	t.Run("controller error rolls back", func(t *testing.T) {
		status, _ := postJSON(t, baseURL+"/scenarios/transactions/controller-error", userJSON("Alice", controllerErrorEmail))
		assertStatus(t, status, http.StatusBadRequest)
		assertUserCount(t, db, controllerErrorEmail, 0)
	})

	t.Run("response preparation error rolls back", func(t *testing.T) {
		status, _ := postJSON(t, baseURL+"/scenarios/transactions/response-error", userJSON("Alice", responseErrorEmail))
		assertStatus(t, status, http.StatusInternalServerError)
		assertUserCount(t, db, responseErrorEmail, 0)
	})

	t.Run("duplicate ID query is rejected", func(t *testing.T) {
		if createdUser.ID <= 0 {
			t.Fatal("successful creation is required")
		}
		endpoint := fmt.Sprintf("%s/users?id=%d&id=999", baseURL, createdUser.ID)
		status, _ := requestJSON(t, http.MethodGet, endpoint, "")
		assertStatus(t, status, http.StatusBadRequest)
	})

	t.Run("read update and delete preserve the CRUD contract", func(t *testing.T) {
		if createdUser.ID <= 0 {
			t.Fatal("successful creation is required for the CRUD round trip")
		}
		endpoint := fmt.Sprintf("%s/users?id=%d", baseURL, createdUser.ID)
		status, body := requestJSON(t, http.MethodGet, endpoint, "")
		assertStatus(t, status, http.StatusOK)
		var user struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.Unmarshal([]byte(body), &user); err != nil || user != createdUser {
			t.Fatalf("read user: %#v, error=%v, body=%s", user, err, body)
		}
		status, _ = requestJSON(t, http.MethodPut, endpoint, `{"name":"Bob"}`)
		assertStatus(t, status, http.StatusOK)
		status, body = requestJSON(t, http.MethodGet, endpoint, "")
		assertStatus(t, status, http.StatusOK)
		if err := json.Unmarshal([]byte(body), &user); err != nil || user.Name != "Bob" || user.Email != successEmail || user.ID != createdUser.ID {
			t.Fatalf("updated user: %#v, error=%v, body=%s", user, err, body)
		}
		status, _ = requestJSON(t, http.MethodDelete, endpoint, "")
		assertStatus(t, status, http.StatusOK)
		assertUserCount(t, db, successEmail, 0)
		for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
			status, _ = requestJSON(t, method, endpoint, `{"name":"Bob"}`)
			assertStatus(t, status, http.StatusNotFound)
		}
	})
}

func requiredEnv(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Skipf("%s is required for external E2E", name)
	}
	return value
}

func openDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	return db
}

func userJSON(name string, email string) string {
	return fmt.Sprintf(`{"name":%q,"email":%q}`, name, email)
}

func postJSON(t *testing.T, endpoint string, body string) (int, string) {
	t.Helper()
	return requestJSON(t, http.MethodPost, endpoint, body)
}

func requestJSON(t *testing.T, method, endpoint, body string) (int, string) {
	t.Helper()
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(method, endpoint, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(responseBody)
}

func assertStatus(t *testing.T, got int, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
}

func assertUserCount(t *testing.T, db *sql.DB, email string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("user count for %s = %d, want %d", email, got, want)
	}
}
