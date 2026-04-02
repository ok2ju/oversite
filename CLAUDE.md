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
│   │   ├── websocket/              # WS hub, Yjs relay
│   │   └── worker/                 # Job queue consumer
│   ├── migrations/                 # SQL migration files
│   ├── queries/                    # sqlc SQL files
│   └── Makefile
├── frontend/
│   ├── src/
│   │   ├── app/                    # Next.js App Router pages
│   │   ├── components/             # UI, viewer, strat, layout
│   │   ├── hooks/                  # Custom React hooks
│   │   ├── lib/                    # API client, pixi, yjs, maps
│   │   ├── stores/                 # Zustand stores
│   │   ├── types/                  # TypeScript types
│   │   └── utils/
│   └── public/maps/                # Radar images
├── nginx/nginx.conf
├── docker-compose.yml
├── docker-compose.dev.yml
├── Makefile                        # Root dev commands
└── docs/                           # PRD, Architecture, Plans
```

## Development Commands

```bash
# Docker
make up                  # Start all services
make down                # Stop all services
make dev                 # Start with hot-reload
make logs                # Tail all logs
make logs s=api          # Tail specific service

# Database
make migrate-up          # Run migrations
make migrate-down        # Rollback last migration
make migrate-create      # New migration files
make sqlc                # Regenerate Go code from SQL

# Backend (in backend/)
go build ./cmd/oversite  # Build binary
go test ./...            # Run tests
golangci-lint run        # Lint

# Frontend (in frontend/)
pnpm dev                 # Dev server on :3000
pnpm build               # Production build
pnpm lint                # ESLint
pnpm typecheck           # tsc --noEmit
pnpm test                # Vitest

# All
make lint                # Lint Go + TS
make test                # Test Go + TS
make build               # Build all

# Git Hooks
make hooks               # Install lefthook pre-commit hooks
make hooks-fallback      # Fallback: no extra tools needed
```

## Coding Conventions

### Go

- **Router**: chi. Group routes by resource. Middleware applied per-group.
- **SQL**: sqlc generates all DB access code. Write SQL in `queries/*.sql`, run `make sqlc`.
- **Errors**: Return sentinel errors from services (`ErrNotFound`, `ErrForbidden`). Handlers map to HTTP status codes.
- **Logging**: `slog` (stdlib). Structured JSON. Include request ID.
- **Config**: Environment variables. Loaded via `internal/config` into a typed struct.
- **Testing**: TDD (Red-Green-Refactor). Table-driven tests. Use `testcontainers` for integration tests with real DB. Golden file tests for parser output (`-update` flag to regenerate). Integration tests use `//go:build integration` tag. Interface-based DI for service mocking (`Store`, `S3Client`, `SessionStore`, `JobQueue`, `FaceitAPI`). Run unit: `go test ./...`, integration: `go test -tags integration ./...`.

### TypeScript / React

- **State**: Zustand stores per domain (`viewerStore`, `stratStore`, `uiStore`, `faceitStore`). Use selector hooks to minimize re-renders.
- **Data fetching**: TanStack Query for all API calls. No raw `fetch` in components.
- **Components**: shadcn/ui for standard UI. Custom components in `components/viewer/`, `components/strat/`.
- **Styling**: Tailwind CSS utility classes. No CSS modules or styled-components.
- **Testing**: TDD (Red-Green-Refactor). Vitest with React Testing Library for components/hooks. MSW (Mock Service Worker) for API mocking. Pure unit tests for Zustand stores. PixiJS logic unit-tested (transforms, interpolation, state); visual output screenshot-tested with Playwright. TanStack Query hooks tested via `renderHook()` + MSW. Yjs tested with in-memory `Y.Doc` pairs.

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

## Documentation

- `docs/PRD.md` -- Product requirements, user stories, data models
- `docs/ARCHITECTURE.md` -- System design, DB schema, data flows
- `docs/IMPLEMENTATION_PLAN.md` -- 6-phase delivery plan
- `docs/TASK_BREAKDOWN.md` -- 68 granular tasks with acceptance criteria
