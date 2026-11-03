//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestProtectedUserSecurityBoundary(t *testing.T) {
	baseURL := requiredEnv(t, "SPINE_DEMO_E2E_BASE_URL")
	db := openDB(t, requiredEnv(t, "SPINE_DEMO_MYSQL_DSN"))
	token := requiredEnv(t, "SPINE_DEMO_E2E_SECURITY_TOKEN")
	email := fmt.Sprintf("security-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() { _, _ = db.Exec("DELETE FROM users WHERE email = ?", email) })

	request := func(authorization, body string) (int, string) {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, baseURL+"/scenarios/security/users", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, string(responseBody)
	}

	for _, tt := range []struct {
		name, authorization, body string
		want                      int
	}{
		{"missing token", "", userJSON("Alice", email), http.StatusUnauthorized},
		{"wrong token", "Bearer incorrect", userJSON("Alice", email), http.StatusUnauthorized},
		{"authentication precedes body parsing", "", "{", http.StatusUnauthorized},
		{"authenticated malformed body", "Bearer " + token, "{", http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := request(tt.authorization, tt.body)
			assertStatus(t, status, tt.want)
			assertUserCount(t, db, email, 0)
		})
	}

	t.Run("valid token permits one committed insert", func(t *testing.T) {
		status, _ := request("Bearer "+token, userJSON("Alice", email))
		assertStatus(t, status, http.StatusOK)
		assertUserCount(t, db, email, 1)
	})
}
