package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"rate-limiter/internal/config"
	"rate-limiter/internal/limiter"
	"testing"
	"time"
)

type fakeResolver struct {
	rule config.Rule
	ok   bool
}

func (f fakeResolver) Resolve(tenant, path string) (config.Rule, bool) {
	return f.rule, f.ok
}

type fakeLimiter struct {
	result limiter.Result
	err    error
}

func (f fakeLimiter) Allow(ctx context.Context, key string, rate, burst float64) (limiter.Result, error) {
	return f.result, f.err
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		wantIP  string
		wantOK  bool
	}{
		{
			name:    "header present",
			headers: map[string]string{"X-Forwarded-For": "1.2.3.4"},
			wantIP:  "1.2.3.4",
			wantOK:  true,
		},
		{
			name:    "header missing",
			headers: map[string]string{},
			wantIP:  "",
			wantOK:  false,
		},
		{
			name:    "multiple IPs, takes the first",
			headers: map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"},
			wantIP:  "1.2.3.4",
			wantOK:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/authorize", nil)
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}

			gotIP, gotOK := clientIP(r)
			if gotIP != tt.wantIP || gotOK != tt.wantOK {
				t.Errorf("clientIP() = (%q, %v), want (%q, %v)", gotIP, gotOK, tt.wantIP, tt.wantOK)
			}
		})
	}
}

func TestAuthorize(t *testing.T) {
	tests := []struct {
		name           string
		originalPath   string
		forwardedFor   string
		resolverRule   config.Rule
		resolverOK     bool
		limiterResult  limiter.Result
		limiterErr     error
		wantStatus     int
		wantRemaining  string // "" means don't check
		wantRetryAfter string // "" means don't check
	}{
		{
			name:         "missing X-Original-Path",
			forwardedFor: "1.2.3.4",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "no rule for path",
			originalPath: "/api/unknown",
			forwardedFor: "1.2.3.4",
			resolverOK:   false,
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "missing X-Forwarded-For",
			originalPath: "/api/orders",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:          "allowed",
			originalPath:  "/api/orders",
			forwardedFor:  "1.2.3.4",
			resolverRule:  config.Rule{RequestsPerSec: 5, Burst: 10},
			resolverOK:    true,
			limiterResult: limiter.Result{Allowed: true, Remaining: 5},
			wantStatus:    http.StatusOK,
			wantRemaining: "5",
		},
		{
			name:           "denied",
			originalPath:   "/api/orders",
			forwardedFor:   "1.2.3.4",
			resolverRule:   config.Rule{RequestsPerSec: 5, Burst: 10},
			resolverOK:     true,
			limiterResult:  limiter.Result{Allowed: false, Remaining: 0, RetryAfter: 3 * time.Second},
			wantStatus:     http.StatusTooManyRequests,
			wantRemaining:  "0",
			wantRetryAfter: "4", // 3s rounded up to the next second
		},
		{
			name:         "limiter error",
			originalPath: "/api/orders",
			forwardedFor: "1.2.3.4",
			resolverRule: config.Rule{RequestsPerSec: 5, Burst: 10},
			resolverOK:   true,
			limiterErr:   errors.New("boom"),
			wantStatus:   http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/authorize", nil)
			if tt.originalPath != "" {
				r.Header.Set("X-Original-Path", tt.originalPath)
			}
			if tt.forwardedFor != "" {
				r.Header.Set("X-Forwarded-For", tt.forwardedFor)
			}

			resolver := fakeResolver{rule: tt.resolverRule, ok: tt.resolverOK}
			l := fakeLimiter{result: tt.limiterResult, err: tt.limiterErr}

			w := httptest.NewRecorder()
			Authorize(resolver, l).ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if tt.wantRemaining != "" && w.Header().Get("X-RateLimit-Remaining") != tt.wantRemaining {
				t.Errorf("X-RateLimit-Remaining = %q, want %q", w.Header().Get("X-RateLimit-Remaining"), tt.wantRemaining)
			}
			if tt.wantRetryAfter != "" && w.Header().Get("Retry-After") != tt.wantRetryAfter {
				t.Errorf("Retry-After = %q, want %q", w.Header().Get("Retry-After"), tt.wantRetryAfter)
			}
		})
	}
}
