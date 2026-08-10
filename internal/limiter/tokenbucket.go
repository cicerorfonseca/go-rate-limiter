package limiter

import (
	"context"
	"sync"
	"time"
)

// bucket holds the state of a token bucket for rate limiting.
type bucket struct {
	tokens     float64
	lastRefill time.Time
}

// TokenBucketLimiter is an in-memory, per-key token bucket rate limiter.
// Each key gets its own token bucket that fills at 'rate' tokens/second
// up to a 'burst' cap of tokens
type TokenBucketLimiter struct {
	rate  float64 // tokens per second
	burst float64 // maximum number of tokens in the bucket

	mu      sync.Mutex
	buckets map[string]*bucket
}

// NewTokenBucketLimiter creates a limiter that allows 'rate' tokens per second
// per key on average, with bursts up to 'burst' tokens
func NewTokenBucketLimiter(rate, burst float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:    rate,
		burst:   burst,
		buckets: make(map[string]*bucket),
	}
}

// Allow implements Limiter
func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (Result, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, lastRefill: now}
		l.buckets[key] = b
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = min(l.burst, b.tokens+elapsed*l.rate)
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return Result{Allowed: true, Remaining: int(b.tokens)}, nil
	}

	deficit := 1 - b.tokens
	wait := time.Duration(deficit / l.rate * float64(time.Second))
	return Result{Allowed: false, Remaining: int(b.tokens), RetryAfter: wait}, nil
}
