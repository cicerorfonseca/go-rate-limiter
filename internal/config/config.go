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

// Config defines the config rules for bucket creation.
// When a new request comes in, the bucket will be created based on the
// default values or overriden if tenant available
type Config struct {
	defaults        map[string]Rule
	tenantOverrides map[string]map[string]Rule
}

type Resolver interface {
	Resolve(tenant, path string) (Rule, bool)
}

// New creates a reusable instance of Config, avoiding the config loading
// for every incoming request.
func New() *Config {
	return &Config{
		defaults: map[string]Rule{
			"/api/orders": {RequestsPerSec: 10, Burst: 20},
			"/api/login":  {RequestsPerSec: 5, Burst: 10},
			"/api/users":  {RequestsPerSec: 20, Burst: 40},
		},
		tenantOverrides: map[string]map[string]Rule{
			"tenant1": {
				"/api/orders": {RequestsPerSec: 50, Burst: 100},
				"/api/login":  {RequestsPerSec: 10, Burst: 20},
			},
			"tenant2": {
				"/api/orders": {RequestsPerSec: 25, Burst: 50},
				"/api/login":  {RequestsPerSec: 7, Burst: 14},
				"/api/users":  {RequestsPerSec: 60, Burst: 120},
			},
		},
	}
}

// Resolve finds and returns the configuration for bucket initialization
// It tries to find the tenant-specific config, otherwise it fallsback to the defaults
func (c *Config) Resolve(tenant, path string) (Rule, bool) {
	if tenantRules, exists := c.tenantOverrides[tenant]; exists {
		if override, exists := tenantRules[path]; exists {
			return override, true
		}
	}

	rule, ok := c.defaults[path]
	return rule, ok
}
