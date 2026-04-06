# Oversite -- Task Breakdown

> **Version:** 1.0
> **Last Updated:** 2026-03-31

---

## Table of Contents

1. [Task Legend](#1-task-legend)
2. [Phase 1: Foundation](#2-phase-1-foundation)
3. [Phase 2: Auth & Demo Pipeline](#3-phase-2-auth--demo-pipeline)
4. [Phase 3: Core 2D Viewer](#4-phase-3-core-2d-viewer)
5. [Phase 4: Faceit & Heatmaps](#5-phase-4-faceit--heatmaps)
6. [Phase 5: Strategy Board & Lineups](#6-phase-5-strategy-board--lineups)
7. [Phase 6: Polish & Deploy](#7-phase-6-polish--deploy)
8. [Critical Path Analysis](#8-critical-path-analysis)
9. [Risk Register](#9-risk-register)
10. [Development Environment Setup](#10-development-environment-setup)
11. [Sprint Pairing Recommendations](#11-sprint-pairing-recommendations)

---

## 1. Task Legend

| Field | Description |
|-------|-------------|
| **ID** | `P{phase}-T{number}` |
| **Complexity** | S (< 4h), M (4-12h), L (1-3 days), XL (3-5 days) |
| **Deps** | Task IDs that must complete first |
| **Test Types** | `unit`, `integration`, `golden`, `component`, `screenshot`, `e2e` -- which apply to this task |
| **TDD Workflow** | RED → GREEN → REFACTOR steps specific to this task |
| **Key Files** | Primary files created or modified (including test files) |

### TDD Workflow Convention

Every task follows the **Red-Green-Refactor** cycle unless marked `N/A`:

1. **RED**: Write failing tests that define the expected behavior
2. **GREEN**: Write the minimum code to make tests pass
3. **REFACTOR**: Clean up implementation and tests; all tests stay green
4. **COMMIT**: Commit after each green-to-refactor cycle

**Exceptions** (marked `TDD Workflow: N/A`): Infrastructure and configuration tasks (P1-T01, P1-T02, P1-T03, P1-T10, P1-T11, P1-T12, P6-T04, P6-T06, P6-T07, P6-T08) are verified via smoke tests or health checks, not TDD.

---

## 2. Phase 1: Foundation

### P1-T01: Initialize monorepo structure ✅

| | |
|---|---|
| **Complexity** | S |
| **Deps** | None |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via directory structure inspection and `go mod init` / `pnpm install` success |
| **Description** | Create the top-level directory structure for the monorepo: `backend/`, `frontend/`, `nginx/`, `docs/`, root configs. Initialize Go module (`go mod init`), Next.js project (`pnpm create next-app`), and root Makefile. |
| **Key Files** | `backend/go.mod`, `frontend/package.json`, `Makefile`, `.gitignore` |
| **Acceptance Criteria** | - Directory structure matches ARCHITECTURE.md Section 12 |
| | - `go mod init` succeeds in `backend/` |
| | - `pnpm install` succeeds in `frontend/` |
| | - Root `.gitignore` covers Go binaries, `node_modules`, `.env`, IDE files |

### P1-T02: Set up Docker Compose (all 8 services) ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T01 |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via `docker compose up` and health checks |
| **Description** | Create `docker-compose.yml` with all 8 services: nginx, web (Next.js), api (Go), ws (Go), worker (Go), postgres (TimescaleDB), redis, minio. Create `docker-compose.dev.yml` with hot-reload overrides. Define `frontend` and `backend` networks. Define named volumes for data persistence. |
| **Key Files** | `docker-compose.yml`, `docker-compose.dev.yml`, `backend/Dockerfile`, `frontend/Dockerfile` |
| **Acceptance Criteria** | - `docker compose up` starts all 8 containers |
| | - `docker compose ps` shows all healthy |
| | - PostgreSQL accepts connections on port 5432 |
| | - Redis accepts connections on port 6379 |
| | - MinIO console accessible on port 9001 |
| | - Network segmentation: postgres/redis/minio not on `frontend` network |

### P1-T03: Set up nginx reverse proxy config ✅

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T02 |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via curl to each upstream path |
| **Description** | Create nginx configuration that routes `/` to the frontend, `/api/` to the Go API server, and `/ws/` to the Go WebSocket server. Configure WebSocket upgrade headers. Set `client_max_body_size` for demo uploads. |
| **Key Files** | `nginx/nginx.conf` |
| **Acceptance Criteria** | - HTTP requests to `/` reach Next.js |
| | - HTTP requests to `/api/healthz` reach Go API |
| | - WebSocket upgrade on `/ws/` succeeds |
| | - Upload limit set to 500 MB |

### P1-T04: Scaffold Go backend ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T01 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write test for `GET /healthz` returning 200 with `{"status":"ok"}`. Write test for config loading from env vars. 2. GREEN: Implement main.go, chi router, health handler, config struct. 3. REFACTOR: Extract router setup into reusable function. |
| **Description** | Create the Go backend structure: `cmd/oversite/main.go` with subcommands (serve, ws, worker, migrate). Set up chi router with basic middleware (logging, recovery, CORS). Implement `/healthz` and `/readyz` endpoints. Create the internal package skeleton. |
| **Key Files** | `backend/cmd/oversite/main.go`, `backend/internal/handler/health.go`, `backend/internal/middleware/*.go`, `backend/internal/config/config.go`, `backend/internal/handler/health_test.go`, `backend/internal/config/config_test.go` |
| **Acceptance Criteria** | - `go build ./cmd/oversite` compiles |
| | - `./oversite serve` starts HTTP server on :8080 |
| | - `GET /healthz` returns `{"status": "ok"}` |
| | - `GET /readyz` checks DB, Redis, MinIO and reports status |
| | - Environment-based configuration loads from `DATABASE_URL`, `REDIS_URL`, etc. |
| | - Health handler test passes |
| | - Config loading test covers required env vars |

### P1-T05: Create database migrations ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T02, P1-T04 |
| **Test Types** | integration |
| **TDD Workflow** | 1. RED: Write integration test that runs migrate-up and verifies all tables exist via `information_schema`. 2. GREEN: Create migration SQL files. 3. REFACTOR: Ensure idempotent up/down migrations. |
| **Description** | Set up golang-migrate. Create migration files for the full database schema: users, demos, rounds, player_rounds, tick_data (hypertable), game_events, strategy_boards, grenade_lineups, faceit_matches. Include indexes and TimescaleDB configuration (compression, chunk interval). |
| **Key Files** | `backend/migrations/001_initial_schema.up.sql`, `backend/migrations/001_initial_schema.down.sql`, `backend/migrations/001_initial_schema_test.go` |
| **Acceptance Criteria** | - `./oversite migrate up` creates all tables |
| | - `./oversite migrate down` drops all tables cleanly |
| | - `tick_data` is a TimescaleDB hypertable |
| | - All indexes from ARCHITECTURE.md Section 8 are created |
| | - Compression policy is set on `tick_data` |
| | - Integration test verifies all 9 tables created via testcontainers |
| | - Down migration test verifies clean rollback |

### P1-T06: Configure sqlc and generate Go code ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T05 |
| **Test Types** | integration |
| **TDD Workflow** | 1. RED: Write integration tests that INSERT and SELECT each entity type against testcontainers PG. 2. GREEN: Write SQL queries in `queries/*.sql`, run `sqlc generate`. 3. REFACTOR: Optimize batch insert queries; add missing indexes. |
| **Description** | Configure `sqlc.yaml` for PostgreSQL. Write initial SQL queries for CRUD operations on all tables. Generate Go code. Verify generated types match the schema. |
| **Key Files** | `backend/sqlc.yaml`, `backend/queries/*.sql`, `backend/internal/store/` (generated), `backend/internal/store/queries_integration_test.go` |
| **Acceptance Criteria** | - `sqlc generate` produces Go code without errors |
| | - Generated types for all 9 tables |
| | - Basic CRUD queries for users, demos, rounds, game_events |
| | - Batch insert query for tick_data |
| | - Query for tick range retrieval (by demo_id, start_tick, end_tick) |
| | - Integration tests verify CRUD for users, demos, rounds, game_events |
| | - Batch insert test for tick_data verifies 10k+ rows |
| | - All sqlc integration tests pass in CI |

### P1-T07: Scaffold Next.js frontend ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T01 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write component test for sidebar rendering navigation links. Write test for dark mode toggle. 2. GREEN: Implement layout, sidebar, header, placeholder pages. 3. REFACTOR: Extract nav items to config array. |
| **Description** | Initialize Next.js 14+ with App Router, TypeScript, Tailwind CSS, and pnpm. Install and configure shadcn/ui. Create the authenticated app shell layout (sidebar + header). Create placeholder pages for all routes (dashboard, demos, heatmaps, strats, lineups, settings). Set up TanStack Query provider. |
| **Key Files** | `frontend/src/app/(app)/layout.tsx`, `frontend/src/app/(app)/dashboard/page.tsx`, `frontend/src/components/layout/sidebar.tsx`, `frontend/src/components/layout/header.tsx`, `frontend/tailwind.config.ts`, `frontend/next.config.js`, `frontend/src/components/layout/sidebar.test.tsx` |
| **Acceptance Criteria** | - `pnpm dev` starts Next.js on :3000 |
| | - App shell renders with sidebar navigation |
| | - All placeholder routes accessible |
| | - shadcn/ui Button, Card, and other base components work |
| | - TanStack Query provider wraps the app |
| | - Dark mode toggle works |
| | - Sidebar component test verifies all navigation links render |
| | - Dark mode toggle test passes |

### P1-T08: Set up Zustand stores ✅

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T07 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write unit tests for each store: viewerStore (setTick, setSpeed, togglePlay), stratStore (setTool, setBoard), uiStore (toggleSidebar), faceitStore (setProfile). 2. GREEN: Implement store definitions with typed state and actions. 3. REFACTOR: Ensure selectors are type-safe and performant. |
| **Description** | Create skeleton Zustand stores for each domain: `viewerStore` (current tick, playback state, speed), `stratStore` (current board, tool), `uiStore` (sidebar open, modals), `faceitStore` (profile, matches). Export typed hooks. |
| **Key Files** | `frontend/src/stores/viewer.ts`, `frontend/src/stores/strat.ts`, `frontend/src/stores/ui.ts`, `frontend/src/stores/faceit.ts`, `frontend/src/stores/viewer.test.ts`, `frontend/src/stores/strat.test.ts`, `frontend/src/stores/ui.test.ts`, `frontend/src/stores/faceit.test.ts` |
| **Acceptance Criteria** | - Stores import and initialize without errors |
| | - TypeScript types for all store state and actions |
| | - Selector hooks exported (e.g., `useViewerStore(s => s.currentTick)`) |
| | - Unit tests for all store actions and selectors |
| | - Stores reset correctly between tests |
| | - All store tests pass in CI |

### P1-T09: Set up CI pipeline ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T04, P1-T07 |
| **Test Types** | N/A (CI is the test runner) |
| **TDD Workflow** | N/A -- the CI pipeline itself is the test infrastructure. Verify by pushing a commit and watching the pipeline. |
| **Description** | Create GitHub Actions workflow for CI: lint Go (`golangci-lint`), lint TS (`eslint`, `tsc --noEmit`), run Go tests, run Next.js tests (Vitest), build Go binary, build Next.js. Run on push to `main` and all PRs. |
| **Key Files** | `.github/workflows/ci.yml` |
| **Acceptance Criteria** | - CI runs on push to main and PRs |
| | - Go lint + test + build steps pass |
| | - Frontend lint + typecheck + build steps pass |
| | - Pipeline fails on any step failure |
| | - Docker build step validates Dockerfiles |
| | - CI runs unit tests (`go test`, `pnpm test`) as separate stage |
| | - CI runs integration tests (`go test -tags integration`) as separate stage |
| | - Integration test stage has Docker/testcontainers support |

### P1-T10: Create root Makefile ✅

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T02 |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify by running each make target |
| **Description** | Create a root `Makefile` with common development commands: `make up` (start Docker), `make down`, `make logs`, `make migrate-up`, `make migrate-down`, `make sqlc`, `make lint`, `make test`, `make build`. Also `make dev` for hot-reload mode. |
| **Key Files** | `Makefile` |
| **Acceptance Criteria** | - `make up` starts Docker Compose |
| | - `make down` stops Docker Compose |
| | - `make migrate-up` runs migrations |
| | - `make lint` runs both Go and TS linters |
| | - `make test` runs both Go and TS tests |
| | - `make dev` starts with hot-reload overrides |
| | - `make test-unit` runs Go + TS unit tests |
| | - `make test-integration` runs Go integration tests with testcontainers |
| | - `make test-e2e` runs Playwright E2E tests |
| | - `make test` runs all test tiers |

### P1-T11: Set up Go test infrastructure ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T04, P1-T05 |
| **Test Types** | N/A (test infrastructure) |
| **TDD Workflow** | N/A -- this task creates the test infrastructure that enables TDD for all subsequent tasks. Verify by running a sample integration test. |
| **Description** | Create the Go test infrastructure: testcontainers-go base helpers for ephemeral PostgreSQL+TimescaleDB, Redis, and MinIO containers. Implement `TestMain` pattern in a shared `testutil` package. Create migration runner for test DB. Create fixture loading utilities. Set up `backend/testdata/` directory with a sample fixture. Define mock interfaces for `Store`, `S3Client`, `SessionStore`, `JobQueue`, `FaceitAPI`. Add integration test stage to CI pipeline. |
| **Key Files** | `backend/internal/testutil/containers.go`, `backend/internal/testutil/fixtures.go`, `backend/internal/testutil/mocks.go`, `backend/testdata/`, `.github/workflows/ci.yml` |
| **Acceptance Criteria** | - testcontainers spins up PG+TimescaleDB in `TestMain` |
| | - Migrations run automatically against test container |
| | - Sample integration test connects, inserts, and queries |
| | - Redis and MinIO test containers start and connect |
| | - Mock interfaces compile and have stub implementations |
| | - CI runs integration tests in a separate stage |
| | - `go test -tags integration ./internal/testutil/...` passes |

### P1-T12: Set up frontend test infrastructure ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T07 |
| **Test Types** | N/A (test infrastructure) |
| **TDD Workflow** | N/A -- this task creates the test infrastructure that enables TDD for all subsequent frontend tasks. Verify by running a sample component test. |
| **Description** | Create the frontend test infrastructure: configure Vitest with jsdom environment and path aliases. Create custom `renderWithProviders()` helper wrapping QueryClientProvider, test Zustand stores, and MemoryRouter. Set up MSW with `setupServer()` and base auth handler. Configure Playwright for component screenshot tests and E2E tests. Create test utility directory structure. Add Vitest stage to CI pipeline. |
| **Key Files** | `frontend/vitest.config.ts`, `frontend/src/test/setup.ts`, `frontend/src/test/render.tsx`, `frontend/src/test/msw/handlers.ts`, `frontend/src/test/msw/server.ts`, `frontend/playwright.config.ts`, `e2e/playwright.config.ts` |
| **Acceptance Criteria** | - `pnpm test` runs Vitest successfully |
| | - Sample component test renders with `renderWithProviders()` |
| | - MSW intercepts a test API call |
| | - Playwright config exists for component and E2E tests |
| | - Test utilities importable via `@/test/*` alias |
| | - CI runs `pnpm test` as a stage |
| | - `pnpm test` passes in CI |

---

## 3. Phase 2: Auth & Demo Pipeline

### P2-T01: Implement Faceit OAuth 2.0 + PKCE flow ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T04, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for PKCE code verifier/challenge generation (deterministic with seed). Write tests for OAuth URL construction with correct params. Write integration test for full callback flow with mock Faceit server. 2. GREEN: Implement oauth.go and auth handler. 3. REFACTOR: Extract PKCE logic into reusable utility. |
| **Description** | Implement the full Faceit OAuth 2.0 authorization code flow with PKCE. Generate code verifier/challenge, redirect to Faceit, handle callback, exchange code for tokens, fetch user profile from Faceit API, upsert user in DB. |
| **Key Files** | `backend/internal/auth/oauth.go`, `backend/internal/handler/auth.go`, `backend/internal/auth/oauth_test.go`, `backend/internal/handler/auth_test.go` |
| **Acceptance Criteria** | - `GET /api/v1/auth/faceit` redirects to Faceit with correct params |
| | - Callback exchanges code for access + refresh tokens |
| | - User profile fetched and upserted in `users` table |
| | - PKCE code verifier stored in Redis (short TTL) |
| | - State parameter validated to prevent CSRF |
| | - Unit tests: PKCE generation, OAuth URL params, state validation |
| | - Integration test: full callback flow with mock Faceit API |
| | - All auth tests pass in CI |

### P2-T02: Implement Redis session management ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T04, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write table-driven unit tests for SessionStore interface: Create (happy path, duplicate), Get (valid, expired, missing), Delete, Refresh (extends TTL). Write integration tests using testcontainers Redis for actual SET/GET/DEL/TTL behavior. 2. GREEN: Implement session.go. 3. REFACTOR: Extract serialization helper; clean up TTL sliding logic. |
| **Description** | Implement session creation, validation, and destruction using Redis. Session ID in a secure HttpOnly cookie. Session data includes user_id, faceit tokens, created_at, expires_at. 7-day TTL with sliding expiration. |
| **Key Files** | `backend/internal/auth/session.go`, `backend/internal/auth/session_test.go`, `backend/internal/auth/session_integration_test.go` |
| **Acceptance Criteria** | - Session created on successful OAuth callback |
| | - Session stored in Redis with `session:{token}` key |
| | - `HttpOnly`, `Secure`, `SameSite=Lax` cookie flags set |
| | - Session expires after 7 days of inactivity |
| | - `POST /api/v1/auth/logout` deletes session from Redis |
| | - Unit tests: table-driven for Create, Get (valid/expired/missing), Delete, Refresh |
| | - Integration tests: testcontainers Redis verifies actual SET/GET/DEL/TTL |
| | - All session tests pass in CI |

### P2-T03: Create auth middleware ✅

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P2-T02, P1-T11 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write httptest tests: unauthenticated request returns 401, valid session injects userID into context, expired session returns 401, auth endpoints are excluded, health endpoints are excluded. 2. GREEN: Implement middleware.go. 3. REFACTOR: Ensure middleware is composable with other chi middleware. |
| **Description** | Create chi middleware that extracts session cookie, validates against Redis, and injects user ID into request context. Returns 401 for missing/invalid sessions. Applied to all `/api/v1/*` routes except auth endpoints. |
| **Key Files** | `backend/internal/auth/middleware.go`, `backend/internal/auth/middleware_test.go` |
| **Acceptance Criteria** | - Unauthenticated requests to protected routes get 401 |
| | - Authenticated requests have `userID` in context |
| | - Auth endpoints (`/api/v1/auth/*`) are excluded |
| | - Health check endpoints are excluded |
| | - Unit tests: 401 for missing/invalid session, userID injected for valid session |
| | - Tests verify auth and health endpoint exclusions |
| | - All middleware tests pass in CI |

### P2-T04: Create AuthProvider + login page ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T07, P2-T01, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: AuthProvider shows loading state while checking session, redirects to /login when unauthenticated, renders children when authenticated. Login page renders "Sign in with Faceit" button. Callback page handles redirect. 2. GREEN: Implement components. 3. REFACTOR: Extract auth state logic into custom hook. |
| **Description** | Create a React `AuthProvider` that checks session status on mount (`GET /api/v1/auth/me`). Build login page with "Sign in with Faceit" button. Implement OAuth callback page. Redirect unauthenticated users to login. |
| **Key Files** | `frontend/src/app/(auth)/login/page.tsx`, `frontend/src/app/(auth)/callback/page.tsx`, `frontend/src/components/providers/auth-provider.tsx`, `frontend/src/components/providers/auth-provider.test.tsx`, `frontend/src/app/(auth)/login/page.test.tsx` |
| **Acceptance Criteria** | - Login page shows "Sign in with Faceit" button |
| | - Button redirects to `/api/v1/auth/faceit` |
| | - Callback page handles redirect and navigates to `/dashboard` |
| | - Unauthenticated users redirected to `/login` |
| | - AuthProvider exposes `user` and `isLoading` state |
| | - Component tests: AuthProvider loading/authenticated/unauthenticated states |
| | - Login page test: renders button, verifies href |
| | - All auth component tests pass in CI |

### P2-T05: Set up MinIO buckets and S3 client ✅

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T02, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for S3Client interface methods (PutObject, GetObject, RemoveObject, PresignedURL) with mock. Write integration tests using testcontainers MinIO. 2. GREEN: Implement s3.go. 3. REFACTOR: Add context cancellation support and retry logic. |
| **Description** | Create an init script or Go code to ensure MinIO buckets exist on startup (`demos`, `exports`). Create a Go S3 client wrapper using MinIO SDK that handles upload (PutObject), download (GetObject), delete (RemoveObject), and presigned URL generation. |
| **Key Files** | `backend/internal/storage/s3.go`, `backend/internal/storage/s3_test.go`, `backend/internal/storage/s3_integration_test.go` |
| **Acceptance Criteria** | - `demos` bucket created on first startup |
| | - Upload file to MinIO and retrieve it |
| | - Delete file from MinIO |
| | - Generate presigned download URL (15 min expiry) |
| | - Unit tests: all S3Client interface methods |
| | - Integration tests: testcontainers MinIO upload, download, delete, presigned URL |
| | - All S3 tests pass in CI |

### P2-T06: Implement demo upload endpoint ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T03, P2-T05, P1-T06, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write httptest handler tests: valid upload returns 202, invalid extension returns 400, oversized file returns 413, bad magic bytes returns 400. Write integration test for full upload → MinIO → DB → queue flow. 2. GREEN: Implement handler and service. 3. REFACTOR: Extract file validation into reusable validator. |
| **Description** | Implement `POST /api/v1/demos` multipart upload. Validate file: check `.dem` extension, magic bytes, size (max 500 MB). Stream file to MinIO. Create `demos` DB record with status `uploaded`. Enqueue parse job on Redis Streams. |
| **Key Files** | `backend/internal/handler/demo.go`, `backend/internal/demo/service.go`, `backend/internal/handler/demo_test.go`, `backend/internal/demo/service_test.go` |
| **Acceptance Criteria** | - Multipart upload works for files up to 500 MB |
| | - Invalid files rejected (wrong extension, too large, bad magic bytes) |
| | - File stored in MinIO at `demos/{user_id}/{demo_id}.dem` |
| | - Demo record created in DB with status `uploaded` |
| | - Parse job enqueued on Redis Streams |
| | - Returns 202 with demo ID and status |
| | - Unit tests: httptest for valid upload (202), invalid extension (400), oversized (413), bad magic bytes (400) |
| | - Integration test: full upload flow (file → MinIO → DB record → Redis Streams job) |
| | - All upload tests pass in CI |

### P2-T07: Set up Redis Streams job queue ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T04, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for JobQueue interface: Enqueue creates stream entry, Ack marks complete. Write integration tests using testcontainers Redis: produce → consume → ack flow, retry on failure, dead-letter after max attempts. 2. GREEN: Implement queue.go and worker.go. 3. REFACTOR: Extract consumer group setup into initialization function. |
| **Description** | Implement a Redis Streams-based job queue. Producer: `XADD` function for enqueuing jobs (parse, faceit_sync). Consumer: worker goroutine using `XREADGROUP` with consumer groups. Handle acknowledgment (`XACK`), retries, and dead-letter after max attempts. |
| **Key Files** | `backend/internal/worker/queue.go`, `backend/internal/worker/worker.go`, `backend/internal/worker/queue_test.go`, `backend/internal/worker/queue_integration_test.go` |
| **Acceptance Criteria** | - Producer can enqueue jobs with arbitrary payloads |
| | - Consumer reads jobs via consumer group |
| | - Successful jobs acknowledged (`XACK`) |
| | - Failed jobs retried up to 3 times |
| | - Dead-letter stream for permanently failed jobs |
| | - Graceful shutdown waits for in-flight jobs |
| | - Unit tests: Enqueue, Consume, Ack interface methods |
| | - Integration tests: testcontainers Redis for produce → consume → ack, retry, dead-letter |
| | - Graceful shutdown test: in-flight jobs complete before exit |
| | - All queue tests pass in CI |

### P2-T08: Implement demo parser core ✅

| | |
|---|---|
| **Complexity** | XL |
| **Deps** | P2-T05, P2-T07, P1-T06, P1-T11 |
| **Test Types** | unit, integration, golden |
| **TDD Workflow** | 1. SPIKE: Prototype demoinfocs-golang integration with a real .dem file (not TDD -- exploratory). 2. RED: Write golden file tests: parse fixture .dem → compare output to .golden.json. Write table-driven unit tests for warmup detection, overtime detection, bot filtering, tick sampling. 3. GREEN: Implement parser.go handlers. 4. REFACTOR: Extract composable event handlers; optimize memory usage. |
| **Description** | Integrate `markus-wa/demoinfocs-golang` v5 to parse CS2 `.dem` files. Register handlers for: player position (every Nth tick), kills, grenade throws/detonations, bomb plant/defuse, round start/end. Extract match metadata (map, tick rate, duration). Handle edge cases: warmup, overtime, bot players, half-time swap. Output structured data ready for DB insertion. |
| **Key Files** | `backend/internal/demo/parser.go`, `backend/internal/demo/parser_test.go`, `backend/internal/demo/parser_integration_test.go`, `backend/testdata/demos/small_match.dem`, `backend/testdata/golden/small_match_ticks.golden.json`, `backend/testdata/golden/small_match_events.golden.json` |
| **Acceptance Criteria** | - Parse a real CS2 `.dem` file without errors |
| | - Extract player positions at configurable tick interval (default every 4th tick) |
| | - Extract all kill events with attacker, victim, weapon, headshot, position |
| | - Extract grenade events (throw + detonate) with positions |
| | - Extract bomb events (plant, defuse, explode) |
| | - Extract round boundaries (start tick, end tick, winner, reason) |
| | - Handle warmup rounds (skip or flag) |
| | - Handle overtime rounds correctly |
| | - Memory usage stays under 2 GB for a 200 MB demo |
| | - Parse time < 30s for a 100 MB demo |
| | - Golden file tests: 3+ fixture demos parsed, output matches .golden.json |
| | - Unit tests: table-driven for warmup detection, overtime, bot filtering, tick sampling |
| | - Update mechanism: `go test -update` regenerates golden files |
| | - All parser tests pass in CI |

### P2-T09: Parse ticks → batch insert into TimescaleDB ✅

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P2-T08, P1-T06, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for synthetic time column computation and batch chunking logic. Write integration test: insert 500k rows into testcontainers TimescaleDB hypertable, query back by (demo_id, tick range). 2. GREEN: Implement ingest.go batch insertion. 3. REFACTOR: Optimize batch size; ensure idempotent cleanup on re-parse. |
| **Description** | Take parsed tick data from the demo parser and batch-insert into the `tick_data` hypertable. Use `COPY` protocol or multi-row INSERT for performance. Synthetic `time` column: use match_date + tick offset for hypertable partitioning. Implement in chunks to avoid OOM. |
| **Key Files** | `backend/internal/demo/ingest.go`, `backend/internal/demo/ingest_test.go`, `backend/internal/demo/ingest_integration_test.go` |
| **Acceptance Criteria** | - Tick data inserted into `tick_data` hypertable |
| | - Batch size configurable (default 10,000 rows per batch) |
| | - Synthetic `time` column correctly computed |
| | - Insertion completes in < 10s for typical demo (~500k rows) |
| | - No duplicate data on re-parse (idempotent via demo_id cleanup) |
| | - Unit tests: synthetic time computation, batch chunking boundaries |
| | - Integration test: batch insert 500k rows into testcontainers, query by demo_id + tick range |
| | - Idempotency test: re-ingest same demo without duplicates |
| | - All ingest tests pass in CI |

### P2-T10: Parse events → insert game_events ✅

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P2-T08, P1-T06, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for event-type-to-column mapping and extra_data JSONB construction. Write integration test: insert events, query by demo_id and event_type, verify JSONB fields. 2. GREEN: Implement event insertion in ingest.go. 3. REFACTOR: Extract JSONB builders per event type. |
| **Description** | Take parsed game events from the demo parser and insert into `game_events` table. Map each event type to the correct columns. Store event-specific data (headshot, penetration, flash assists, through-smoke) in the `extra_data` JSONB column. |
| **Key Files** | `backend/internal/demo/ingest.go`, `backend/internal/demo/event_ingest_test.go` |
| **Acceptance Criteria** | - All kill events inserted with correct attacker/victim/weapon |
| | - Grenade throw and detonate events stored with positions |
| | - Bomb plant and defuse events stored |
| | - `extra_data` JSONB contains event-specific metadata |
| | - Events linked to correct `round_id` |
| | - Unit tests: event type mapping, JSONB construction for kills, grenades, bombs |
| | - Integration test: insert and query events with correct attacker/victim/weapon/extra_data |
| | - Events correctly linked to round_id |
| | - All event tests pass in CI |

### P2-T11: Parse rounds → insert rounds + player_rounds ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T08, P1-T06, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for K/D/A aggregation from events, first-kill/first-death detection, clutch calculation. Write integration test: insert rounds + player_rounds, verify per-player stats match expected values from a known demo. 2. GREEN: Implement round/player_round insertion. 3. REFACTOR: Simplify stat aggregation pipeline. |
| **Description** | Insert round summaries and per-player-per-round stats. Calculate K/D/A/damage/headshot kills from parsed events. Determine first kill/death and clutch situations. |
| **Key Files** | `backend/internal/demo/ingest.go`, `backend/internal/demo/round_ingest_test.go` |
| **Acceptance Criteria** | - All rounds inserted with correct start/end ticks, winner, reason, scores |
| | - Per-player stats accurate (K/D/A/damage/HS) |
| | - First kill/death flags correctly set |
| | - Overtime rounds handled |
| | - Unit tests: K/D/A aggregation, first-kill/first-death detection, clutch calculation |
| | - Integration test: round + player_round insertion with verified stats |
| | - Overtime round handling test |
| | - All round tests pass in CI |

### P2-T12: Build demo library UI ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T04, P2-T06, P1-T12 |
| **Test Types** | component, e2e |
| **TDD Workflow** | 1. RED: Write RTL component tests: demo list renders demos with map/date/status, upload dialog accepts .dem files, status badge updates on poll, delete shows confirmation, empty state renders. Write smoke E2E test: upload demo → poll until parsed → verify in library. 2. GREEN: Implement page and components. 3. REFACTOR: Extract demo card into reusable component. |
| **Description** | Build the demo library page: list of user's demos with map name, date, status, file size. Upload button that opens file picker. Status polling for demos being parsed. Delete button with confirmation. Click to navigate to viewer. |
| **Key Files** | `frontend/src/app/(app)/demos/page.tsx`, `frontend/src/components/demos/demo-list.tsx`, `frontend/src/components/demos/upload-dialog.tsx`, `frontend/src/app/(app)/demos/page.test.tsx`, `frontend/src/components/demos/demo-list.test.tsx`, `frontend/src/components/demos/upload-dialog.test.tsx`, `e2e/tests/demo-upload.spec.ts` |
| **Acceptance Criteria** | - Demo list shows all user's demos sorted by date |
| | - Upload dialog accepts `.dem` files with progress indicator |
| | - Status badge updates when parsing completes (poll or refetch) |
| | - Delete button removes demo with confirmation |
| | - Click on ready demo navigates to `/demos/{id}` |
| | - Empty state shown when no demos exist |
| | - Component tests: list rendering, upload dialog, status polling, delete confirmation, empty state |
| | - Smoke E2E test: upload → parse → appears in library |
| | - All demo library tests pass in CI |

---

## 4. Phase 3: Core 2D Viewer

### P3-T01: Set up PixiJS Application + canvas container ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T07, P1-T12 |
| **Test Types** | unit, component |
| **TDD Workflow** | 1. RED: Write unit test for PixiJS app initialization (canvas created, render loop started). Write component test: viewer-canvas mounts, resizes with container, cleans up on unmount. 2. GREEN: Implement viewer-canvas.tsx and app.ts. 3. REFACTOR: Extract resize observer logic. |
| **Description** | Create a React component that mounts a PixiJS v8 Application. The Application instance is created in `useEffect` and attached to a container div. Implement cleanup on unmount. Set up the render loop. Bridge PixiJS state with the Zustand `viewerStore`. |
| **Key Files** | `frontend/src/components/viewer/viewer-canvas.tsx`, `frontend/src/lib/pixi/app.ts`, `frontend/src/components/viewer/viewer-canvas.test.tsx`, `frontend/src/lib/pixi/app.test.ts` |
| **Acceptance Criteria** | - PixiJS canvas renders inside the React component |
| | - Canvas resizes with container (responsive) |
| | - Application properly cleaned up on unmount |
| | - 60 FPS render loop running |
| | - Zustand store changes trigger PixiJS updates |
| | - Unit test: PixiJS app creates canvas and starts render loop |
| | - Component test: mount, resize, and unmount lifecycle |
| | - All PixiJS setup tests pass in CI |

### P3-T02: Implement map layer ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T01, P1-T12 |
| **Test Types** | unit, screenshot |
| **TDD Workflow** | 1. RED: Write unit tests for coordinate calibration: worldToPixel() with 5+ known coordinate pairs per map. Write Playwright screenshot test: render map layer with correct radar image. 2. GREEN: Implement map-layer.ts and calibration.ts. 3. REFACTOR: Type-safe calibration data structure per map. |
| **Description** | Create a PixiJS container layer that displays the correct radar image for the demo's map. Implement coordinate calibration: map world-space coordinates (from demo) to pixel-space on the radar image. Store calibration data per map (origin, scale). |
| **Key Files** | `frontend/src/lib/pixi/layers/map-layer.ts`, `frontend/src/lib/maps/calibration.ts`, `frontend/public/maps/*.png`, `frontend/src/lib/maps/calibration.test.ts`, `frontend/src/lib/pixi/layers/map-layer.test.ts` |
| **Acceptance Criteria** | - Radar image loads for the demo's map |
| | - World-space coordinate (0,0) maps to correct pixel position |
| | - All Active Duty maps have calibration data |
| | - Radar images included in `public/maps/` |
| | - Map layer is the bottom-most layer |
| | - Unit tests: worldToPixel() verified with 5+ known pairs per Active Duty map |
| | - Screenshot test: radar image renders at correct scale |
| | - All map layer tests pass in CI |

### P3-T03: Implement tick data fetching ✅

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P2-T09, P3-T01, P1-T12 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write unit tests for tick buffer: fetch triggers for correct ranges, look-ahead pre-fetches next chunk, seek flushes and re-fetches, cached ranges not re-fetched, out-of-order responses handled. 2. GREEN: Implement tick-buffer.ts and use-tick-data.ts with MSW for API mocking. 3. REFACTOR: Optimize cache eviction for memory management. |
| **Description** | Create an API client and data buffer for tick data. Fetch tick data in chunks from `GET /api/v1/demos/:id/ticks`. Implement a look-ahead buffer that pre-fetches upcoming ticks during playback. Cache already-fetched ranges. Handle seek (flush and re-fetch for the target range). |
| **Key Files** | `frontend/src/lib/pixi/tick-buffer.ts`, `frontend/src/hooks/use-tick-data.ts`, `frontend/src/lib/pixi/tick-buffer.test.ts`, `frontend/src/hooks/use-tick-data.test.ts` |
| **Acceptance Criteria** | - Tick data fetched for current playback range |
| | - Look-ahead buffer pre-fetches next chunk during playback |
| | - Seek to a new position triggers fetch for that range |
| | - Cached ranges are not re-fetched |
| | - Buffer handles out-of-order responses |
| | - Network errors don't crash playback |
| | - Unit tests: fetch range calculation, look-ahead logic, seek behavior, cache hit/miss |
| | - Hook test with MSW: fetches data, caches, handles errors |
| | - All tick data tests pass in CI |

### P3-T04: Implement player layer

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T02, P3-T03, P1-T12 |
| **Test Types** | unit, screenshot |
| **TDD Workflow** | 1. RED: Write unit tests for player sprite logic: team color selection (CT=blue, T=orange), worldToPixel position transform, view angle rotation math (0°/90°/180°/270°), visibility states (alive/dead/selected/faded). Write Playwright screenshot test: render 10 players with mock tick data. 2. GREEN: Implement player-layer.ts and player.ts. 3. REFACTOR: Object pool for player sprites; separate data transform from draw calls. |
| **Description** | Create a PixiJS layer that renders players. Each player: colored circle (CT blue, T orange), name label, view-angle indicator (small cone or line). Dead players shown faded with death X marker. Selected player highlighted. Update positions every tick from the tick buffer. |
| **Key Files** | `frontend/src/lib/pixi/layers/player-layer.ts`, `frontend/src/lib/pixi/sprites/player.ts`, `frontend/src/lib/pixi/layers/player-layer.test.ts`, `frontend/src/lib/pixi/sprites/player.test.ts` |
| **Acceptance Criteria** | - 10 players rendered at correct positions |
| | - Team colors distinguish CT and T |
| | - Name labels readable and non-overlapping |
| | - View angle indicator shows correct direction |
| | - Dead players faded; death X at kill location |
| | - Click on player selects and highlights them |
| | - Unit tests: team color, coordinate transform (5+ pairs), view angle rotation, visibility states |
| | - Screenshot test: 10 players rendered with correct positions and colors |
| | - All player layer tests pass in CI |

### P3-T05: Implement event layer

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P3-T02, P3-T03, P1-T12 |
| **Test Types** | unit, screenshot |
| **TDD Workflow** | 1. RED: Write unit tests for event timing logic: smoke duration calculation, flash decay timing, grenade effect radius. Write tests for kill line geometry (start/end positions). Write Playwright screenshot test: render event layer with multiple concurrent effects. 2. GREEN: Implement event-layer.ts and effects.ts. 3. REFACTOR: Pool effect sprites for performance. |
| **Description** | Create a PixiJS layer for game events. Kill: line from killer to victim + X marker. HE grenade: expanding red circle. Smoke: gray circle with fade-in/out. Flash: yellow flash. Molotov: orange area. Bomb: flashing icon at plant site. Events rendered at the correct ticks with appropriate durations. |
| **Key Files** | `frontend/src/lib/pixi/layers/event-layer.ts`, `frontend/src/lib/pixi/sprites/effects.ts`, `frontend/src/lib/pixi/layers/event-layer.test.ts`, `frontend/src/lib/pixi/sprites/effects.test.ts` |
| **Acceptance Criteria** | - Kill lines drawn at the correct tick |
| | - Smoke circles appear, persist, and fade matching game duration |
| | - HE, flash, molotov effects render at correct positions |
| | - Bomb plant/defuse icons display correctly |
| | - Events from `game_events` table rendered at correct positions |
| | - Performance: 60 FPS maintained with many concurrent effects |
| | - Unit tests: smoke duration, flash decay, grenade radius, kill line geometry |
| | - Screenshot test: concurrent effects (smoke + kill + flash) render correctly |
| | - All event layer tests pass in CI |

### P3-T06: Implement playback engine

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P3-T03, P1-T12 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write unit tests for playback engine: tick advancement at each speed (0.25x/0.5x/1x/2x/4x), pause freezes tick, speed change is instant, seek jumps to exact tick, interpolation between sampled ticks returns correct intermediate positions. 2. GREEN: Implement playback-engine.ts. 3. REFACTOR: Separate time management from tick interpolation logic. |
| **Description** | Create the playback engine that advances the current tick based on elapsed time and playback speed. Handle play/pause, speed changes (0.25x-4x), and seek. Manage the relationship between real time and game time. Tick interpolation for smooth player movement between sampled ticks. |
| **Key Files** | `frontend/src/lib/pixi/playback-engine.ts`, `frontend/src/lib/pixi/playback-engine.test.ts` |
| **Acceptance Criteria** | - Play advances ticks at correct rate for each speed |
| | - Pause freezes on current tick |
| | - Speed change is instant and smooth |
| | - Seek jumps to exact tick and resumes |
| | - Interpolation smooths movement between sampled ticks |
| | - Playback pauses at round end if auto-pause enabled |
| | - Unit tests: tick rate at each speed, pause/resume, seek, interpolation math |
| | - Speed change test: verify no tick discontinuity |
| | - Round boundary test: auto-pause at round end |
| | - All playback engine tests pass in CI |

### P3-T07: Build playback controls UI

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T06, P1-T08, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: PlaybackControls renders play/pause button, click toggles playback in viewerStore. Speed selector shows all options and dispatches to store. Timeline renders with correct range. Tick counter displays current/total. Round boundary markers render at correct positions. 2. GREEN: Implement components. 3. REFACTOR: Extract timeline math into pure utility functions. |
| **Description** | Build the playback control bar: play/pause button, speed selector dropdown, timeline scrubber (range input or custom), tick counter display, total duration. Timeline shows round boundaries as markers. Controls sync with viewerStore. |
| **Key Files** | `frontend/src/components/viewer/playback-controls.tsx`, `frontend/src/components/viewer/timeline.tsx`, `frontend/src/components/viewer/playback-controls.test.tsx`, `frontend/src/components/viewer/timeline.test.tsx`, `frontend/src/lib/pixi/timeline-utils.test.ts` |
| **Acceptance Criteria** | - Play/pause button toggles playback |
| | - Speed selector shows 0.25x, 0.5x, 1x, 2x, 4x options |
| | - Timeline scrubber allows seeking to any tick |
| | - Round boundaries shown as tick marks on timeline |
| | - Current tick / total ticks displayed |
| | - Controls are responsive (work at various widths) |
| | - Component tests: play/pause toggle, speed selector dispatch, timeline seek, tick counter |
| | - Unit tests: tick-to-pixel and pixel-to-tick conversions, round marker positions |
| | - All playback control tests pass in CI |

### P3-T08: Implement round selector

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P3-T06, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: round list renders all rounds with scores, click dispatches seek action to viewerStore, current round highlighted, win reason displayed. 2. GREEN: Implement round-selector.tsx. 3. REFACTOR: Memoize round list to avoid re-renders. |
| **Description** | Build a round selector that lists all rounds with their scores. Clicking a round seeks playback to that round's start tick. Highlight the currently playing round. Show round winner (CT/T) and win reason. |
| **Key Files** | `frontend/src/components/viewer/round-selector.tsx`, `frontend/src/components/viewer/round-selector.test.tsx` |
| **Acceptance Criteria** | - All rounds listed with round number and score |
| | - Click jumps to round start tick |
| | - Current round highlighted |
| | - Win reason icon or label shown |
| | - Component tests: renders all rounds, click seeks, current round highlighted, win reason shown |
| | - All round selector tests pass in CI |

### P3-T09: Implement zoom and pan

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T02, P1-T12 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write unit tests for camera logic: zoom-to-point math (zoom centers on cursor), pan offset clamping (can't pan beyond map bounds), zoom range enforcement (0.5x-4x), reset-view restores defaults. Write test for mini-map viewport rectangle calculation. 2. GREEN: Implement camera.ts and mini-map. 3. REFACTOR: Extract zoom math into pure functions. |
| **Description** | Add zoom (scroll wheel) and pan (click-drag) to the PixiJS canvas. Zoom range: 0.5x to 4x. Implement a mini-map in the corner showing the full map with a viewport rectangle. Add a reset-view button. |
| **Key Files** | `frontend/src/lib/pixi/camera.ts`, `frontend/src/components/viewer/mini-map.tsx`, `frontend/src/lib/pixi/camera.test.ts`, `frontend/src/components/viewer/mini-map.test.tsx` |
| **Acceptance Criteria** | - Scroll-to-zoom works smoothly (0.5x to 4x) |
| | - Click-and-drag pans the view |
| | - Mini-map shows viewport position |
| | - Reset-view button restores default zoom and position |
| | - Zoom centers on mouse cursor position |
| | - Unit tests: zoom-to-point math, pan clamping, zoom range, reset-view |
| | - Mini-map viewport rectangle calculated correctly at different zoom levels |
| | - All camera tests pass in CI |

### P3-T10: Build scoreboard overlay

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T04, P2-T11, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: scoreboard renders per-player stats (K/D/A/ADR/HS%/KAST/Rating), teams visually separated (CT/T), current round highlighted, toggle shows/hides. Write unit tests for stat aggregation (sum kills across rounds). 2. GREEN: Implement scoreboard.tsx. 3. REFACTOR: Extract stat calculation into pure utility. |
| **Description** | Build a toggle-able scoreboard overlay showing per-player stats for the current round and match totals. Columns: Player, K, D, A, ADR, HS%, KAST, Rating. Data from `player_rounds` table. Highlight the current round in the round-by-round view. |
| **Key Files** | `frontend/src/components/viewer/scoreboard.tsx`, `frontend/src/components/viewer/scoreboard.test.tsx` |
| **Acceptance Criteria** | - Scoreboard toggles with Tab key or button |
| | - Per-player stats accurate for the current round |
| | - Match totals calculated correctly |
| | - CT and T teams visually separated |
| | - Current round row highlighted |
| | - Scoreboard doesn't block critical map areas (positioned on edge or semi-transparent) |
| | - Component tests: per-player stats render, team separation, round highlight, toggle |
| | - Unit tests: stat aggregation math (ADR, KAST, Rating) |
| | - All scoreboard tests pass in CI |

### P3-T11: Implement keyboard shortcuts

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P3-T07, P1-T12 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write tests for useViewerShortcuts hook: Space dispatches togglePlay, Left/Right dispatches skip, Up/Down dispatches speed change, Tab toggles scoreboard, number keys select player. Write test that shortcuts don't fire when input is focused. 2. GREEN: Implement hook. 3. REFACTOR: Extract key mapping to config object. |
| **Description** | Add keyboard shortcuts for the 2D viewer: Space (play/pause), Left/Right arrows (skip 5 seconds), Up/Down arrows (increase/decrease speed), Tab (scoreboard toggle), Escape (deselect/close overlays), number keys 1-9 (select player). |
| **Key Files** | `frontend/src/hooks/use-viewer-shortcuts.ts`, `frontend/src/hooks/use-viewer-shortcuts.test.ts` |
| **Acceptance Criteria** | - All shortcuts work when viewer is focused |
| | - Shortcuts don't fire when typing in an input field |
| | - Shortcuts match the key mapping listed in the UI |
| | - Unit tests: all shortcut keys dispatch correct store actions |
| | - Input focus test: shortcuts suppressed when typing in text field |
| | - All shortcut tests pass in CI |

### P3-T12: Connect viewer Zustand store to PixiJS

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T01, P1-T08, P1-T12 |
| **Test Types** | unit, e2e |
| **TDD Workflow** | 1. RED: Write unit tests for store bridge: Zustand speed change triggers PixiJS update, PixiJS click updates Zustand selected player, current tick syncs bidirectionally. Write smoke E2E test: open parsed demo, canvas renders, play/pause works. 2. GREEN: Implement store-bridge.ts. 3. REFACTOR: Optimize subscription selectors to minimize unnecessary updates. |
| **Description** | Implement the bridge between React (Zustand) and PixiJS. Zustand store changes (speed, selected player, show/hide layers) trigger PixiJS updates. PixiJS events (click on player, current tick) update Zustand. Use Zustand `subscribe` for non-React listeners. |
| **Key Files** | `frontend/src/lib/pixi/store-bridge.ts`, `frontend/src/lib/pixi/store-bridge.test.ts`, `e2e/tests/demo-viewer.spec.ts` |
| **Acceptance Criteria** | - Changing speed in React UI immediately affects PixiJS playback |
| | - Clicking a player in PixiJS updates the selected player in React |
| | - Current tick in PixiJS reflected in React timeline |
| | - No unnecessary re-renders (selector-based subscriptions) |
| | - Unit tests: bidirectional sync between Zustand and PixiJS for speed, selection, tick |
| | - No unnecessary re-render test: verify selector-based subscriptions |
| | - Smoke E2E test: open demo, canvas renders, play/pause toggles |
| | - All store bridge tests pass in CI |

---

## 5. Phase 4: Faceit & Heatmaps

### P4-T01: Implement Faceit API client ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T04, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for Faceit client with mock HTTP server: GetProfile returns parsed profile, GetMatches handles pagination, rate limit 429 triggers backoff. Write integration tests for Redis caching (cache hit returns cached, cache miss fetches). 2. GREEN: Implement client.go. 3. REFACTOR: Extract HTTP retry/backoff into shared utility. |
| **Description** | Create a Go HTTP client for the Faceit Data API. Endpoints: player profile, match history (paginated), match details. Handle rate limiting (Faceit API limits). Cache responses in Redis with appropriate TTLs. Use the user's Faceit access token from session. |
| **Key Files** | `backend/internal/faceit/client.go`, `backend/internal/faceit/client_test.go`, `backend/internal/faceit/client_integration_test.go` |
| **Acceptance Criteria** | - Fetch player profile by Faceit ID |
| | - Fetch match history with pagination |
| | - Responses cached in Redis (profile: 15m, matches: 5m) |
| | - Rate limiting respected (back-off on 429) |
| | - Error handling for API failures |
| | - Unit tests: GetProfile, GetMatches pagination, 429 backoff, error handling |
| | - Integration tests: Redis caching (hit/miss/TTL expiry) |
| | - All Faceit client tests pass in CI |

### P4-T02: Implement Faceit sync worker job ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P4-T01, P2-T07, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for sync logic: new matches inserted, duplicates skipped (idempotent), ELO delta calculated from consecutive matches. Write integration test: full sync with mock Faceit API → DB assertions. 2. GREEN: Implement sync.go. 3. REFACTOR: Separate match fetching from DB insertion. |
| **Description** | Create a worker job that syncs a user's Faceit match history. Triggered on login and via manual sync endpoint. Fetches recent matches, calculates ELO deltas, inserts into `faceit_matches` table. Skips already-synced matches (idempotent via `faceit_match_id` unique constraint). |
| **Key Files** | `backend/internal/faceit/sync.go`, `backend/internal/worker/faceit_handler.go`, `backend/internal/faceit/sync_test.go`, `backend/internal/worker/faceit_handler_test.go` |
| **Acceptance Criteria** | - New matches inserted into `faceit_matches` |
| | - Duplicate matches skipped (upsert or check) |
| | - ELO before/after calculated from consecutive matches |
| | - Job completes in < 10s for initial sync (last 100 matches) |
| | - Demo URLs extracted and stored |
| | - Unit tests: match insertion, duplicate skip, ELO delta calculation |
| | - Integration test: full sync flow with mock API and real DB |
| | - All sync tests pass in CI |

### P4-T03: Build Faceit dashboard page

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P4-T01, P2-T04, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: profile card renders avatar/nickname/level/ELO/country. ELO chart renders with data points. Time range selector filters data. Loading and error states displayed. 2. GREEN: Implement dashboard page and components. 3. REFACTOR: Extract chart config into reusable component. |
| **Description** | Build the Faceit dashboard page. Profile card: avatar, nickname, level badge, ELO, country flag. ELO history chart: line chart using a charting library (Recharts or similar). Time range selector (30, 90, 180 days, all time). Current streak indicator. |
| **Key Files** | `frontend/src/app/(app)/dashboard/page.tsx`, `frontend/src/components/dashboard/profile-card.tsx`, `frontend/src/components/dashboard/elo-chart.tsx`, `frontend/src/app/(app)/dashboard/page.test.tsx`, `frontend/src/components/dashboard/profile-card.test.tsx`, `frontend/src/components/dashboard/elo-chart.test.tsx` |
| **Acceptance Criteria** | - Profile card displays correct Faceit data |
| | - ELO chart renders with data points from API |
| | - Time range selector filters chart data |
| | - Hover on chart shows exact ELO + date |
| | - Loading and error states handled |
| | - Component tests: profile card data, ELO chart rendering, time range filter, loading/error states |
| | - All dashboard tests pass in CI |

### P4-T04: Build match history list

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P4-T02, P2-T04, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: match list renders paginated entries with map/score/K:D:A/ELO change. ELO change color-coded (green gain, red loss). Map and result filters work. "Import Demo" button shown for matches without demos. 2. GREEN: Implement match-list.tsx. 3. REFACTOR: Extract match card into reusable component. |
| **Description** | Build a paginated match history list. Each entry: map icon, score, K/D/A, ELO change (+/- with color), date. Win/loss color coding. Filter by map and result. Click opens demo in viewer (if imported) or offers to import. |
| **Key Files** | `frontend/src/components/dashboard/match-list.tsx`, `frontend/src/components/dashboard/match-list.test.tsx` |
| **Acceptance Criteria** | - Match list paginated (20 per page) |
| | - ELO change color-coded (green for gain, red for loss) |
| | - Map and result filters work |
| | - Click on match with imported demo navigates to viewer |
| | - "Import Demo" button for matches without imported demos |
| | - Component tests: pagination, ELO color coding, filters, import button, demo link |
| | - All match list tests pass in CI |

### P4-T05: Implement demo auto-import from Faceit

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P4-T02, P2-T06, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for import logic: download from URL, upload to MinIO, create demo record, link faceit_match.demo_id. Write integration test: full import flow. Write test for failure handling (download failure doesn't block sync). 2. GREEN: Implement demo_import.go. 3. REFACTOR: Add configurable auto-import settings. |
| **Description** | Extend the Faceit sync job to optionally download demo files from Faceit match demo URLs. Download to a temp file, then upload to MinIO and create a demo record. Enqueue parse job. Configurable: auto-import last N matches, or manual per-match. |
| **Key Files** | `backend/internal/faceit/demo_import.go`, `backend/internal/faceit/demo_import_test.go`, `backend/internal/faceit/demo_import_integration_test.go` |
| **Acceptance Criteria** | - Demo downloaded from Faceit demo URL |
| | - File uploaded to MinIO, demo record created |
| | - Parse job enqueued |
| | - `faceit_matches.demo_id` linked to imported demo |
| | - Failed downloads don't block sync job |
| | - Manual import via `POST /api/v1/faceit/matches/:id/import` works |
| | - Unit tests: download, upload, record creation, linking |
| | - Integration test: full import flow with testcontainers |
| | - Failure isolation test: failed download doesn't block sync |
| | - All import tests pass in CI |

### P4-T06: Implement heatmap data endpoint

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T09, P2-T10, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for filter construction (side, weapon, player). Write integration test: seed game_events for 3 demos, query aggregate endpoint, verify correct position data returned. 2. GREEN: Implement heatmap handler and query. 3. REFACTOR: Optimize aggregation query with EXPLAIN ANALYZE. |
| **Description** | Implement `POST /api/v1/heatmaps/aggregate` that queries game events across one or more demos and returns position data points for heatmap rendering. Filters: side, weapon category, player. Returns `{x, y, intensity}` data points. For single demo, query `game_events`. For multi-demo, aggregate across selected demo IDs. |
| **Key Files** | `backend/internal/handler/heatmap.go`, `backend/internal/demo/heatmap.go`, `backend/internal/handler/heatmap_test.go`, `backend/internal/demo/heatmap_test.go`, `backend/internal/demo/heatmap_integration_test.go` |
| **Acceptance Criteria** | - Single-demo heatmap returns kill positions |
| | - Multi-demo aggregation combines positions across demos |
| | - Filters correctly applied (side, weapon, player) |
| | - Response includes normalized intensity values |
| | - Query performance < 2s for 10 demos |
| | - Unit tests: filter construction for side/weapon/player combinations |
| | - Integration test: multi-demo aggregation with seeded events |
| | - Query performance test: < 2s for 10-demo aggregate |
| | - All heatmap tests pass in CI |

### P4-T07: Implement client-side KDE rendering

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P4-T06, P3-T02, P1-T12 |
| **Test Types** | unit, screenshot |
| **TDD Workflow** | 1. RED: Write unit tests for KDE algorithm: known input points produce expected density grid values. Test bandwidth parameter affects spread. Test color scale mapping (intensity → RGBA). Write Playwright screenshot test: render heatmap overlay on map. 2. GREEN: Implement kde.ts and heatmap-layer.ts. 3. REFACTOR: Optimize KDE with spatial indexing for large datasets. |
| **Description** | Implement Kernel Density Estimation on the client and render as a PixiJS heatmap overlay on the map layer. Take data points from the heatmap API and compute KDE on a grid. Render as a color gradient texture (transparent → yellow → red). Configurable bandwidth and opacity. |
| **Key Files** | `frontend/src/lib/pixi/layers/heatmap-layer.ts`, `frontend/src/lib/heatmap/kde.ts`, `frontend/src/lib/heatmap/kde.test.ts`, `frontend/src/lib/pixi/layers/heatmap-layer.test.ts` |
| **Acceptance Criteria** | - KDE computed correctly for input data points |
| | - Heatmap renders as color gradient overlay on map |
| | - Color scale: transparent (low) → yellow → red (high) |
| | - Renders in < 2s for single demo data |
| | - Renders in < 5s for 10-demo aggregate |
| | - Opacity adjustable via slider |
| | - Unit tests: KDE density values for known inputs, bandwidth effect, color scale mapping |
| | - Screenshot test: heatmap overlay renders on map with correct color gradient |
| | - All KDE tests pass in CI |

### P4-T08: Build heatmap filter controls

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P4-T07, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: filter controls render map/side/weapon/player options. Changing a filter dispatches refetch. Multi-demo selector allows picking demos. Reset button clears all filters. Active filters shown in UI. 2. GREEN: Implement heatmap-filters.tsx and pages. 3. REFACTOR: Extract filter state into URL search params. |
| **Description** | Build filter controls for heatmaps: map selector, side (CT/T/Both), weapon category (rifle, pistol, SMG, sniper, shotgun), player (from demo participants). Demo selector for multi-demo mode. Filters trigger re-fetch and re-render. |
| **Key Files** | `frontend/src/components/heatmap/heatmap-filters.tsx`, `frontend/src/app/(app)/demos/[demoId]/heatmap/page.tsx`, `frontend/src/app/(app)/heatmaps/page.tsx`, `frontend/src/components/heatmap/heatmap-filters.test.tsx` |
| **Acceptance Criteria** | - All filter options populated from available data |
| | - Changing a filter updates the heatmap in real-time |
| | - Multi-demo selector allows picking multiple demos |
| | - Active filters clearly shown in UI |
| | - Reset filters button |
| | - Component tests: filter options render, filter changes trigger refetch, reset clears, active filters shown |
| | - All heatmap filter tests pass in CI |

### P4-T09: Build per-demo stats view

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T11, P2-T12, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: stats panel renders per-player K/D/A/ADR/HS%/KAST/Rating. CT/T side breakdown shown. Weapon kill distribution chart renders. Stats match mock data. 2. GREEN: Implement stats-panel.tsx. 3. REFACTOR: Extract stat calculation functions into testable utilities. |
| **Description** | Build a stats tab within the demo detail page. Show per-player stats: K/D/A, ADR, HS%, KAST (estimated), Rating. Break down by round half (CT side / T side). Weapon kill distribution bar chart. |
| **Key Files** | `frontend/src/components/viewer/stats-panel.tsx`, `frontend/src/components/viewer/stats-panel.test.tsx` |
| **Acceptance Criteria** | - All player stats displayed accurately |
| | - CT/T side breakdown shown |
| | - Weapon kill distribution chart renders |
| | - Stats match what the scoreboard shows |
| | - Component tests: per-player stats render accurately, side breakdown shown, weapon chart renders |
| | - All stats panel tests pass in CI |

### P4-T10: Build cross-demo trend charts

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P4-T09, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: trend charts render line data for K/D, win rate, ADR, HS%. Rolling average computed correctly. Map breakdown shows per-map stats. Date range selector filters data. 2. GREEN: Implement trend-charts.tsx. 3. REFACTOR: Extract rolling average calculation into utility. |
| **Description** | Build a trends page showing stats across multiple demos. Line charts for: K/D ratio, win rate (rolling), ADR, HS%. Map-specific performance breakdown. Best/worst maps identification. Use Recharts or similar. |
| **Key Files** | `frontend/src/app/(app)/heatmaps/page.tsx` (trends tab), `frontend/src/components/analytics/trend-charts.tsx`, `frontend/src/components/analytics/trend-charts.test.tsx` |
| **Acceptance Criteria** | - Line charts render with correct demo-by-demo data points |
| | - Rolling averages computed (configurable window) |
| | - Map breakdown shows per-map stats |
| | - Date range selector works |
| | - Loading states shown while computing |
| | - Component tests: line charts render, rolling averages correct, map breakdown, date filter |
| | - Unit tests: rolling average calculation with known inputs |
| | - All trend chart tests pass in CI |

---

## 6. Phase 5: Strategy Board & Lineups

### P5-T01: Implement WebSocket server ✅

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P1-T04, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for hub logic: client register/unregister, message broadcast to room (exclude sender), room cleanup on last disconnect. Write integration test: two gorilla/websocket test clients connect, one sends, other receives. 2. GREEN: Implement server.go, hub.go, client.go. 3. REFACTOR: Extract room management into its own type. |
| **Description** | Implement the Go WebSocket server using gorilla/websocket. Room management: clients grouped by strat board ID. Authentication via session cookie on upgrade. Connection lifecycle: register, unregister, broadcast. Redis PubSub for cross-instance broadcast (future multi-instance). |
| **Key Files** | `backend/internal/websocket/server.go`, `backend/internal/websocket/hub.go`, `backend/internal/websocket/client.go`, `backend/internal/websocket/hub_test.go`, `backend/internal/websocket/server_integration_test.go` |
| **Acceptance Criteria** | - WebSocket connections established at `/ws/strat/:id` |
| | - Auth validated on upgrade (401 for invalid session) |
| | - Clients grouped into rooms by board ID |
| | - Messages broadcast to all other clients in the same room |
| | - Clean disconnect handling (remove from room) |
| | - Redis PubSub channel per board for future scaling |
| | - Unit tests: register, unregister, broadcast, room cleanup |
| | - Integration test: two test clients connect and exchange messages |
| | - Auth test: 401 returned for invalid session on upgrade |
| | - All WebSocket tests pass in CI |

### P5-T02: Implement Yjs relay protocol in Go ✅

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P5-T01, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for message routing: sync messages (type 0) relayed, awareness messages (type 1) relayed, unknown types dropped. Write integration test: initial state loaded from DB on first connect, state persisted on last disconnect, auto-save triggers every 30s. 2. GREEN: Implement yjs_relay.go. 3. REFACTOR: Separate persistence logic from relay logic. |
| **Description** | Implement the Yjs WebSocket sync and awareness protocol on the Go server. The server acts as a "dumb relay": it doesn't parse Yjs binary messages, just routes them. On first client connect to a room: load Yjs state from PostgreSQL and send as initial sync. On last client disconnect: persist current state. Periodic auto-save every 30s while clients are connected. |
| **Key Files** | `backend/internal/websocket/yjs_relay.go`, `backend/internal/websocket/yjs_relay_test.go`, `backend/internal/websocket/yjs_relay_integration_test.go` |
| **Acceptance Criteria** | - Yjs sync messages (type 0) relayed between clients |
| | - Yjs awareness messages (type 1) relayed between clients |
| | - Initial state loaded from DB on first connection |
| | - State persisted on last disconnect |
| | - Periodic auto-save while active |
| | - Binary messages passed through without parsing |
| | - Unit tests: message type routing (sync, awareness, unknown) |
| | - Integration test: state load from DB, state persist on disconnect, auto-save |
| | - Binary passthrough test: messages relayed without modification |
| | - All Yjs relay tests pass in CI |

### P5-T03: Set up Yjs client ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T01, P1-T07, P1-T12 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests with two in-memory Y.Doc instances: set value on doc1, apply update to doc2, assert convergence. Test shared types (Y.Map, Y.Array). Test awareness: set cursor on provider1, verify provider2 receives it. Write integration test: two clients connect to WS server, sync state. 2. GREEN: Implement doc.ts, provider.ts, awareness.ts. 3. REFACTOR: Extract shared type definitions into typed helpers. |
| **Description** | Set up the Yjs client in the frontend. Create a Yjs `Doc`, connect via `WebsocketProvider` to `/ws/strat/:id`. Set up Awareness protocol for cursor/user presence. Create shared types: `Y.Map` for board settings, `Y.Array` for drawing elements, `Y.Map` for each element's properties. |
| **Key Files** | `frontend/src/lib/yjs/doc.ts`, `frontend/src/lib/yjs/provider.ts`, `frontend/src/lib/yjs/awareness.ts`, `frontend/src/lib/yjs/doc.test.ts`, `frontend/src/lib/yjs/provider.test.ts`, `frontend/src/lib/yjs/awareness.test.ts` |
| **Acceptance Criteria** | - Yjs Doc connects to WS server |
| | - State syncs between two browser tabs |
| | - Awareness shows other users' cursor positions |
| | - Reconnection works after temporary disconnect |
| | - Shared types structured for drawing elements |
| | - Unit tests: two Y.Doc instances sync map/array mutations, awareness propagates |
| | - Integration test: two clients connect to WS, verify real-time sync |
| | - Reconnection test: state reconciles after disconnect/reconnect |
| | - All Yjs client tests pass in CI |

### P5-T04: Implement drawing canvas

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P5-T03, P3-T02, P1-T12 |
| **Test Types** | unit, screenshot |
| **TDD Workflow** | 1. RED: Write unit tests for renderer: Yjs document changes trigger re-render of affected elements. Write tests for element-to-sprite conversion. Write Playwright screenshot test: render canvas with map background and 10 drawing elements. 2. GREEN: Implement strat-canvas.tsx and renderer.ts. 3. REFACTOR: Batch Yjs observer updates to reduce render calls. |
| **Description** | Create the strategy board canvas with the map as background. Use PixiJS or Canvas 2D (whichever integrates better with Yjs). Drawing elements stored in Yjs shared types. Render all elements from the Yjs document. Observe Yjs changes and re-render affected elements. |
| **Key Files** | `frontend/src/components/strat/strat-canvas.tsx`, `frontend/src/lib/strat/renderer.ts`, `frontend/src/components/strat/strat-canvas.test.tsx`, `frontend/src/lib/strat/renderer.test.ts` |
| **Acceptance Criteria** | - Map background renders correctly |
| | - Drawing elements from Yjs document rendered on canvas |
| | - Changes to Yjs doc automatically update the canvas |
| | - Canvas supports pan and zoom (reuse from P3-T09) |
| | - Performance: smooth rendering with 100+ elements |
| | - Unit tests: Yjs change triggers re-render, element-to-sprite conversion |
| | - Screenshot test: canvas with map + 10 drawing elements |
| | - All drawing canvas tests pass in CI |

### P5-T05: Implement drawing tools

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P5-T04, P1-T12 |
| **Test Types** | unit, screenshot |
| **TDD Workflow** | 1. RED: Write unit tests for each tool: freehand creates path points, line creates start/end, arrow creates line + head, rect/circle create correct bounds. Write tests for color picker state and line thickness. Write test that eraser removes elements from Yjs doc. Write Playwright screenshot test: canvas with one of each element type. 2. GREEN: Implement tools and toolbar. 3. REFACTOR: Extract common tool base class. |
| **Description** | Implement drawing tools: freehand (path), straight line, arrow, rectangle, circle, text label. Each tool creates elements in the Yjs shared type. Tool selector UI (toolbar). Color picker (preset team colors + custom). Line thickness selector. Eraser tool (removes elements by click). |
| **Key Files** | `frontend/src/components/strat/toolbar.tsx`, `frontend/src/lib/strat/tools/*.ts`, `frontend/src/lib/strat/tools/freehand.test.ts`, `frontend/src/lib/strat/tools/shapes.test.ts`, `frontend/src/components/strat/toolbar.test.tsx` |
| **Acceptance Criteria** | - Freehand drawing creates smooth paths |
| | - Line, arrow, rect, circle draw correctly |
| | - Text tool allows typing a label |
| | - Color picker works (presets + custom) |
| | - Line thickness adjustable |
| | - Eraser removes clicked elements |
| | - All drawing operations are Yjs mutations (synced to others) |
| | - Unit tests: each tool creates correct Yjs mutations (freehand, line, arrow, rect, circle, text) |
| | - Eraser test: removes element from Yjs doc |
| | - Screenshot test: canvas with all element types rendered |
| | - All drawing tool tests pass in CI |

### P5-T06: Implement strategy primitives

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T04, P1-T12 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write unit tests: player token creation (5 CT blue, 5 T orange, labeled), token drag updates position in Yjs doc. Grenade markers placed at correct positions. Timing waypoints numbered sequentially and renumber on delete. 2. GREEN: Implement primitives. 3. REFACTOR: Shared base class for draggable primitives. |
| **Description** | Create specialized strategy elements: player tokens (CT1-CT5, T1-T5) that are draggable and labeled, grenade trajectory lines with arc indicators, smoke/flash/molotov/HE markers, and numbered timing waypoints for execute order. |
| **Key Files** | `frontend/src/lib/strat/primitives/*.ts`, `frontend/src/lib/strat/primitives/player-tokens.test.ts`, `frontend/src/lib/strat/primitives/grenade-markers.test.ts`, `frontend/src/lib/strat/primitives/waypoints.test.ts` |
| **Acceptance Criteria** | - Player tokens draggable with labels |
| | - 5 CT tokens (blue) and 5 T tokens (orange) |
| | - Grenade markers placed on map |
| | - Timing waypoints numbered sequentially |
| | - All primitives are Yjs-synced elements |
| | - Unit tests: token creation (CT/T count, colors, labels), drag updates Yjs, markers, waypoint numbering |
| | - All primitive tests pass in CI |

### P5-T07: Implement undo/redo

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T05, P1-T12 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write unit tests: undo reverts last Yjs mutation, redo restores it. Undo only affects current user's changes (not collaborator's). Undo stack survives tool switches. Toolbar buttons reflect undo/redo availability (disabled when empty). 2. GREEN: Implement undo-manager.ts. 3. REFACTOR: Scope UndoManager to tracked origins. |
| **Description** | Integrate Yjs `UndoManager` for undo/redo on the strategy board. Scope to the current user's changes (don't undo others' work). Ctrl+Z / Ctrl+Shift+Z keyboard shortcuts. Undo/redo buttons in toolbar. |
| **Key Files** | `frontend/src/lib/strat/undo-manager.ts`, `frontend/src/lib/strat/undo-manager.test.ts` |
| **Acceptance Criteria** | - Ctrl+Z undoes the user's last action |
| | - Ctrl+Shift+Z redoes |
| | - Only the current user's changes are undone (not collaborators') |
| | - Undo stack survives tool switches |
| | - Toolbar buttons reflect undo/redo availability |
| | - Unit tests: undo reverts, redo restores, scoped to current user, survives tool switch |
| | - Availability test: buttons disabled when nothing to undo/redo |
| | - All undo/redo tests pass in CI |

### P5-T08: Implement board persistence

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T02, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for state encoding/decoding (Yjs binary ↔ BYTEA). Write integration test: save state, disconnect all clients, reconnect, verify state restored. Test auto-save triggers at 30s interval. Test empty board creates valid empty Yjs doc. 2. GREEN: Implement persistence in service and relay. 3. REFACTOR: Compress Yjs state before DB storage. |
| **Description** | Implement persistence for strategy boards. Save: encode Yjs doc state as binary, store in `strategy_boards.yjs_state`. Load: on WS connection, decode and apply state to new Yjs doc. Auto-save periodically and on last disconnect. Handle empty boards (new creation). |
| **Key Files** | `backend/internal/strat/service.go`, `backend/internal/websocket/yjs_relay.go`, `backend/internal/strat/service_test.go`, `backend/internal/websocket/yjs_relay_integration_test.go` |
| **Acceptance Criteria** | - Board state persists after all clients disconnect |
| | - Reconnecting clients see the saved state |
| | - New boards start with empty Yjs doc |
| | - Auto-save runs every 30s while clients connected |
| | - Binary state correctly encoded/decoded |
| | - Unit tests: Yjs state encode/decode roundtrip |
| | - Integration test: save → disconnect → reconnect → state restored |
| | - Auto-save test: state written at 30s interval |
| | - Empty board test: new board starts with valid Yjs doc |
| | - All persistence tests pass in CI |

### P5-T09: Build board list + create/delete UI

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P2-T04, P1-T12 |
| **Test Types** | component, e2e |
| **TDD Workflow** | 1. RED: Write RTL tests: board list renders user's boards with title/map/date. Create dialog opens with title and map fields. Delete shows confirmation. Click navigates to board page. Empty state shown. Write smoke E2E test: create board → draw → reload → verify persistence. 2. GREEN: Implement page and components. 3. REFACTOR: Extract board card component. |
| **Description** | Build the strategy board list page. Show user's boards with title, map name, last updated. Create button opens dialog (title + map selector). Delete button with confirmation. Click opens the board editor. |
| **Key Files** | `frontend/src/app/(app)/strats/page.tsx`, `frontend/src/components/strat/board-list.tsx`, `frontend/src/app/(app)/strats/page.test.tsx`, `frontend/src/components/strat/board-list.test.tsx`, `e2e/tests/strat-board.spec.ts` |
| **Acceptance Criteria** | - Board list shows all user's boards |
| | - Create dialog with title and map picker |
| | - Delete with confirmation |
| | - Click navigates to `/strats/{id}` |
| | - Empty state for no boards |
| | - Component tests: list rendering, create dialog, delete confirmation, navigation, empty state |
| | - Smoke E2E test: create board → draw → reload → persistence verified |
| | - All board list tests pass in CI |

### P5-T10: Implement sharing

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T08, P1-T11 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write unit tests for share token generation (unique, correct length). Write handler tests: update share mode, access shared board by token. Write integration test: create board → share → access as anonymous (read-only) and authenticated (editable). 2. GREEN: Implement sharing.go and handler. 3. REFACTOR: Centralize authorization logic for share modes. |
| **Description** | Implement board sharing. Generate unique share token. Share modes: private, read_only, editable. Share dialog with copy-link button. Shared board page accessible without auth (read_only) or with auth (editable). Update authorization in WS connection. |
| **Key Files** | `backend/internal/strat/sharing.go`, `backend/internal/handler/strat.go`, `frontend/src/components/strat/share-dialog.tsx`, `backend/internal/strat/sharing_test.go`, `backend/internal/handler/strat_test.go`, `frontend/src/components/strat/share-dialog.test.tsx` |
| **Acceptance Criteria** | - Share token generated on first share |
| | - Share link accessible: `/strats/shared/{token}` |
| | - Read-only mode: view but can't draw |
| | - Editable mode: full drawing access |
| | - Owner can change share mode or revoke |
| | - WS server enforces share mode permissions |
| | - Unit tests: token generation, share mode update, authorization checks |
| | - Integration test: full sharing flow (create → share → access by token) |
| | - Permission test: read-only users can't draw, editable users can |
| | - All sharing tests pass in CI |

### P5-T11: Implement PNG export

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P5-T04, P1-T12 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write unit tests for export: canvas capture produces valid PNG blob, filename derived from board title, export captures full view (not just viewport). 2. GREEN: Implement export.ts. 3. REFACTOR: Add option for current-view vs full-board export. |
| **Description** | Add "Export as PNG" button to the strategy board. Capture the current canvas state as a PNG image. Download to user's device. |
| **Key Files** | `frontend/src/lib/strat/export.ts`, `frontend/src/lib/strat/export.test.ts` |
| **Acceptance Criteria** | - PNG export captures full board including map background |
| | - Export respects current zoom/pan (or option for full view) |
| | - Downloaded file named `{board-title}.png` |
| | - Reasonable file size (< 5 MB) |
| | - Unit tests: PNG blob creation, filename generation, full-view capture |
| | - All export tests pass in CI |

### P5-T12: Add grenade extraction to demo parser ✅

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T08, P1-T11 |
| **Test Types** | unit, integration, golden |
| **TDD Workflow** | 1. RED: Write golden file tests: parse fixture demo → verify grenade lineups in output match .golden.json. Write unit tests for throw/detonate correlation logic (entity ID + tick proximity). Write test for auto-title generation. 2. GREEN: Implement grenade_extractor.go. 3. REFACTOR: Share entity correlation logic with kill event handling. |
| **Description** | Extend the demo parser to extract grenade lineups. For each grenade throw event, capture: thrower position, aim angles (yaw, pitch), grenade type. For each grenade detonate event, capture: landing position. Correlate throw → detonate pairs. Auto-create entries in `grenade_lineups` table. |
| **Key Files** | `backend/internal/demo/grenade_extractor.go`, `backend/internal/demo/grenade_extractor_test.go`, `backend/testdata/golden/small_match_lineups.golden.json` |
| **Acceptance Criteria** | - Grenade throw/detonate pairs correctly correlated |
| | - Lineup entries created with all required fields |
| | - Throw and landing positions accurate |
| | - Aim angles (yaw, pitch) captured |
| | - Auto-generated title (e.g., "Smoke T Spawn → A Site") |
| | - Linked to demo_id and tick |
| | - Golden file tests: grenade lineups match expected output |
| | - Unit tests: throw/detonate correlation, auto-title generation |
| | - Accuracy test: throw and landing positions verified against known coordinates |
| | - All grenade extraction tests pass in CI |

### P5-T13: Build lineup catalog page

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T12, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: lineup list filterable by map and grenade type. 2D preview renders throw → landing on minimap. Search by title/tags works. "View in Demo" link navigates to correct tick. Pagination works. 2. GREEN: Implement page and components. 3. REFACTOR: Extract lineup preview into reusable component. |
| **Description** | Build the grenade lineup catalog page. Browse by map and grenade type. Each entry: 2D preview (throw position + arrow to landing on minimap), title, tags. Search functionality. "View in Demo" link to jump to the source tick. |
| **Key Files** | `frontend/src/app/(app)/lineups/page.tsx`, `frontend/src/components/lineups/lineup-card.tsx`, `frontend/src/components/lineups/lineup-preview.tsx`, `frontend/src/app/(app)/lineups/page.test.tsx`, `frontend/src/components/lineups/lineup-card.test.tsx`, `frontend/src/components/lineups/lineup-preview.test.tsx` |
| **Acceptance Criteria** | - Lineup list filterable by map and grenade type |
| | - 2D preview shows throw → landing on minimap |
| | - Search by title and tags works |
| | - "View in Demo" navigates to viewer at correct tick |
| | - Pagination for large collections |
| | - Component tests: filter by map/type, 2D preview, search, "View in Demo" navigation, pagination |
| | - All lineup catalog tests pass in CI |

### P5-T14: Implement lineup CRUD + favorites

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T13, P1-T12 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests: edit dialog allows changing title/description/tags. Delete shows confirmation. Favorite toggle sends API call. "My Lineups" filter works. Manual lineup creation by clicking map. 2. GREEN: Implement components and handlers. 3. REFACTOR: Extract form validation logic. |
| **Description** | Implement full CRUD for lineups: edit title/description/tags, delete. Favorite toggle (star icon). "My Lineups" filtered view showing only user-saved lineups. Manual lineup creation (click throw + landing positions on map). |
| **Key Files** | `frontend/src/components/lineups/lineup-edit-dialog.tsx`, `frontend/src/app/(app)/lineups/[lineupId]/page.tsx`, `frontend/src/components/lineups/lineup-edit-dialog.test.tsx`, `frontend/src/app/(app)/lineups/[lineupId]/page.test.tsx` |
| **Acceptance Criteria** | - Edit dialog for title, description, tags |
| | - Delete with confirmation |
| | - Favorite toggle saves to DB |
| | - "My Lineups" filter works |
| | - Manual lineup creation by clicking map positions |
| | - Component tests: edit dialog, delete confirmation, favorite toggle, "My Lineups" filter, manual creation |
| | - All lineup CRUD tests pass in CI |

---

## 7. Phase 6: Polish & Deploy

### P6-T01: Performance profiling and optimization (frontend)

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P3, P4, P5 |
| **Test Types** | integration (benchmark) |
| **TDD Workflow** | 1. RED: Write performance benchmark tests: PixiJS render loop must maintain 60 FPS with 10 players + 5 effects. Heatmap render must complete in < 2s. Bundle size check must be < 500 KB gzipped. 2. GREEN: Profile and optimize to meet benchmarks. 3. REFACTOR: Apply optimizations (sprite batching, code splitting, lazy loading). |
| **Description** | Profile frontend performance: PixiJS render loop (maintain 60 FPS), bundle size (code splitting, tree shaking), TanStack Query cache efficiency, Zustand re-render frequency. Optimize: reduce draw calls, use sprite batching, lazy-load routes, optimize images. |
| **Key Files** | Various frontend files, frontend/src/lib/pixi/__benchmarks__/render.bench.ts |
| **Acceptance Criteria** | - 2D Viewer maintains 60 FPS at 1080p |
| | - Heatmap renders in < 2s (single demo) |
| | - TTI < 3 seconds on broadband |
| | - Bundle size reasonable (< 500 KB gzipped initial load) |
| | - No memory leaks over extended viewer sessions |
| | - Benchmark tests: 60 FPS render, < 2s heatmap, < 500 KB bundle |
| | - Memory leak test: no growth over extended viewer session |
| | - All benchmarks pass in CI |

### P6-T02: Performance profiling and optimization (backend)

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P3, P4, P5 |
| **Test Types** | integration (benchmark) |
| **TDD Workflow** | 1. RED: Write benchmark tests: demo parse < 30s for 100 MB, API p95 < 200ms, tick query < 100ms for 6400-tick range. 2. GREEN: Profile and optimize SQL queries (EXPLAIN ANALYZE), batch inserts, connection pooling. 3. REFACTOR: Apply query optimizations and index tuning. |
| **Description** | Profile backend performance: demo parse time, DB query latency, API response times. Optimize: SQL query plans (EXPLAIN ANALYZE), batch insert efficiency, Redis caching strategy, connection pooling. Target: < 30s parse, < 200ms API p95. |
| **Key Files** | Various backend files, backend/internal/demo/parser_benchmark_test.go, backend/internal/store/benchmark_test.go |
| **Acceptance Criteria** | - Demo parse < 30s for 100 MB file |
| | - API p95 latency < 200ms |
| | - Tick data query efficient for 6400-tick ranges |
| | - No N+1 queries |
| | - Connection pool sizes tuned |
| | - Benchmark tests: parse < 30s, API p95 < 200ms, tick query < 100ms |
| | - No N+1 queries verified via query logging |
| | - All benchmarks pass in CI |

### P6-T03: Security hardening

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T03 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write tests for CSRF token validation (reject missing/invalid tokens). Write rate limit tests (requests rejected after limit). Write input validation tests (reject malformed inputs). Write header tests (CSP, HSTS, X-Frame-Options present). 2. GREEN: Implement middleware. 3. REFACTOR: Consolidate security middleware chain. |
| **Description** | Implement security measures: CSRF tokens on state-changing endpoints, rate limiting (per-IP and per-user), input validation on all endpoints, SQL injection prevention (sqlc already handles this), XSS prevention (Content-Security-Policy headers), secure headers (HSTS, X-Frame-Options, etc.). Demo file validation hardening. |
| **Key Files** | `backend/internal/middleware/security.go`, `backend/internal/middleware/ratelimit.go`, backend/internal/middleware/security_test.go, backend/internal/middleware/ratelimit_test.go |
| **Acceptance Criteria** | - CSRF protection on POST/PUT/DELETE endpoints |
| | - Rate limiting: 100 req/min per IP on API, 5 uploads/hour per user |
| | - Security headers set (CSP, HSTS, X-Frame-Options, X-Content-Type-Options) |
| | - All user input validated and sanitized |
| | - Demo file validated beyond extension (magic bytes, structure) |
| | - Unit tests: CSRF validation, rate limit enforcement, input sanitization |
| | - Integration test: security headers present on responses |
| | - All security tests pass in CI |

### P6-T04: Add TLS configuration to nginx

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T03 |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via curl with TLS and header inspection. |
| **Description** | Add TLS support to nginx config. Self-signed certs for local dev. Let's Encrypt / cert-manager placeholder for production. Configure TLS 1.3, strong cipher suites, HSTS header. |
| **Key Files** | `nginx/nginx.conf`, `nginx/ssl/` |
| **Acceptance Criteria** | - HTTPS works locally with self-signed cert |
| | - TLS 1.3 enforced |
| | - HTTP redirects to HTTPS |
| | - HSTS header set |

### P6-T05: Implement responsive layouts

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T07, P4-T03, P5-T09 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write RTL tests with different viewport widths: sidebar collapses at tablet width, controls stack vertically on narrow screens, no horizontal overflow at 768px. 2. GREEN: Implement responsive CSS/layout changes. 3. REFACTOR: Extract breakpoint constants. |
| **Description** | Make all pages responsive. Sidebar collapses to hamburger menu on mobile. 2D Viewer: controls stack vertically on narrow screens, canvas full-width. Dashboard and lists: card layout adapts. Strat board: toolbar moves to bottom on mobile. Minimum supported width: 768px (tablet). |
| **Key Files** | Various frontend layout files, frontend/src/components/layout/sidebar.responsive.test.tsx |
| **Acceptance Criteria** | - Usable on tablet (1024px width) |
| | - Functional on mobile (768px width) |
| | - No horizontal overflow |
| | - Touch interactions work (pan, zoom, draw) |
| | - Sidebar responsive (collapsible) |
| | - Component tests: layout adapts at 768px, 1024px breakpoints |
| | - No overflow test: no horizontal scroll at 768px width |
| | - All responsive tests pass in CI |

### P6-T06: Write README.md

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T10 |
| **Test Types** | N/A (documentation) |
| **TDD Workflow** | N/A -- review-based quality gate. |
| **Description** | Write a comprehensive README.md: project description, features, screenshots (placeholder), tech stack, prerequisites, getting started (clone → running), development commands, architecture overview (link to docs), contributing guidelines, license. |
| **Key Files** | `README.md` |
| **Acceptance Criteria** | - Clear getting-started instructions |
| | - All prerequisites listed (Docker, pnpm, Go) |
| | - Dev commands documented |
| | - Links to `/docs/` for detailed documentation |

### P6-T07: Write API documentation

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P6-T02 |
| **Test Types** | N/A (documentation) |
| **TDD Workflow** | N/A -- review-based quality gate. |
| **Description** | Generate or write API documentation for all REST endpoints. Include request/response examples, error codes, authentication requirements. Consider OpenAPI/Swagger spec generation from the Go handlers. |
| **Key Files** | `docs/API.md` or `backend/api/openapi.yaml` |
| **Acceptance Criteria** | - All REST endpoints documented |
| | - Request/response examples for each endpoint |
| | - Error codes and meanings listed |
| | - Auth requirements clearly marked |
| | - WebSocket protocol documented |

### P6-T08: Create production Docker Compose

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T02 |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via `docker compose -f docker-compose.yml -f docker-compose.prod.yml up` and health checks. |
| **Description** | Create `docker-compose.prod.yml` with production overrides: environment variables from `.env` file, resource limits (memory, CPU), restart policies, log rotation, no debug ports exposed, multi-stage build Dockerfiles, health checks on all services. |
| **Key Files** | `docker-compose.prod.yml`, `.env.example` |
| **Acceptance Criteria** | - `docker compose -f docker-compose.yml -f docker-compose.prod.yml up` works |
| | - No debug ports exposed |
| | - Resource limits set on all services |
| | - Restart policies: `unless-stopped` |
| | - `.env.example` documents all required variables |
| | - Health checks on all services |

### P6-T09: Add error tracking setup

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T04, P1-T07 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write tests: frontend error boundary captures and reports. Backend middleware captures panics. User ID and environment attached to events. 2. GREEN: Implement Sentry integration. 3. REFACTOR: Centralize error context enrichment. |
| **Description** | Set up error tracking (Sentry or equivalent). Frontend: capture unhandled errors, React error boundaries report to Sentry. Backend: middleware captures panics and errors. Source maps uploaded for frontend. Environment and user context attached to events. |
| **Key Files** | `frontend/src/lib/error-tracking.ts`, `backend/internal/middleware/sentry.go`, frontend/src/lib/error-tracking.test.ts, backend/internal/middleware/sentry_test.go |
| **Acceptance Criteria** | - Frontend errors captured and sent to Sentry |
| | - Backend panics captured with stack trace |
| | - User ID attached to error events |
| | - Environment (dev/prod) tagged |
| | - Source maps enable readable stack traces |
| | - Unit tests: error capture and context enrichment |
| | - Integration test: error events contain user ID and environment |
| | - All error tracking tests pass in CI |

### P6-T10: End-to-end testing of critical paths

| | |
|---|---|
| **Complexity** | L |
| **Deps** | All previous phases |
| **Test Types** | e2e |
| **TDD Workflow** | 1. RED: Write Playwright E2E tests for all four critical user journeys: login → upload → parse → view; login → Faceit dashboard; create strat board → draw → share → view shared; browse lineups → view in demo. 2. GREEN: Fix any failures discovered. 3. REFACTOR: Extract common test helpers (login, navigation). |
| **Description** | Write end-to-end tests for critical user flows: login → upload demo → view in 2D viewer; login → view Faceit dashboard; create strat board → draw → share link → view shared. Use Playwright or Cypress. Test against Docker Compose environment. |
| **Key Files** | `e2e/tests/*.spec.ts`, e2e/tests/auth-flow.spec.ts, e2e/tests/demo-flow.spec.ts, e2e/tests/faceit-dashboard.spec.ts, e2e/tests/strat-board-flow.spec.ts, e2e/tests/lineup-flow.spec.ts |
| **Acceptance Criteria** | - Login flow test passes |
| | - Demo upload + parse + view flow test passes |
| | - Faceit dashboard loads with data |
| | - Strat board create + draw + share flow test passes |
| | - Tests run in CI against Docker Compose |
| | - Tests complete in < 5 minutes |
| | - Consolidates smoke E2E tests from P2-T12, P3-T12, P5-T09 into comprehensive suite |
| | - Coverage: 100% of US-01 (login), US-04 (upload), US-08 (viewer), US-21 (strat board) |
| | - All E2E tests pass in CI |

---

## 8. Critical Path Analysis

The longest dependency chain through the project:

```
P1-T01 → P1-T02 → P1-T05 → P1-T11 → P1-T06 → P2-T06 → P2-T08 → P2-T09 → P3-T03 → P3-T06 → P6-T01
  S         M         M         M         M         M         XL        L         L         L        L
```

**Critical path items:**

| Task | Why Critical |
|------|-------------|
| **P1-T11** (Go Test Infrastructure) | Test infrastructure must be ready before any TDD task in P2+ can begin |
| **P2-T08** (Demo Parser Core) | XL complexity; all viewer features depend on parsed data; highest risk of delays |
| **P2-T09** (Tick Data Ingestion) | Viewer can't render without tick data in DB |
| **P3-T03** (Tick Data Fetching) | Client-side buffer needed before any rendering |
| **P3-T06** (Playback Engine) | Core viewer functionality; all other viewer features depend on it |

### Bottleneck: P2-T08 Demo Parser

This single task is the project's biggest risk. Recommendations:

1. **Start a spike early**: Before Phase 2 officially begins, spend a day prototyping with `demoinfocs-golang` to validate assumptions
2. **Test with real demos**: Collect 5+ CS2 demo files of varying size and complexity (overtime, bots, disconnects)
3. **Incremental extraction**: Parse positions first, add events second, add edge cases third
4. **Benchmark continuously**: Track parse time and memory usage as features are added
5. **Golden file tests early**: Write golden file tests against spike output before full implementation

---

## 9. Risk Register

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|------------|
| R1 | **demoinfocs-golang doesn't support latest CS2 demo format** | Medium | Critical | Monitor library's GitHub issues; contribute patches if needed; keep parser modular for library swaps |
| R2 | **TimescaleDB tick data volume exceeds expectations** | Medium | High | Configurable tick sampling rate (every Nth tick); compression policies; retention policies; option to store only event ticks |
| R3 | **PixiJS 60 FPS not achievable with many concurrent effects** | Low | High | Object pooling for sprites; reduce draw calls via sprite batching; LOD (hide details at low zoom); profile early in P3 |
| R4 | **Yjs state grows unbounded for heavily-edited boards** | Medium | Medium | Periodic Yjs garbage collection; limit max elements per board; warn users when approaching limits |
| R5 | **Faceit API rate limiting disrupts sync** | Medium | Medium | Aggressive caching (Redis); exponential back-off; queue sync jobs with rate limiting; batch API calls |
| R6 | **Map coordinate calibration inaccurate** | Low | High | Verify calibration against known positions in demos; allow manual calibration adjustment; community-sourced data |
| R7 | **WebSocket connections dropped by corporate firewalls/proxies** | Medium | Medium | Implement reconnection with exponential back-off; Yjs handles state reconciliation on reconnect; fallback to polling (future) |
| R8 | **Demo file upload bandwidth bottleneck (100 MB+ files)** | Medium | Medium | Chunked upload (tus protocol or custom); progress indicator; resume on failure; compress before upload (future) |
| R9 | **TDD overhead slows early velocity** | Medium | Medium | Start with highest-value tests (parser golden files, auth integration); defer low-value tests; keep unit tests < 30s; timebox refactoring to 20% of task time |

---

## 10. Development Environment Setup

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Docker + Docker Compose | 24+ | Container runtime |
| Go | 1.22+ | Backend development |
| Node.js | 20 LTS | Frontend development |
| pnpm | 9+ | Package manager |
| Git | 2.40+ | Version control (worktree support) |
| Playwright | Latest | E2E and component screenshot testing |

### Clone to Running (First Time)

```bash
# 1. Clone bare repo + create worktree
git clone --bare git@github.com:user/oversite.git oversite
cd oversite
git worktree add ../oversite-main main
cd ../oversite-main

# 2. Copy environment file
cp .env.example .env
# Edit .env with your Faceit OAuth credentials

# 3. Start all services
make up

# 4. Run database migrations
make migrate-up

# 5. Install frontend dependencies
cd frontend && pnpm install && cd ..

# 6. Start development (hot reload)
make dev

# 7. Open in browser
# http://localhost (via nginx)
# http://localhost:3000 (Next.js direct)
# http://localhost:9001 (MinIO console)
```

### Common Development Commands

```bash
make up              # Start Docker Compose (background)
make down            # Stop Docker Compose
make dev             # Start with hot-reload overrides
make logs            # Tail all service logs
make logs s=api      # Tail specific service logs
make migrate-up      # Run pending migrations
make migrate-down    # Rollback last migration
make migrate-create  # Create new migration files
make sqlc            # Regenerate sqlc Go code
make lint            # Run all linters (Go + TS)
make test-unit       # Run Go + TS unit tests only
make test-integration # Run Go integration tests (testcontainers)
make test-e2e        # Run Playwright E2E tests
make test            # Run all test tiers sequentially
make build           # Build all artifacts
```

---

## 11. Sprint Pairing Recommendations

For solo development, these tasks can be naturally grouped into focused work sessions. For team development, these pairings allow parallel work with minimal conflicts.

### Parallel Tracks (After P1 Complete)

**Track A: Backend Pipeline**
- P2-T01 → P2-T02 → P2-T03 → P2-T05 → P2-T06 → P2-T07 → P2-T08 → P2-T09 → P2-T10 → P2-T11

**Track B: Frontend Shell**
- P2-T04 → P2-T12 → P3-T01 → P3-T02 → P3-T08 → P3-T09

### Parallel Tracks (After P2 Complete)

**Track A: Viewer Rendering**
- P3-T03 → P3-T04 → P3-T05 → P3-T06 → P3-T12

**Track B: Viewer UI**
- P3-T07 → P3-T10 → P3-T11

**Track C: Faceit Backend** (can start early)
- P4-T01 → P4-T02 → P4-T05

### Parallel Tracks (After P3 Complete)

**Track A: Heatmaps**
- P4-T06 → P4-T07 → P4-T08

**Track B: Faceit Dashboard**
- P4-T03 → P4-T04 → P4-T09 → P4-T10

**Track C: WebSocket Infrastructure**
- P5-T01 → P5-T02 → P5-T08

### Natural Sprint Groupings (Solo Dev)

| Sprint | Tasks | Focus |
|--------|-------|-------|
| 1 | P1-T01 through P1-T12 | Foundation + test infrastructure |
| 2 | P2-T01 through P2-T07 | Auth + infrastructure |
| 3 | P2-T08 through P2-T12 | Demo parser + library UI |
| 4 | P3-T01 through P3-T06 | Viewer core rendering |
| 5 | P3-T07 through P3-T12 | Viewer UI + polish |
| 6 | P4-T01 through P4-T05 | Faceit integration |
| 7 | P4-T06 through P4-T10 | Heatmaps + analytics |
| 8 | P5-T01 through P5-T08 | Strat board core |
| 9 | P5-T09 through P5-T14 | Sharing + lineups |
| 10 | P6-T01 through P6-T10 | Polish + deploy |

---

*Cross-references: [PRD.md](PRD.md) for feature requirements, [ARCHITECTURE.md](ARCHITECTURE.md) for system design, [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for phase milestones.*
