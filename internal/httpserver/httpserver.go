package httpserver

import (
	"net/http"
	"strconv"
	"strings"

	"rate-limiter/internal/limiter"
)

// Authorize answers a gateway "should this request proceed?" check.
// It expects the original request's path in X-Original-Path and the
// original client's IP in X-Forwarded-For, both set by the gateway.
func Authorize(rules map[string]limiter.Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.Header.Get("X-Original-Path")
		if path == "" {
			http.Error(w, "missing X-Original-Path header", http.StatusBadRequest)
			return
		}

		l, ok := rules[path]
		if !ok {
			http.Error(w, "no rate limit rule for this path", http.StatusForbidden)
			return
		}

		ip, ok := clientIP(r)
		if !ok {
			http.Error(w, "missing client IP", http.StatusBadRequest)
			return
		}

		result, err := l.Allow(r.Context(), ip)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))

		if !result.Allowed {
			retrySecs := int(result.RetryAfter.Seconds()) + 1 // round up to the next second
			w.Header().Set("Retry-After", strconv.Itoa(retrySecs))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// clientIP extracts the real client's address from X-Forwarded-For, set by
// the gateway. ok is false if the header is missing or empty.
// r.RemoteAddr is not a usable fallback here, since every request to Authorize
// comes from the gateway itself, not the original client.
func clientIP(r *http.Request) (string, bool) {
	fwd := r.Header.Get("X-Forwarded-For")
	if fwd == "" {
		return "", false
	}

	return strings.TrimSpace(strings.Split(fwd, ",")[0]), true
}
