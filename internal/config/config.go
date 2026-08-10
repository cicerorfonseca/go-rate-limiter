// Package config holds the list of rate-limited endpoints and their configuration.
package config

// Rule defines the rate limit applied to a single endpoint.
type Rule struct {
	// RequestsPerSec is the sustained rate allowed per client.
	RequestsPerSec float64
	// Burst is the maximum number of requests a single client can send
	// back-to-back before being throttled.
	Burst float64
}

// Default returns the built-in map of rate-limited endpoints keyed by request path.
// This is hardcoded for now, on purpose. Swapping this for a YAML/JSON file
// or env-driven loader later requires only changing this function.
func Default() map[string]Rule {
	return map[string]Rule{
		"/api/orders": {RequestsPerSec: 5, Burst: 10},
		"/api/login":  {RequestsPerSec: 1, Burst: 3},
	}
}
