# CLAUDE.md

This file gives Claude Code context for working in this repository.
Keep it short and factual: commands, layout, conventions, and rules.

## Project overview

Nateway is an API gateway written in Go. Requests pass through a middleware
chain (rate limiting -> routing -> auth) before being proxied to a backend.

<!-- TODO: describe goals, scope, and what is intentionally out of scope. -->

## Commands

```bash
go build ./...                       # build
go vet ./...                         # static checks
go test ./...                        # unit tests
go test -bench=. ./internal/domain   # benchmarks (needs key.pem at repo root)
go run ./cmd                         # run the gateway (needs Redis + cert.pem)
go run echo.go :8080                 # sample echo backend
k6 run test.js                       # load test
```

<!-- TODO: add setup steps (Redis, generating key.pem / cert.pem, docker-compose). -->

## Layout

- `cmd/main.go` - entry point; loads `gateway.yaml`, starts the watcher and server
- `internal/domain/gateway.go` - config types, `LoadGateway`, hot-reload watcher
- `internal/domain/middleware.go` - `Middleware`, `Chain`, `ProxyHandler`
- `internal/domain/limiter.go` - rate limiter middleware
- `internal/domain/redis_token_bucket.go`, `token_bucket.lua` - Redis token bucket
- `internal/domain/router.go`, `trie.go`, `trie_router.go` - routing
- `internal/domain/auth.go` - JWT (RSA) auth middleware
- `gateway.yaml` - example config

## Conventions

<!-- TODO: code style, error handling, logging, naming, commit message format. -->

## Testing

<!-- TODO: testing approach (TDD, table-driven tests, httptest, miniredis, etc.). -->

## Rules for Claude

<!-- TODO: what Claude should and should not do (e.g. "suggest tests, don't write them"). -->
