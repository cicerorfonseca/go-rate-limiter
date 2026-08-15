package main

import (
	"log"
	"net/http"

	"rate-limiter/internal/config"
	"rate-limiter/internal/httpserver"
	"rate-limiter/internal/limiter"
)

func main() {
	cfg := config.New()
	l := limiter.NewTokenBucketLimiter()

	mux := http.NewServeMux()
	mux.Handle("/authorize", httpserver.Authorize(cfg, l))
	mux.HandleFunc("/healthz", healthHandler)

	const addr = ":8080"
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))

}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok\n"))
}
