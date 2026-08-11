package main

import (
	"log"
	"net/http"

	"rate-limiter/internal/config"
	"rate-limiter/internal/httpserver"
	"rate-limiter/internal/limiter"
)

func main() {
	rules := config.Default()
	limiters := buildLimiters(rules)

	mux := http.NewServeMux()
	mux.Handle("/authorize", httpserver.Authorize(limiters))
	mux.HandleFunc("/healthz", healthHandler)

	const addr = ":8080"
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))

}

// buildLimiters creates a map of path -> Limiter from the given rules.
// It also starts a janitor goroutine for each limiter to clean up old buckets.
func buildLimiters(rules map[string]config.Rule) map[string]limiter.Limiter {
	limiters := make(map[string]limiter.Limiter, len(rules))
	for path, rule := range rules {
		tb := limiter.NewTokenBucketLimiter(rule.RequestsPerSec, rule.Burst)
		limiters[path] = tb
	}

	return limiters
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok\n"))
}
