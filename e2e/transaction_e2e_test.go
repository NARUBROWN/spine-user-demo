//go:build e2e

package e2e

import (
	"bytes"
	"database/sql"
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
	controllerErrorEmail := "controller-error-" + suffix + "@example.com"
	responseErrorEmail := "response-error-" + suffix + "@example.com"
	for _, email := range []string{successEmail, controllerErrorEmail, responseErrorEmail} {
		email := email
		t.Cleanup(func() { _, _ = db.Exec("DELETE FROM users WHERE email = ?", email) })
	}

	t.Run("successful response commits", func(t *testing.T) {
		status, _ := postJSON(t, baseURL+"/users", userJSON("Alice", successEmail))
		assertStatus(t, status, http.StatusOK)
		assertUserCount(t, db, successEmail, 1)
	})

	t.Run("invalid input returns 400", func(t *testing.T) {
		status, _ := postJSON(t, baseURL+"/users", userJSON("", "not-an-email"))
		assertStatus(t, status, http.StatusBadRequest)
	})

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
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(endpoint, "application/json", bytes.NewBufferString(body))
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
