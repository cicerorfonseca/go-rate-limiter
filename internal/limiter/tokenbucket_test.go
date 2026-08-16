package limiter

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucketLimiter_Allow(t *testing.T) {
	l := NewTokenBucketLimiter()
	ctx := context.Background()

	result, err := l.Allow(ctx, "client-a", 1, 5)
	if err != nil {
		t.Fatalf("Allow returned unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("expected first request on a frech bucket to be allowed")
	}

	// Drain the bucket instantly instead of looping Allow, so this test doesn't
	// depend on exactly how many tokens a fresh bucket starts with.
	l.buckets["client-a"].tokens = 0

	result, err = l.Allow(ctx, "client-a", 1, 5)
	if err != nil {
		t.Fatalf("Allow returned unexpected error: %v", err)
	}
	if result.Allowed {
		t.Fatalf("expected request to be denied when bucket is empty")
	}
	if result.RetryAfter <= 0 {
		t.Fatalf("expected a positive RetryAfter, got %v", result.RetryAfter)
	}

	// Backdate lastRefill by 3 seconds. At 1 token/sec that's 3 tokens of refill
	// with no real sleeping in the test
	l.buckets["client-a"].lastRefill = time.Now().Add(-3 * time.Second)

	result, err = l.Allow(ctx, "client-a", 1, 5)
	if err != nil {
		t.Fatalf("Allow returned unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("expected request to be allowed after refill")
	}
}

func TestCleanUp(t *testing.T) {
	l := NewTokenBucketLimiter()
	ctx := context.Background()

	l.Allow(ctx, "stale", 1, 5)
	l.Allow(ctx, "fresh", 1, 5)

	l.buckets["stale"].lastRefill = time.Now().Add(-1 * time.Hour)

	l.Cleanup(time.Now().Add(-10 * time.Minute))

	if _, exists := l.buckets["stale"]; exists {
		t.Errorf("expected stale bucket to be removed")
	}
	if _, exists := l.buckets["fresh"]; !exists {
		t.Errorf("expected fresh bucket to remain")
	}
}
