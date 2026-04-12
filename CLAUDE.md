# Oversite

CS2 2D demo viewer and analytics platform for Faceit players. Upload demos, watch top-down playback, generate heatmaps, collaborate on strategies, and track Faceit stats -- all in the browser.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Next.js 14+ (App Router), TypeScript, PixiJS v8, shadcn/ui, Tailwind CSS, Zustand, TanStack Query v5 |
| Backend | Go 1.22+, chi router, gorilla/websocket |
| Demo Parsing | markus-wa/demoinfocs-golang v5 |
| Database | PostgreSQL 16 + TimescaleDB (hypertable for tick data) |
| SQL | sqlc (type-safe generated Go) |
| Cache/Queue | Redis 7 (sessions, cache, Redis Streams job queue) |
| Object Storage | MinIO (S3-compatible, `.dem` files) |
| Collaboration | Yjs CRDT (strategy board real-time sync) |
| Auth | Faceit OAuth 2.0 + PKCE |
| Infra | Docker Compose, nginx reverse proxy |

## Monorepo Structure

```
oversite/
├── backend/
│   ├── cmd/oversite/main.go        # CLI: serve, ws, worker, migrate
│   ├── internal/
│   │   ├── auth/                   # OAuth, sessions, middleware
│   │   ├── config/                 # Env-based config
│   │   ├── demo/                   # Parser, ingest, heatmap
│   │   ├── faceit/                 # API client, sync
│   │   ├── handler/                # HTTP handlers (chi)
│   │   ├── lineup/                 # Grenade lineup service
│   │   ├── middleware/             # CORS, rate limit, logging
│   │   ├── model/                  # Domain types
│   │   ├── store/                  # sqlc generated code
│   │   ├── strat/                  # Strategy board service
│   │   ├── testutil/               # Shared test helpers
│   │   ├── websocket/              # WS hub, Yjs relay
│   │   └── worker/                 # Job queue consumer
│   ├── migrations/                 # SQL migration files
│   ├── queries/                    # sqlc SQL files
│   ├── testdata/                   # Golden files for parser tests
│   └── Makefile
├── frontend/
│   ├── src/
│   │   ├── app/                    # Next.js App Router pages
│   │   ├── components/             # UI, viewer, strat, layout
│   │   ├── hooks/                  # Custom React hooks
│   │   ├── lib/                    # API client, pixi, yjs, maps
│   │   ├── stores/                 # Zustand stores
│   │   ├── test/                   # Test setup and helpers
│   │   ├── types/                  # TypeScript types
│   │   └── utils/
│   └── public/maps/                # Radar images
├── e2e/                            # Playwright E2E tests
├── nginx/nginx.conf
├── docker-compose.yml
├── docker-compose.dev.yml
├── lefthook.yml                    # Pre-commit hook config
├── Makefile                        # Root dev commands
└── docs/                           # PRD, Architecture, Plans, ADRs
```

## Development Commands

```bash
# Docker
make up                  # Start all services in background
make down                # Stop all services
make dev                 # Start with hot-reload (foreground)
make logs                # Tail all logs
make logs s=api          # Tail specific service
make ps                  # Show running services
make restart s=api       # Restart a specific service

# Database
make migrate-up          # Run all pending migrations
make migrate-down        # Rollback last migration
make migrate-create      # New migration files (interactive prompt)
make sqlc                # Regenerate Go code from SQL

# Backend (in backend/)
go build ./cmd/oversite  # Build binary
go test -race ./...      # Run unit tests (with race detector)
go tool golangci-lint run  # Lint

# Frontend (in frontend/)
pnpm dev                 # Dev server on :3000
pnpm build               # Production build
pnpm lint                # ESLint
pnpm typecheck           # tsc --noEmit
pnpm test                # Vitest

# Testing
make test                # Run all tests (unit + integration)
make test-unit           # Go + TS unit tests only
make test-integration    # Go integration tests (requires Docker)
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

### Yjs Dumb Relay

The Go WebSocket server does **not** parse Yjs messages. It receives binary Yjs updates from one client and broadcasts to all others in the room. State persistence: encode full Yjs doc to binary, store in `strategy_boards.yjs_state` (BYTEA column). This keeps Go simple -- all CRDT logic runs in the browser.

### Redis Streams Job Queue

Background jobs (demo parsing, Faceit sync) use Redis Streams with consumer groups. API server produces jobs (`XADD`), worker process consumes (`XREADGROUP`). Jobs acknowledged on success (`XACK`), retried on failure (max 3 attempts), dead-lettered after.

### TimescaleDB for Tick Data

Player position data (10 players x 64 ticks/sec x ~2000 seconds = ~1.28M rows per demo) stored in a TimescaleDB hypertable partitioned by synthetic timestamp. Compression policy compresses chunks > 7 days old. Query by `(demo_id, tick range)` for viewer playback.

### Coordinate Calibration

Each CS2 map has calibration data (`origin_x`, `origin_y`, `scale`) mapping game world-space to radar image pixel-space. Stored in `frontend/src/lib/maps/calibration.ts`. Formula: `pixel_x = (world_x - origin_x) / scale`.

## Docker Services

| Service | Port | Network |
|---------|------|---------|
| nginx | 80, 443 | frontend |
| web (Next.js) | 3000 | frontend |
| api (Go) | 8080 | frontend, backend |
| ws (Go) | 8081 | frontend, backend |
| worker (Go) | - | backend |
| postgres | 5432 | backend |
| redis | 6379 | backend |
| minio | 9000, 9001 | backend |

## API Routes

- `/api/v1/auth/*` -- Faceit OAuth (no auth required)
- `/api/v1/demos/*` -- Demo CRUD, upload, tick data, events
- `/api/v1/heatmaps/*` -- Aggregated heatmap generation
- `/api/v1/strats/*` -- Strategy board CRUD + sharing
- `/api/v1/lineups/*` -- Grenade lineup CRUD
- `/api/v1/faceit/*` -- Faceit profile, ELO, matches
- `/ws/viewer/:demoId` -- Demo playback sync
- `/ws/strat/:stratId` -- Yjs strategy board collaboration
- `/healthz`, `/readyz` -- Health checks

## Test-Writing Discipline

Before writing or modifying any test file, you **must**:

1. **Read an existing test** in the same directory/package to match patterns exactly (imports, wrappers, mock style)
2. **Use the project's test utilities** — never reinvent wrappers:
   - **Frontend**: Always use `renderWithProviders()` from `src/test/render.tsx` (provides QueryClientProvider, ThemeProvider, AuthProvider). Never create a raw `QueryClientProvider` wrapper in a test file.
   - **Frontend mocks**: Use MSW handlers from `src/test/msw/handlers.ts` for API mocking. Use PixiJS mock factories from `src/test/mocks/pixi.ts`. Use `vi.mock()` with hoisting only for Next.js navigation (`next/navigation`, `next/router`).
   - **Go mocks**: Use stub implementations from `internal/testutil/mocks.go` (`StubS3Client`, `StubSessionStore`, `StubJobQueue`, `StubFaceitAPI`). Never create ad-hoc mock structs that duplicate these.
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
- `docs/TASK_BREAKDOWN.md` -- 68 granular tasks with acceptance criteria
