# Oversite

CS2 2D demo viewer and analytics platform. Single-binary desktop app -- import local demos, watch top-down playback, generate heatmaps, and plan strategies.

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| [Go](https://go.dev/) | 1.26+ | `brew install go` |
| [Node.js](https://nodejs.org/) | 20 LTS | `brew install node` |
| [pnpm](https://pnpm.io/) | 9+ | `corepack enable && corepack prepare pnpm@latest --activate` |
| [Wails](https://wails.io/) | v2 | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |

### Platform-specific

| Platform | Additional requirement |
|----------|----------------------|
| macOS | Xcode Command Line Tools (`xcode-select --install`) |
| Windows | [WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) (usually pre-installed on Windows 10+) |
| Linux | `sudo apt install libwebkit2gtk-4.0-dev build-essential` (Ubuntu/Debian) |

## Quick Start

```bash
git clone git@github.com:ok2ju/oversite.git
cd oversite

# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Install pre-commit hooks
make hooks

# Start dev mode (Go backend + frontend with hot-reload)
wails dev
```

The app opens in a native window. Go changes trigger a backend rebuild; frontend changes hot-reload via Vite.

## Development Commands

```bash
# Wails
wails dev                # Dev mode with hot-reload (Go + frontend)
wails build              # Production build (single binary)

# Go (from project root)
go build ./...           # Build all Go code
go test -race ./...      # Run unit tests (with race detector)
go tool golangci-lint run  # Lint
make sqlc                # Regenerate Go code from SQL queries

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
```

## Project Structure

```
oversite/
├── main.go              # Wails entry point
├── app.go               # App struct (Wails bindings)
├── internal/            # Go business logic (database, demo, store, etc.)
├── migrations/          # SQLite migration files (embedded in binary)
├── queries/             # sqlc SQL files
├── frontend/            # Vite + React 19 SPA
│   ├── src/
│   │   ├── routes/      # react-router-dom pages
│   │   ├── components/  # UI, viewer, strat, layout
│   │   ├── stores/      # Zustand stores
│   │   └── lib/         # PixiJS, maps, utils
│   └── wailsjs/         # Auto-generated Wails bindings
├── e2e/                 # Playwright E2E tests
├── docs/                # Obsidian vault: product, architecture, decisions, plans, knowledge
└── Makefile             # Root dev commands
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Runtime | Wails v2 (Go backend + system WebView frontend) |
| Frontend | Vite + React 19, TypeScript, PixiJS v8, shadcn/ui, Tailwind CSS, Zustand, TanStack Query v5 |
| Backend | Go 1.26+, Wails bindings (no HTTP server) |
| Demo Parsing | markus-wa/demoinfocs-golang v5 |
| Database | SQLite (modernc.org/sqlite, pure Go, WAL mode) |
| SQL | sqlc (type-safe generated Go, SQLite dialect) |
| Auth | None — single-tenant local app |
| Packaging | Single native binary per platform (macOS, Windows, Linux) |

## Documentation

Docs live in an Obsidian vault under [`docs/`](docs/). Start at [`docs/index.md`](docs/index.md) for the map of contents.

- [`docs/product/`](docs/product/) -- Vision, personas, features, user stories, NFRs, data models, Wails bindings
- [`docs/architecture/`](docs/architecture/) -- Arc42 system design: overview, structure, components, data flows, database, crosscutting, testing
- [`docs/decisions/`](docs/decisions/) -- Architecture Decision Records (ADRs)
- [`docs/plans/`](docs/plans/) -- Phase implementation plans (P1–P4)
- [`docs/roadmap.md`](docs/roadmap.md) -- 6-phase delivery plan
- [`docs/tasks.md`](docs/tasks.md) -- 63 granular tasks with acceptance criteria
- [`docs/knowledge/`](docs/knowledge/) -- LLM-maintained wiki of implementation entities/patterns
- [`docs/log.md`](docs/log.md) -- Append-only project activity log
