package gateway

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// BearerTokenAuth returns middleware that validates Bearer tokens.
// If allowedTokens is empty, auth is disabled (all requests pass through).
func BearerTokenAuth(allowedTokens []string) func(http.Handler) http.Handler {
	tokenSet := make(map[string]struct{}, len(allowedTokens))
	for _, tok := range allowedTokens {
		tok = strings.TrimSpace(tok)
		if tok != "" {
			tokenSet[tok] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// /readyz is always public.
			if r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			// No tokens configured → auth disabled.
			if len(tokenSet) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="korus"`)
				writeError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="korus"`)
				writeError(w, http.StatusUnauthorized, "Authorization must use Bearer scheme")
				return
			}

			token := strings.TrimSpace(auth[len(prefix):])
			if token == "" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="korus"`)
				writeError(w, http.StatusUnauthorized, "empty bearer token")
				return
			}

			if _, ok := tokenSet[token]; !ok {
				writeError(w, http.StatusForbidden, "invalid bearer token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter is a per-key token bucket rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rate     float64 // tokens per second
	capacity int
	now      func() time.Time
}

type tokenBucket struct {
	tokens   float64
	lastFill time.Time
}

// NewRateLimiter creates a rate limiter with the given requests-per-second
// and burst capacity. If rps <= 0, returns nil (no limiting).
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	if rps <= 0 || burst <= 0 {
		return nil
	}
	return &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     rps,
		capacity: burst,
	}
}

// Allow checks whether a request from the given key is allowed.
func (rl *RateLimiter) Allow(key string) bool {
	if rl == nil {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.nowTime()
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &tokenBucket{tokens: float64(rl.capacity), lastFill: now}
		rl.buckets[key] = bucket
	}

	// Refill tokens based on elapsed time.
	elapsed := now.Sub(bucket.lastFill).Seconds()
	bucket.tokens += elapsed * rl.rate
	if bucket.tokens > float64(rl.capacity) {
		bucket.tokens = float64(rl.capacity)
	}
	bucket.lastFill = now

	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

func (rl *RateLimiter) nowTime() time.Time {
	if rl.now != nil {
		return rl.now()
	}
	return time.Now()
}

// RateLimitMiddleware returns middleware that applies per-IP rate limiting.
// If limiter is nil, all requests pass through.
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			// /readyz is exempt.
			if r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			key := clientIP(r)
			if !limiter.Allow(key) {
				w.Header().Set("Retry-After", "1")
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	// X-Forwarded-For takes precedence (for proxied setups).
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	// X-Real-IP is common for nginx.
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Fall back to RemoteAddr.
	host, _, found := strings.Cut(r.RemoteAddr, ":")
	if !found || host == "" {
		return r.RemoteAddr
	}
	return host
}
