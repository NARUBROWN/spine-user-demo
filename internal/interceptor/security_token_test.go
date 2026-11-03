package interceptor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	platform "github.com/NARUBROWN/spine-echo-platform"
	"github.com/NARUBROWN/spine/core"
	"github.com/NARUBROWN/spine/pkg/boot"
	"github.com/NARUBROWN/spine/pkg/httpx"
	"github.com/NARUBROWN/spine/pkg/route"
	"github.com/NARUBROWN/spine/pkg/spine"
)

type securityRequest struct {
	Name string `json:"name"`
}

type securityResponse struct {
	Marker string `json:"marker"`
}

type readTrackingBody struct {
	reader io.Reader
	reads  atomic.Int64
}

func (body *readTrackingBody) Read(p []byte) (int, error) {
	body.reads.Add(1)
	return body.reader.Read(p)
}

func (*readTrackingBody) Close() error { return nil }

type securityController struct {
	calls *atomic.Int64
}

func (c *securityController) Create(_ *securityRequest, ctx spine.Ctx) httpx.Response[securityResponse] {
	c.calls.Add(1)
	marker, _ := ctx.Get("security-test-marker")
	value, _ := marker.(string)
	return httpx.Response[securityResponse]{Body: securityResponse{Marker: value}}
}

type securityMarkerInterceptor struct{}

func (*securityMarkerInterceptor) PreHandle(ctx core.ExecutionContext, _ core.HandlerMeta) error {
	ctx.Set("security-test-marker", ctx.Header("X-Test-Marker"))
	return nil
}
func (*securityMarkerInterceptor) PostHandle(core.ExecutionContext, core.HandlerMeta) {}
func (*securityMarkerInterceptor) BeforeResponse(core.ExecutionContext, core.HandlerMeta, error) error {
	return nil
}
func (*securityMarkerInterceptor) AfterCompletion(core.ExecutionContext, core.HandlerMeta, error) {}

func TestSecurityPipeline(t *testing.T) {
	var calls atomic.Int64
	app := platform.New()
	app.Constructor(func() *securityController { return &securityController{calls: &calls} })
	app.Route("POST", "/protected", (*securityController).Create, route.WithInterceptors(
		&SecurityTokenInterceptor{Token: "test-token"}, &securityMarkerInterceptor{},
	))
	prepared, err := app.Prepare(context.Background(), boot.Options{HTTP: &boot.HTTPOptions{MaxBodyBytes: 64}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := prepared.Close(ctx); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	handler := prepared.Handler()

	tests := []struct {
		name, authorization, body string
		status                    int
	}{
		{"missing token and malformed body", "", "{", http.StatusUnauthorized},
		{"wrong token and malformed body", "Bearer wrong", "{", http.StatusUnauthorized},
		{"invalid scheme", "Basic test-token", `{"name":"alice"}`, http.StatusUnauthorized},
		{"valid token and malformed body", "Bearer test-token", "{", http.StatusBadRequest},
		{"valid token and oversized body", "Bearer test-token", `{"name":"` + strings.Repeat("x", 65) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/protected", strings.NewReader(tt.body))
			body := &readTrackingBody{reader: strings.NewReader(tt.body)}
			request.Body = body
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", tt.authorization)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d, body=%s", response.Code, tt.status, response.Body.String())
			}
			if calls.Load() != 0 {
				t.Fatalf("controller ran after rejected input: calls = %d", calls.Load())
			}
			if tt.status == http.StatusUnauthorized && body.reads.Load() != 0 {
				t.Fatalf("body was read before authentication: reads = %d", body.reads.Load())
			}
			if tt.status != http.StatusUnauthorized && body.reads.Load() == 0 {
				t.Fatal("authenticated request body was not read")
			}
		})
	}

	t.Run("concurrent request context stays isolated", func(t *testing.T) {
		const requests = 32
		failures := make(chan error, requests)
		var wg sync.WaitGroup
		for i := 0; i < requests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				marker := fmt.Sprintf("request-%d", index)
				request := httptest.NewRequest(http.MethodPost, "/protected", strings.NewReader(`{"name":"alice"}`))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", "Bearer test-token")
				request.Header.Set("X-Test-Marker", marker)
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				var result securityResponse
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || response.Code != http.StatusOK || result.Marker != marker {
					failures <- fmt.Errorf("marker=%s status=%d result=%+v error=%v", marker, response.Code, result, err)
				}
			}(i)
		}
		wg.Wait()
		close(failures)
		for err := range failures {
			t.Error(err)
		}
		if got := calls.Load(); got != requests {
			t.Fatalf("controller calls = %d, want %d", got, requests)
		}
	})
}
