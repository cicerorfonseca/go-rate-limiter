package main

import (
	"log"
	"net/http"
	"time"

	"rate-limiter/internal/config"
	"rate-limiter/internal/httpserver"
	"rate-limiter/internal/limiter"
)

func main() {
	cfg := config.New()
	l := limiter.NewTokenBucketLimiter()
	go janitor(l)

	mux := http.NewServeMux()
	mux.Handle("/authorize", httpserver.Authorize(cfg, l))
	mux.HandleFunc("/healthz", healthHandler)

	const addr = ":8080"
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// janitor periodically removes buckets idle for more than 10 minutes
// so memory doesn't grow forever as new tenant/client/endpoint combinations
// show up over the processe's lifetime
func janitor(l *limiter.TokenBucketLimiter) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.Cleanup(time.Now().Add(-10 * time.Minute))
	}
}

// healthHandler acts as healthcheck endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok\n"))
}
