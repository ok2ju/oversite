# Backend — Go

## Coding Conventions

- **Router**: chi. Group routes by resource. Middleware applied per-group.
- **SQL**: sqlc generates all DB access code. Write SQL in `queries/*.sql`, run `make sqlc`.
- **Errors**: Return sentinel errors from services (`ErrNotFound`, `ErrForbidden`). Handlers map to HTTP status codes.
- **Logging**: `slog` (stdlib). Structured JSON. Include request ID.
- **Config**: Environment variables. Loaded via `internal/config` into a typed struct.

## Testing

TDD (Red-Green-Refactor). All conventions below apply.

- **Unit tests**: Table-driven tests. Run: `go test -race ./...` (always use `-race`)
- **Integration tests**: Use `testcontainers` for real DB. Build tag: `//go:build integration`. Run: `go test -race -tags integration ./...`
- **Golden file tests**: For parser output. Use `-update` flag to regenerate.
- **Mocking**: Interface-based DI. Mock interfaces: `Store`, `S3Client`, `SessionStore`, `JobQueue`, `FaceitAPI`.

## sqlc Workflow

1. Write SQL in `queries/*.sql`
2. Run `make sqlc` to regenerate Go code in `internal/store/`
3. Never edit `*.sql.go` files directly (blocked by pre-commit hook)
