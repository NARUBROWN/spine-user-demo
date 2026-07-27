package interceptor

import "testing"

func TestRequestID(t *testing.T) {
	if got := requestID("client-request-123"); got != "client-request-123" {
		t.Fatalf("requestID() = %q, want client request ID", got)
	}
	if got := requestID("bad\r\nid"); got == "bad\r\nid" || got == "" {
		t.Fatalf("requestID() must replace an unsafe value, got %q", got)
	}
	if got := requestID("bad\tid"); got == "bad\tid" || got == "" {
		t.Fatalf("requestID() must replace a control character, got %q", got)
	}
}
