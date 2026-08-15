// Package limiter defines the rate-limiting abstraction used by the HTTP
// layer, plus concrete implementations of it.
package limiter

import (
	"context"
	"time"
)

// Result describes the outcome of a single rate limit check.
type Result struct {
	Allowed bool
	// The remaining amount of tokens in the bucket after this request, if Allowed is true.
	Remaining int
	// How long the caller should wait before trying again.
	// Only meaningful when Allowed is false.
	RetryAfter time.Duration
}

// Limiter decides whether a request identified by key is allowed to proceed right now.
// key is caller-defined, scoped per endpoint by the caller.
type Limiter interface {
	Allow(ctx context.Context, key string, rate, burst float64) (Result, error)
}
