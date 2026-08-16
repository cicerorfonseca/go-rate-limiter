# rate-limiter

A multi-tenant rate limiter written as a learning project in Go. It's built as a **check-only authorization sidecar**: an API gateway (nginx `auth_request`, Envoy `ext_authz`, or similar) calls it before forwarding a request to the real backend, and it answers with a single HTTP status code. The gateway decides what to do from there. The service never proxies or serves the real request itself.

Currently in-memory only (token bucket, per-process state). A Redis-backed implementation is planned, see [Roadmap](#roadmap).

## How it works

**Algorithm:** token bucket. Each `(tenant, client IP, endpoint)` combination gets its own bucket that holds up to `burst` tokens and refills at `rate` tokens/second. Every allowed request spends one token. Refilling is computed lazily, from elapsed time on each check, rather than with a per-key background goroutine.

**Multi-tenancy:** rate limits are resolved per request from `internal/config`, which holds a default rule per endpoint plus optional per-tenant overrides. A request with no recognized tenant (missing header, or an unknown tenant ID) falls back to the default rule for that endpoint rather than being rejected.

**Keying:** buckets are stored in one flat map, keyed by a composite string `tenant:ip:path`. This shape was chosen deliberately to match how the eventual Redis implementation will store state, Redis TTLs apply per top-level key, so a flat keyspace (rather than nested structures) is what a Redis-backed limiter needs, and keeping the in-memory version in the same shape now means the swap won't require reworking the HTTP layer.

**Idle cleanup:** a background janitor removes buckets that haven't been touched in 10 minutes, so memory doesn't grow without bound as new tenant/client/endpoint combinations show up over the life of the process.

## API

### `POST /authorize` (no body is read)

`POST`, not `GET`, on purpose: despite being a read-like "can this proceed?" check, `Allow` mutates state on every call (it spends a token from the bucket), it is not idempotent, so it shouldn't carry GET's safe/cacheable/auto-retryable semantics. Any other method gets a `405` from the router before the handler even runs.

Request headers:

| Header            | Required | Meaning                                                                  |
| ----------------- | -------- | ------------------------------------------------------------------------ |
| `X-Original-Path` | yes      | The path the real client is requesting, e.g. `/api/orders`               |
| `X-Forwarded-For` | yes      | The real client's IP address                                             |
| `X-Tenant-Id`     | no       | Tenant identifier; falls back to default rules if absent or unrecognized |

Response:

| Status | Meaning                                                  |
| ------ | -------------------------------------------------------- |
| `200`  | Allowed. Proceed to the real backend.                    |
| `429`  | Rate limit exceeded. `Retry-After` header set (seconds). |
| `403`  | No rate limit rule configured for this path at all.      |
| `400`  | Missing `X-Original-Path` or `X-Forwarded-For`.          |
| `405`  | Wrong HTTP method — only `POST` is accepted.             |
| `500`  | Internal limiter error.                                  |

Every successful check (`200` or `429`) also sets `X-RateLimit-Remaining`, the number of requests this key can still make right now.

### `GET /healthz`

Always returns `200`, unauthenticated, unthrottled. For container/orchestrator liveness checks.

## Project structure

```
cmd/server/           entrypoint: wires everything together, starts the HTTP server
internal/limiter/     the Limiter interface + in-memory token bucket implementation
internal/config/      endpoint rate limit rules, default + per-tenant overrides
internal/httpserver/  the /authorize handler (net/http middleware-style)
```

`internal/limiter`'s `Limiter` interface is the seam a future `RedisLimiter` will implement, nothing outside that package needs to change when it lands.

## Running locally

```bash
go run ./cmd/server
```

```bash
curl -i \
  -H "X-Original-Path: /api/orders" \
  -H "X-Forwarded-For: 1.2.3.4" \
  -H "X-Tenant-Id: tenant1" \
  http://localhost:8080/authorize
```

## Running with Docker

```bash
docker build -t rate-limiter .
docker run --rm -p 8080:8080 rate-limiter
```

## Testing

```bash
go test ./...
```

## Roadmap

- [ ] `RedisLimiter` implementing the `Limiter` interface, for shared state across multiple instances
- [ ] `docker-compose.yml` running the app alongside a Redis container
- [ ] Config loaded from a file/env instead of hardcoded in `internal/config`
- [ ] Collision-safe key encoding. The current `tenant:ip:path` key can theoretically collide when an IPv6 address (which contains colons) lands at a field boundary; low practical impact today, noted in `internal/httpserver/httpserver.go`
