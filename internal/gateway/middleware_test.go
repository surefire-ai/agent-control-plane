package gateway

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBearerTokenAuthRejectsMissingHeader(t *testing.T) {
	handler := BearerTokenAuth([]string{"secret-token"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("expected WWW-Authenticate header on 401")
	}
}

func TestBearerTokenAuthRejectsInvalidToken(t *testing.T) {
	handler := BearerTokenAuth([]string{"secret-token"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer wrong-token")
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestBearerTokenAuthRejectsNonBearerScheme(t *testing.T) {
	handler := BearerTokenAuth([]string{"secret-token"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestBearerTokenAuthAcceptsValidToken(t *testing.T) {
	handler := BearerTokenAuth([]string{"secret-token"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer secret-token")
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestBearerTokenAuthDisabledWhenNoTokensConfigured(t *testing.T) {
	handler := BearerTokenAuth(nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{}`))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 when auth disabled, got %d", recorder.Code)
	}
}

func TestBearerTokenAuthReadyzIsAlwaysPublic(t *testing.T) {
	handler := BearerTokenAuth([]string{"secret-token"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected /readyz to be public, got %d", recorder.Code)
	}
}

func TestRateLimiterAllowsWithinBurst(t *testing.T) {
	rl := NewRateLimiter(10, 3)
	rl.now = func() time.Time { return time.Unix(1000, 0) }

	if !rl.Allow("client-a") {
		t.Fatal("first request should be allowed")
	}
	if !rl.Allow("client-a") {
		t.Fatal("second request should be allowed")
	}
	if !rl.Allow("client-a") {
		t.Fatal("third request should be allowed (burst=3)")
	}
}

func TestRateLimiterRejectsOverBurst(t *testing.T) {
	now := time.Unix(1000, 0)
	rl := NewRateLimiter(1, 2) // 1 rps, burst 2
	rl.now = func() time.Time { return now }

	if !rl.Allow("client-a") {
		t.Fatal("first request should be allowed")
	}
	if !rl.Allow("client-a") {
		t.Fatal("second request should be allowed")
	}
	if rl.Allow("client-a") {
		t.Fatal("third request should be rejected (burst exhausted)")
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	now := time.Unix(1000, 0)
	rl := NewRateLimiter(1, 2) // 1 rps, burst 2
	rl.now = func() time.Time { return now }

	// Exhaust burst.
	rl.Allow("client-a")
	rl.Allow("client-a")

	// Advance 2 seconds → should refill 2 tokens.
	now = time.Unix(1002, 0)
	if !rl.Allow("client-a") {
		t.Fatal("should allow after refill period")
	}
}

func TestRateLimiterSeparatesClients(t *testing.T) {
	rl := NewRateLimiter(1, 1) // 1 rps, burst 1
	rl.now = func() time.Time { return time.Unix(1000, 0) }

	if !rl.Allow("client-a") {
		t.Fatal("client-a first request should be allowed")
	}
	if !rl.Allow("client-b") {
		t.Fatal("client-b first request should be allowed (separate bucket)")
	}
	if rl.Allow("client-a") {
		t.Fatal("client-a second request should be rejected")
	}
}

func TestRateLimiterNilIsPermissive(t *testing.T) {
	var rl *RateLimiter // nil
	if !rl.Allow("anyone") {
		t.Fatal("nil rate limiter should allow all")
	}
}

func TestRateLimitMiddlewareRejectsOverLimit(t *testing.T) {
	limiter := NewRateLimiter(1, 1)
	limiter.now = func() time.Time { return time.Unix(1000, 0) }

	handler := RateLimitMiddleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request — allowed.
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{}`))
	req1.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rec1.Code)
	}

	// Second request — rate limited.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{}`))
	req2.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rec2.Code)
	}
	if rec2.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429")
	}
}

func TestRateLimitMiddlewareReadyzIsExempt(t *testing.T) {
	// Very restrictive limiter: 0.1 rps, burst 1
	limiter := NewRateLimiter(0.1, 1)
	limiter.now = func() time.Time { return time.Unix(1000, 0) }

	handler := RateLimitMiddleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the bucket.
	rec0 := httptest.NewRecorder()
	req0 := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{}`))
	req0.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(rec0, req0)
	if rec0.Code != http.StatusOK {
		t.Fatalf("setup: expected 200, got %d", rec0.Code)
	}

	// Confirm next POST is rate-limited.
	recBlocked := httptest.NewRecorder()
	reqBlocked := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{}`))
	reqBlocked.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(recBlocked, reqBlocked)
	if recBlocked.Code != http.StatusTooManyRequests {
		t.Fatalf("setup: expected 429, got %d", recBlocked.Code)
	}

	// /readyz should still be exempt.
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	request.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected /readyz to be exempt from rate limiting, got %d", recorder.Code)
	}
}

func TestClientIPFromRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	ip := clientIP(req)
	if ip != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1, got %q", ip)
	}
}

func TestClientIPFromXForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18")
	ip := clientIP(req)
	if ip != "203.0.113.50" {
		t.Fatalf("expected 203.0.113.50, got %q", ip)
	}
}

func TestClientIPFromXRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-Ip", "198.51.100.77")
	ip := clientIP(req)
	if ip != "198.51.100.77" {
		t.Fatalf("expected 198.51.100.77, got %q", ip)
	}
}
