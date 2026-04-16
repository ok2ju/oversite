# Oversite

CS2 2D demo viewer and analytics platform for Faceit players. Single-binary Wails desktop app -- import local demos, watch top-down playback, generate heatmaps, plan strategies, and track Faceit stats.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Runtime | Wails v2 (Go backend + system WebView frontend) |
| Frontend | Vite + React, TypeScript, PixiJS v8, shadcn/ui, Tailwind CSS, Zustand, TanStack Query v5, react-router-dom v6 |
| Backend | Go 1.26+, Wails bindings (no HTTP server) |
| Demo Parsing | markus-wa/demoinfocs-golang v5 |
| Database | SQLite (modernc.org/sqlite, pure Go, WAL mode) |
| SQL | sqlc (type-safe generated Go, SQLite dialect) |
| Migrations | golang-migrate (embedded via `//go:embed`) |
| Auth | Faceit OAuth 2.0 + PKCE (loopback redirect), OS keychain (zalando/go-keyring) |
| Packaging | Single native binary per platform (macOS, Windows, Linux) |

## Project Structure

All desktop Go code lives at the **root module level** (`go.mod` = `github.com/ok2ju/oversite`).

```
oversite/
├── main.go                         # Wails entry point
├── app.go                          # App struct (Startup/Shutdown, Wails bindings)
├── types.go                        # Domain types (exposed via Wails bindings)
├── go.mod                          # Root Go module
├── wails.json                      # Wails project config
├── internal/
│   ├── auth/                       # Faceit OAuth, PKCE, keyring, auth service
│   ├── database/                   # SQLite connection, migration runner
│   ├── demo/                       # Demo parser, importer, stats, events, rounds
│   ├── store/                      # sqlc generated code (SQLite)
│   └── testutil/                   # Shared test helpers
├── migrations/                     # SQLite migration files (embedded in binary)
├── queries/                        # sqlc SQL files
├── testdata/                       # Golden files for parser tests
├── frontend/
│   ├── src/
│   │   ├── routes/                 # react-router-dom pages
│   │   ├── components/             # UI, viewer, strat, layout
│   │   │   ├── dashboard/          # Faceit profile, elo chart, match list
│   │   │   ├── demos/              # Demo cards, list, drop zone, upload
│   │   │   ├── providers/          # Auth, Query, Theme providers
│   │   │   ├── ui/                 # shadcn/ui primitives
│   │   │   ├── viewer/             # 2D viewer canvas, controls, scoreboard
│   │   │   └── strat/              # Strategy board canvas
│   │   ├── hooks/                  # Custom React hooks
│   │   ├── lib/                    # PixiJS, maps, utils
│   │   ├── stores/                 # Zustand stores
│   │   ├── test/                   # Test setup and helpers
│   │   ├── types/                  # TypeScript types
│   │   └── utils/
│   ├── wailsjs/                    # Auto-generated Wails bindings
│   └── public/maps/                # Radar images
├── e2e/                            # Playwright E2E tests
├── lefthook.yml                    # Pre-commit hook config
├── Makefile                        # Root dev commands
└── docs/                           # PRD, Architecture, Plans, ADRs
```

## Development Commands

```bash
# Wails
wails dev                # Dev mode with hot-reload (Go + frontend)
wails build              # Production build (single binary)

# Go (from project root)
go build ./...           # Build all Go code
go test -race ./...      # Run unit tests (with race detector)
golangci-lint run ./...    # Lint (installed separately, not a go tool)
make sqlc                # Regenerate Go code from SQL (uses `go tool sqlc generate`)

# Frontend (in frontend/)
pnpm dev                 # Dev server on :3000
pnpm build               # Production build
pnpm lint                # ESLint
pnpm typecheck           # tsc --noEmit
pnpm test                # Vitest

# Testing
make test                # Run all tests (unit + integration)
make test-unit           # Go + TS unit tests only
make test-e2e            # Playwright E2E tests (in e2e/)

# Quality
make lint                # Lint Go + TS
make typecheck           # TypeScript type checking
make build               # Build all artifacts
make clean               # Remove build artifacts

# Git Hooks
make hooks               # Install lefthook pre-commit hooks
make hooks-fallback      # Fallback: git core.hooksPath, no extra tools
```

## Key Architectural Patterns

### PixiJS Outside React

PixiJS Application is **not** rendered by React. React renders a container `<div>`, PixiJS is instantiated in `useEffect` and manages its own render loop. Zustand `subscribe()` bridges React controls to PixiJS state. This avoids React re-render overhead on every frame.

### SQLite with WAL Mode

Single embedded SQLite database using `modernc.org/sqlite` (pure Go, no CGo). WAL mode for concurrent reads, single connection (`SetMaxOpenConns(1)`) to avoid `SQLITE_BUSY`. Migrations embedded in binary via `//go:embed` and run by golang-migrate. Database functions in `internal/database/sqlite.go`.

### Wails Bindings (No REST API)

Go struct methods on the App struct are automatically exposed as TypeScript functions in the frontend. No HTTP server, no REST routes. Long-running operations (demo parsing) report progress via Wails runtime events.

**Current status**: Most binding methods in `app.go` are implemented (auth, demo import/parsing, stats, viewer data). A few stubs remain for later phases. Domain types live in `types.go` at the root package level. To implement a remaining stub: (1) write the Go logic in the appropriate `internal/` package, (2) wire it in `app.go`, (3) update the frontend to use real data instead of mocks.

### Synchronous Processing

Demo parsing and Faceit sync run in-process (no background workers, no Redis). Operations are triggered by Wails binding calls and run synchronously with progress events.

### Coordinate Calibration

Each CS2 map has calibration data (`origin_x`, `origin_y`, `scale`) mapping game world-space to radar image pixel-space. Stored in `frontend/src/lib/maps/calibration.ts`. Formula: `pixel_x = (world_x - origin_x) / scale`.

## Test-Writing Discipline

Before writing or modifying any test file, you **must**:

1. **Read an existing test** in the same directory/package to match patterns exactly (imports, wrappers, mock style)
2. **Use the project's test utilities** — never reinvent wrappers:
   - **Frontend**: Always use `renderWithProviders()` from `src/test/render.tsx` (provides QueryClientProvider, ThemeProvider, AuthProvider). Never create a raw `QueryClientProvider` wrapper in a test file.
   - **Frontend mocks**: Use MSW handlers from `src/test/msw/handlers.ts` for Faceit API mocking. Use PixiJS mock factories from `src/test/mocks/pixi.ts`. Mock Wails binding functions from `src/test/mocks/bindings.ts`.
   - **Go mocks**: Use stub implementations from `internal/testutil/mocks.go` (`MockKeyring`, `MockFaceitClient`). Never create ad-hoc mock structs that duplicate these.
   - **Go database tests**: Use `testutil.NewTestDB(t)` or `testutil.NewTestQueries(t)` from `internal/testutil/db.go` for in-memory SQLite with migrations applied. Never open a test database manually.
   - **Go golden files**: Use `testutil.CompareGolden(t, name, got)` and `testutil.LoadFixture(t, name, &v)` from `internal/testutil/golden.go`. Update goldens with `go test -update`. Fixtures live in `testdata/`.
3. **Run the test immediately** after writing it — do not move to the next file until the test passes (the Stop hook runs tests automatically when your turn ends)

## Claude Code Automations

### Hooks (tiered quality checks)
- **PreToolUse**: Blocks edits to lock files (`pnpm-lock.yaml`, `go.sum`) and sqlc-generated `*.sql.go` files
- **PostToolUse (format)**: Auto-formats on every edit — `prettier --write` + `eslint --fix` on TS/TSX, `gofmt` + `goimports` on Go files. Also tracks edited files for Stop hooks.
- **Stop (tests)**: Runs affected tests once when Claude's turn ends — `vitest --related` for TS/TSX source files, direct run for test files, package-scoped `go test -race` for Go files. Only tests packages with edits this turn.
- **Stop (typecheck)**: Runs `tsc --noEmit` for frontend changes and `go vet ./...` for backend changes. Runs once per turn after tests. Cleans up the edited-files tracking list.

### Subagents (`.claude/agents/`)
- **security-reviewer** -- Reviews code for auth, injection, WebSocket, and data exposure vulnerabilities
- **test-writer** -- Generates tests matching project TDD conventions (table-driven Go, RTL+MSW React, Vitest stores)

### Skills
- `/create-migration <name>` -- Creates numbered golang-migrate up/down SQL file pair
- `/gen-test <file>` -- Generates test file for any Go or TS source file

### MCP Servers (`.mcp.json`)
- **Playwright** -- Browser automation for visual testing and debugging
- **Context7** -- Live documentation lookup for project libraries

## Documentation

- `docs/PRD.md` -- Product requirements, user stories, data models
- `docs/ARCHITECTURE.md` -- System design, DB schema, data flows
- `docs/IMPLEMENTATION_PLAN.md` -- 6-phase delivery plan
- `docs/TASK_BREAKDOWN.md` -- 63 granular tasks with acceptance criteria
- `docs/adr/` -- Architecture Decision Records
- `docs/plans/` -- Phase implementation plans
