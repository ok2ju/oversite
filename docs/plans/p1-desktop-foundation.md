# Phase 1: Desktop Foundation -- Sprint Plan

> **Note (2026-05-06):** Faceit / auth scope was removed from the project. Any
> mention below of `faceitStore`, `MockFaceitClient`, `MockKeyring`, the `users`
> / `faceit_matches` tables, or "Login with Faceit" describes a previous
> direction and is no longer in scope. See `docs/log.md`.

## Context

Oversite is pivoting from a multi-service web app (PostgreSQL, Redis, MinIO, Docker) to a single-binary Wails desktop app (SQLite, local filesystem, no infra). The existing repo has a fully built web-era codebase in `backend/` (Go + Cobra + chi + PostgreSQL) and `frontend/` (Next.js 16 + React 19). **No Wails initialization exists yet.**

Phase 1 establishes the desktop app skeleton: Wails scaffold, SQLite database, Vite+React SPA, test infra, and CI. All subsequent phases (auth, parser, viewer, etc.) build on this foundation.

---

## Dependency Graph

```
T01 (Wails init) ──┬── T02 (SQLite) ──┬── T03 (sqlc)
                   │                   └── T09 (Go test infra) ──┐
                   ├── T04 (React SPA) ┬── T05 (shadcn/Tailwind) │
                   │                   ├── T06 (Zustand stores)   ├── T07 (CI)
                   │                   └── T10 (FE test infra) ───┘
                   └── T08 (Makefile)
```

## Execution Order

| Wave | Tasks | Parallel? | Notes |
|------|-------|-----------|-------|
| 1 | T01 | Serial | Foundation -- everything depends on it |
| 2 | T02, T04, T08 | Parallel | Backend DB + Frontend shell + Makefile |
| 3 | T03, T05, T06, T09, T10 | Parallel | All depend only on Wave 2 |
| 4 | T07 | Serial | CI depends on T09 + T10 |

---

## Task Plans

### T01: Initialize Wails Project -- COMPLETED

**Why:** This is the most structurally impactful task. It defines the directory layout, Go module path, Wails config, embed.FS integration, and app struct that every other task depends on.

**Key decisions requiring dedicated planning:**
- New Go module at project root (`github.com/ok2ju/oversite`) vs keeping `backend/` module
- What to do with existing `backend/` and `frontend/` directories (archive? keep as reference?)
- `wails.json` configuration (asset directory, build commands, dev server)
- `App` struct initial binding method signatures
- `embed.FS` setup for frontend assets
- Go dependency selection (drop lib/pq, redis, minio, cobra, testcontainers; add wails/v2, modernc.org/sqlite, go-keyring)
- Whether to use `wails init` template or scaffold manually

**Acceptance:** `wails dev` opens a window, `go build ./...` compiles, `pnpm dev` serves frontend.

---

### T02: Set Up SQLite with Migrations -- COMPLETED

**Why:** Translating 10 PostgreSQL tables to SQLite involves non-trivial type mapping and design decisions.

**Key decisions requiring dedicated planning:**
- PK strategy: `TEXT` (UUIDs generated in Go) vs `INTEGER AUTOINCREMENT` -- affects every table and all frontend type contracts
- Type mappings: `UUID` -> `TEXT`, `TIMESTAMPTZ` -> `TEXT` (ISO 8601), `JSONB` -> `TEXT`, `BYTEA` -> `BLOB`, `TEXT[]` -> `TEXT` (JSON), `BOOLEAN` -> `INTEGER`, `SMALLINT` -> `INTEGER`
- tick_data table: no hypertable/compression, composite PK `(demo_id, tick, steam_id)` instead of TimescaleDB time-series
- Strategy boards: `board_state TEXT` (JSON) replaces `yjs_state BYTEA` (single-user, no Yjs)
- Migration framework: `golang-migrate/v4` with `sqlite` driver + `embed.FS` source
- Pragmas: `journal_mode=WAL`, `foreign_keys=ON`, `busy_timeout=5000`
- DB file location: `os.UserConfigDir()/oversite/oversite.db`

**Source reference:** `backend/migrations/001_initial_schema.up.sql` (216 lines of PostgreSQL DDL)

**Acceptance:** In-memory SQLite opens, WAL enabled, all tables created, foreign keys enforced.

---

### T03: Configure sqlc for SQLite -- COMPLETED

**Why:** 10 query files need PostgreSQL-to-SQLite translation, with 3-4 complex queries that have no direct SQLite equivalent.

**Key decisions requiring dedicated planning:**
- `sqlc.yaml`: engine `sqlite`, queries/schema paths adjusted for new layout
- Parameter placeholders: `$1` -> `?` (sqlc handles this automatically with engine change)
- `uuid_generate_v4()` in DEFAULTs removed -- Go generates UUIDs before insert
- `NOW()` -> `datetime('now')`
- **Hard translations:**
  - `heatmaps.sql`: uses `ANY($4::varchar[])` and `cardinality()` -- no SQLite equivalent. Rewrite with `json_each()` or dynamic `IN` clause
  - `faceit_matches.sql`: uses `sqlc.narg()` with type casts -- verify SQLite support
  - `tick_data.sql`: uses `ANY()` for steam_id filtering
- `RETURNING *` works in SQLite 3.35+ -- verify modernc.org/sqlite version supports it
- `ON CONFLICT DO NOTHING` supported in SQLite

**Source reference:** `backend/queries/*.sql` (10 files), `backend/sqlc.yaml`

**Acceptance:** `sqlc generate` succeeds, generated code compiles, basic insert+select test passes.

---

### T04: Scaffold Vite + React Frontend -- COMPLETED

**Complexity:** M | **Depends on:** T01

**Implementation plan:**

1. **Set up Vite project** in `frontend/`:
   - `vite.config.ts` with `@vitejs/plugin-react`, path aliases (`@/` -> `src/`)
   - `tsconfig.json` (remove Next.js plugin, `next-env.d.ts`)
   - `index.html` as Vite entry point
   - `src/main.tsx` with `createRoot` (React 19)

2. **Install react-router-dom v6** and create route structure:
   ```
   /              -> redirect to /dashboard
   /login         -> LoginPage (placeholder)
   /dashboard     -> DashboardPage (placeholder)
   /demos         -> DemosPage (placeholder)
   /demos/:id     -> DemoViewerPage (placeholder)
   /heatmaps      -> HeatmapsPage (placeholder)
   /strats        -> StratsPage (placeholder)
   /strats/:id    -> StratBoardPage (placeholder)
   /lineups       -> LineupsPage (placeholder)
   /settings      -> SettingsPage (placeholder)
   ```

3. **Create app shell** (root layout):
   - `routes/root.tsx`: `<Sidebar>` + `<Header>` + `<Outlet>`
   - `components/layout/sidebar.tsx`: nav links with `<NavLink>` (active state via `className` callback)
   - `components/layout/header.tsx`: app title + user avatar placeholder

4. **Wails JS runtime integration:**
   - Import `wailsjs/runtime` for window controls, events
   - Set up `wailsjs/go/main/App` import path for binding calls

5. **Migrate from existing frontend:**
   - Copy `globals.css` (CSS custom properties, Tailwind theme -- no Next.js deps)
   - Copy `lib/utils.ts` (`cn()` function)
   - Rewrite sidebar: replace `next/navigation` (`usePathname`) with `react-router-dom` (`useLocation`, `NavLink`)
   - Rewrite header: remove any Next.js imports

**Key files:**
- `frontend/vite.config.ts` (new)
- `frontend/index.html` (new)
- `frontend/src/main.tsx` (new)
- `frontend/src/App.tsx` (new -- router setup)
- `frontend/src/routes/root.tsx` (new -- app shell)
- `frontend/src/routes/*.tsx` (new -- placeholder pages)
- `frontend/src/components/layout/sidebar.tsx` (rewritten from existing)
- `frontend/src/components/layout/header.tsx` (adapted from existing)

**React version note:** Using React 19. Zustand stores and PixiJS logic are version-agnostic.

**Acceptance:** All routes render placeholders, sidebar highlights active route, Wails JS runtime imports resolve.

---

### T05: Configure shadcn/ui + Tailwind CSS -- COMPLETED

**Complexity:** S | **Depends on:** T04

**Implementation plan:**

1. Tailwind CSS 4 is already configured via `globals.css` (copied in T04) -- `@import "tailwindcss"` + `@theme` block
2. Copy `components.json` (shadcn/ui config) from existing frontend, update paths if needed
3. Copy all existing `components/ui/*.tsx` files verbatim (these are framework-agnostic Radix primitives)
4. Theme toggle: replace `next-themes` with minimal custom provider (toggle `class="dark"` on `<html>`)
5. Install Radix UI deps: `@radix-ui/react-*`, `class-variance-authority`, `clsx`, `tailwind-merge`, `lucide-react`

**Existing shadcn components to copy:** button, card, dialog, dropdown-menu, alert-dialog, badge, progress, skeleton, tabs, select, slider, input, table, toast, separator

**Key files:**
- `frontend/components.json`
- `frontend/src/components/ui/*.tsx` (copied)
- `frontend/src/components/providers/theme-provider.tsx` (rewritten)

**Acceptance:** Components render, dark/light toggle works, Tailwind classes apply.

---

### T06: Set Up Zustand Stores (Skeleton) -- COMPLETED

**Complexity:** S | **Depends on:** T04

**Implementation plan:**

1. Copy existing stores verbatim (they are pure Zustand, zero framework deps):
   - `stores/viewer.ts` + `stores/viewer.test.ts`
   - `stores/ui.ts` + `stores/ui.test.ts`
   - `stores/faceit.ts` + `stores/faceit.test.ts`
   - `stores/strat.ts` + `stores/strat.test.ts`

2. Create new `stores/demo.ts` for desktop-specific demo library state:
   - State: `demos[]`, `selectedDemoId`, `importProgress`, `filters`
   - Actions: `setDemos`, `selectDemo`, `updateImportProgress`, `setFilters`
   - Write unit tests first (TDD)

**Key files:**
- `frontend/src/stores/*.ts` (4 copied, 1 new)
- `frontend/src/stores/*.test.ts` (4 copied, 1 new)

**Acceptance:** All stores typed, tests pass, selectors exported.

---

### T07: Set Up CI Pipeline -- COMPLETED

**Complexity:** M | **Depends on:** T01, T09, T10

**Implementation plan:**

1. Rewrite `.github/workflows/ci.yml`:
   - **Trigger:** push + PR to `main`
   - **Go lint job:** `golangci-lint run ./...` (from project root, not `backend/`)
   - **Go test job:** `go test -race ./...` (uses in-memory SQLite, no containers)
   - **Frontend lint + typecheck:** `pnpm lint` + `pnpm typecheck`
   - **Frontend test:** `pnpm test`
   - **Wails build:** Install Wails CLI, run `wails build` (at least one platform)
   - Remove: Docker build job, integration test job (testcontainers), Go build with CGO_ENABLED=0

2. Build matrix considerations:
   - macOS (ARM64) for primary development
   - Linux (AMD64) for CI validation
   - Windows build can be deferred to P6

3. Cache: Go modules (`~/go/pkg/mod`), pnpm store, Wails cache

**Key files:**
- `.github/workflows/ci.yml` (rewritten)

**Acceptance:** CI runs on push/PR, all lint + test + build jobs pass.

---

### T08: Create Root Makefile -- COMPLETED

**Complexity:** S | **Depends on:** T01

**Implementation plan:**

Replace Docker-centric Makefile with Wails-centric targets:

```makefile
.PHONY: dev build test lint typecheck sqlc migrate-create clean

dev:            ## Start Wails dev mode (hot reload)
	wails dev

build:          ## Build production binary
	wails build

test:           ## Run all tests
	go test -race ./...
	cd frontend && pnpm test

test-go:        ## Run Go tests only
	go test -race ./...

test-fe:        ## Run frontend tests only
	cd frontend && pnpm test

lint:           ## Lint Go + TypeScript
	go tool golangci-lint run ./...
	cd frontend && pnpm lint

typecheck:      ## TypeScript type checking
	cd frontend && pnpm typecheck

sqlc:           ## Regenerate sqlc code
	go tool sqlc generate

migrate-create: ## Create new migration pair
	@read -p "Migration name: " name; \
	touch migrations/$$(printf "%03d" $$(ls migrations/*.up.sql 2>/dev/null | wc -l | tr -d ' ' | xargs -I{} expr {} + 1))_$${name}.up.sql \
	      migrations/$$(printf "%03d" $$(ls migrations/*.up.sql 2>/dev/null | wc -l | tr -d ' ' | xargs -I{} expr {} + 1))_$${name}.down.sql

clean:          ## Remove build artifacts
	rm -rf build/bin
```

**Key files:**
- `Makefile` (rewritten)

**Acceptance:** All targets execute correctly.

---

### T09: Set Up Go Test Infrastructure -- COMPLETED

**Complexity:** M | **Depends on:** T02

**Implementation plan:**

1. Create `internal/testutil/db.go`:
   - `NewTestDB(t *testing.T) *sql.DB` -- opens `:memory:` SQLite, sets pragmas (WAL not needed for memory, but `foreign_keys=ON`), runs all embedded migrations, registers `t.Cleanup` to close
   - `NewTestQueries(t *testing.T) (*store.Queries, *sql.DB)` -- wraps `NewTestDB`, returns sqlc queries instance
   - Migrations embedded via `embed.FS` from `migrations/` directory

2. Create `internal/testutil/mocks.go`:
   - `MockFaceitClient` -- implements Faceit API interface (kept from existing, adapted)
   - `MockKeyring` -- implements keyring interface for auth token storage
   - Remove: `StubS3Client`, `StubSessionStore` (Redis), `StubJobQueue`

3. Create `internal/testutil/golden.go`:
   - Copy from existing `backend/internal/testutil/` -- golden file helpers are framework-agnostic
   - `UpdateGolden(t, filename, got)` + `CompareGolden(t, filename, got)`

4. Write example test to validate the setup works end-to-end

**Key files:**
- `internal/testutil/db.go` (new)
- `internal/testutil/mocks.go` (adapted)
- `internal/testutil/golden.go` (copied/adapted)

**Acceptance:** `NewTestDB()` returns migrated DB, `go test -race ./...` passes with example test.

---

### T10: Set Up Frontend Test Infrastructure -- COMPLETED

**Complexity:** M | **Depends on:** T04

**Implementation plan:**

1. **Vitest config** (`frontend/vitest.config.ts`):
   - Adapt existing config (already Vite-based)
   - Environment: `jsdom`, globals: `true`
   - Setup file: `src/test/setup.ts`
   - Include: `src/**/*.test.{ts,tsx}`
   - Path aliases matching `vite.config.ts`

2. **Test setup** (`frontend/src/test/setup.ts`):
   - Copy existing setup (DOM cleanup, expect extensions)
   - Remove `next-themes` matchMedia stub if not needed

3. **Render helper** (`frontend/src/test/render.tsx`):
   - `renderWithProviders()` wrapping: `QueryClientProvider`, `MemoryRouter` (react-router-dom), `ThemeProvider`
   - Accept `initialRoute` parameter for router testing
   - No `AuthProvider` yet (Phase 2)

4. **Wails binding mocks** (`frontend/src/test/mocks/bindings.ts`):
   - Mock factory for `wailsjs/go/main/App` methods
   - Pattern: `vi.mock('@wailsjs/go/main/App')` with typed return values
   - Replaces MSW handlers (Wails bindings are function calls, not HTTP)

5. **Preserve reusable mock data:**
   - Extract fixture data (mockDemos, mockFaceitMatches, etc.) from existing MSW handlers into `test/fixtures/`
   - These fixtures serve both binding mocks and component tests

6. **Playwright config** (`e2e/playwright.config.ts`):
   - Update `baseURL` to Wails dev server port
   - Update `webServer` command to `wails dev`

**Key files:**
- `frontend/vitest.config.ts` (adapted)
- `frontend/src/test/setup.ts` (adapted)
- `frontend/src/test/render.tsx` (rewritten)
- `frontend/src/test/mocks/bindings.ts` (new)
- `frontend/src/test/fixtures/*.ts` (new, extracted from MSW handlers)
- `e2e/playwright.config.ts` (updated)

**Acceptance:** `pnpm test` runs with example component test passing, `renderWithProviders()` works with router.

---

## Tasks Requiring Separate Planning

| Task | Reason |
|------|--------|
| **T01: Initialize Wails project** | Foundational structure -- Go module path, directory layout, wails.json, embed.FS, dep selection. Every other task depends on choices made here. |
| **T02: Set up SQLite with migrations** | 10-table PostgreSQL-to-SQLite schema translation with type mapping (UUID, TIMESTAMPTZ, JSONB, TEXT[], BYTEA, hypertable). Design decisions on PK strategy affect the entire codebase. |
| **T03: Configure sqlc for SQLite** | 10 query files with 3-4 complex translations (`ANY()`, `cardinality()`, `sqlc.narg` casts). Mechanical for simple CRUD but the heatmap/faceit/tick queries need careful design. |

---

## Reuse Strategy

**Copy verbatim** (no changes needed):
- `stores/viewer.ts`, `stores/ui.ts`, `stores/faceit.ts`, `stores/strat.ts` + tests
- `components/ui/*.tsx` (shadcn/ui -- framework-agnostic)
- `lib/pixi/*` (PixiJS rendering -- no framework deps)
- `lib/maps/calibration.ts`
- `types/*.ts`
- `globals.css`
- `lib/utils.ts`
- `testdata/` golden files
- `demo/parser.go`, `demo/validate.go`, `demo/stats.go` (pure parsing logic)

**Copy with rewrites:**
- `components/layout/sidebar.tsx` -- `next/navigation` -> `react-router-dom`
- `components/providers/theme-provider.tsx` -- `next-themes` -> custom
- `test/render.tsx` -- add `MemoryRouter`, remove Next.js providers

**Do not copy** (replaced by Wails patterns):
- `cmd/oversite/main.go` (Cobra CLI -> Wails entry)
- `handler/*` (HTTP handlers -> Wails binding methods)
- `middleware/*` (no HTTP middleware)
- `worker/*` (no Redis Streams)
- `websocket/*` (no WebSocket in desktop)
- `testutil/containers.go` (testcontainers -> in-memory SQLite)
- MSW handlers (HTTP mocks -> binding mocks)

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| `modernc.org/sqlite` + sqlc compatibility | High | Validate `RETURNING *`, `ON CONFLICT`, `json_each()` in T02/T03 planning |
| Tick data bulk insert perf without COPY | Medium | Benchmark batched INSERT (10K rows/tx) in T02; fallback to `PRAGMA synchronous=OFF` during import |
| Wails WebView + PixiJS WebGL | Low (P1) | Runtime testing deferred to P3 (viewer phase) |
| `golang-migrate` SQLite driver registration | Low | Verify modernc.org/sqlite registers as `"sqlite"` in database/sql |

---

## Verification Plan

After all 10 tasks complete:

1. `wails dev` launches a desktop window with the app shell
2. Sidebar navigation works between all placeholder pages
3. Dark/light theme toggle works
4. `go test -race ./...` passes (SQLite test helpers + example tests)
5. `pnpm test` passes (Vitest + example component test)
6. `make lint` + `make typecheck` pass
7. `sqlc generate` produces compilable code
8. CI pipeline passes on push to feature branch
