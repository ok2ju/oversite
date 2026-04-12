# Oversite -- Task Breakdown

> **Version:** 2.0
> **Last Updated:** 2026-04-12

---

## Table of Contents

1. [Task Legend](#1-task-legend)
2. [Phase 1: Desktop Foundation](#2-phase-1-desktop-foundation)
3. [Phase 2: Auth & Demo Pipeline](#3-phase-2-auth--demo-pipeline)
4. [Phase 3: Core 2D Viewer](#4-phase-3-core-2d-viewer)
5. [Phase 4: Faceit & Heatmaps](#5-phase-4-faceit--heatmaps)
6. [Phase 5: Strategy Board & Lineups](#6-phase-5-strategy-board--lineups)
7. [Phase 6: Polish & Distribute](#7-phase-6-polish--distribute)
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
| **TDD Workflow** | RED -> GREEN -> REFACTOR steps specific to this task |
| **Key Files** | Primary files created or modified (including test files) |

### TDD Workflow Convention

Every task follows the **Red-Green-Refactor** cycle unless marked `N/A`:

1. **RED**: Write failing tests that define the expected behavior
2. **GREEN**: Write the minimum code to make tests pass
3. **REFACTOR**: Clean up implementation and tests; all tests stay green
4. **COMMIT**: Commit after each green-to-refactor cycle

**Exceptions** (marked `TDD Workflow: N/A`): Infrastructure and configuration tasks (P1-T01, P1-T05, P1-T08, P1-T09, P1-T10, P6-T05, P6-T06, P6-T07, P6-T10) are verified via smoke tests or health checks, not TDD.

---

## 2. Phase 1: Desktop Foundation

### P1-T01: Initialize Wails project

| | |
|---|---|
| **Complexity** | M |
| **Deps** | None |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via `wails dev` launching the app window |
| **Description** | Initialize the Wails v2 project: Go module, `wails.json` config, `main.go` entry point with Wails `App` struct, frontend Vite scaffold with `embed.FS` integration. Set up the monorepo directory structure per ARCHITECTURE.md Section 10. |
| **Key Files** | `wails.json`, `backend/cmd/oversite/main.go`, `backend/internal/app/app.go`, `frontend/package.json`, `frontend/vite.config.ts`, `frontend/index.html` |
| **Acceptance Criteria** | - `wails dev` launches a window with the default Wails template |
| | - Go module compiles (`go build ./...`) |
| | - Frontend dev server runs (`pnpm dev`) |
| | - Directory structure matches ARCHITECTURE.md Section 10 |

### P1-T02: Set up SQLite with migrations

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T01 |
| **Test Types** | integration |
| **TDD Workflow** | 1. RED: Write test that opens SQLite DB, runs migrations, and verifies tables exist. 2. GREEN: Implement SQLite connection (modernc.org/sqlite), WAL mode setup, migration runner. 3. REFACTOR: Extract DB setup into reusable function. |
| **Description** | Set up SQLite using `modernc.org/sqlite` (pure Go). Enable WAL mode. Create migration framework (golang-migrate with SQLite driver or custom). Write initial migration with full schema from ARCHITECTURE.md Section 7. |
| **Key Files** | `backend/internal/store/db.go`, `backend/internal/store/db_test.go`, `backend/migrations/001_initial_schema.up.sql`, `backend/migrations/001_initial_schema.down.sql` |
| **Acceptance Criteria** | - SQLite database created in OS app data directory |
| | - WAL mode enabled (`PRAGMA journal_mode` returns `wal`) |
| | - All tables from schema DDL created |
| | - Foreign keys enforced (`PRAGMA foreign_keys` returns 1) |
| | - Integration test creates temp DB, runs migrations, verifies tables |

### P1-T03: Configure sqlc for SQLite dialect

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T02 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write test for a basic query (e.g., insert + select a demo). 2. GREEN: Configure sqlc.yaml for SQLite, write SQL queries, generate Go code. 3. REFACTOR: Organize query files by domain. |
| **Description** | Configure sqlc with SQLite dialect. Write SQL query files for all entities: users, demos, rounds, player_rounds, tick_data, game_events, strategy_boards, grenade_lineups, faceit_matches. Generate type-safe Go code. Verify generated code compiles and basic CRUD works against temp SQLite. |
| **Key Files** | `backend/sqlc.yaml`, `backend/queries/*.sql`, `backend/internal/store/*.sql.go` (generated) |
| **Acceptance Criteria** | - `sqlc generate` succeeds with no errors |
| | - Generated Go code compiles |
| | - Basic insert + select test passes against temp SQLite |
| | - Query files organized by domain (demos.sql, rounds.sql, etc.) |

### P1-T04: Scaffold Vite + React frontend

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T01 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write component test for app shell rendering sidebar + content area. 2. GREEN: Implement react-router-dom routes, root layout with sidebar, placeholder pages. 3. REFACTOR: Extract sidebar into component; add route constants. |
| **Description** | Set up the frontend as a Vite + React SPA with react-router-dom v6. Create the app shell (sidebar + header + content outlet). Set up route structure per PRD Section 4. Create placeholder pages for all routes. Integrate with Wails JS runtime. |
| **Key Files** | `frontend/src/main.tsx`, `frontend/src/routes/root.tsx`, `frontend/src/routes/*.tsx`, `frontend/src/components/layout/sidebar.tsx`, `frontend/src/components/layout/header.tsx` |
| **Acceptance Criteria** | - All routes from PRD Section 4 render correct placeholder pages |
| | - Sidebar navigation highlights active route |
| | - react-router-dom v6 with Outlet-based layout works |
| | - Wails JS runtime imports resolve |
| | - Component test for shell layout passes |

### P1-T05: Configure shadcn/ui + Tailwind CSS

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T04 |
| **Test Types** | N/A (configuration) |
| **TDD Workflow** | N/A -- verify via visual inspection of themed components |
| **Description** | Install and configure shadcn/ui with Tailwind CSS. Set up dark/light theme switching. Install initial component set: Button, Card, Dialog, Tabs, Select, Slider, Input, Table, DropdownMenu, Toast. |
| **Key Files** | `frontend/tailwind.config.ts`, `frontend/src/lib/utils.ts`, `frontend/src/components/ui/*.tsx`, `frontend/components.json` |
| **Acceptance Criteria** | - shadcn/ui components render correctly |
| | - Dark/light theme toggle works |
| | - Tailwind classes apply correctly |
| | - Component library matches design system |

### P1-T06: Set up Zustand stores (skeleton)

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T04 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write tests for each store's initial state and basic actions. 2. GREEN: Create stores with typed state and actions. 3. REFACTOR: Extract shared patterns. |
| **Description** | Create Zustand stores: `viewerStore` (playback state, current tick, speed), `stratStore` (board state, tool selection), `uiStore` (sidebar, modals, theme), `faceitStore` (profile, matches), `demoStore` (library state, filters). |
| **Key Files** | `frontend/src/stores/viewer.ts`, `frontend/src/stores/strat.ts`, `frontend/src/stores/ui.ts`, `frontend/src/stores/faceit.ts`, `frontend/src/stores/demo.ts`, `frontend/src/stores/*.test.ts` |
| **Acceptance Criteria** | - All stores have typed state + actions |
| | - Unit tests verify initial state and basic state transitions |
| | - Stores export selectors for common derived state |

### P1-T07: Set up CI pipeline

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T01, P1-T09, P1-T10 |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via CI pipeline passing |
| **Description** | Set up CI (GitHub Actions) with: Go lint (`golangci-lint`), Go test (`go test -race ./...`), TypeScript lint (ESLint), TypeScript typecheck (`tsc --noEmit`), TypeScript test (Vitest), Wails build (cross-platform). |
| **Key Files** | `.github/workflows/ci.yml` |
| **Acceptance Criteria** | - CI runs on every push and PR |
| | - Go lint + test pass |
| | - TypeScript lint + typecheck + test pass |
| | - Wails build produces binaries for at least one platform |

### P1-T08: Create root Makefile

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T01 |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via running make targets |
| **Description** | Create root Makefile with development commands: `make dev` (wails dev), `make build` (wails build), `make test` (go test + pnpm test), `make lint`, `make typecheck`, `make sqlc`, `make migrate-up`, `make migrate-down`, `make clean`. |
| **Key Files** | `Makefile` |
| **Acceptance Criteria** | - All make targets execute correctly |
| | - `make dev` starts Wails dev mode |
| | - `make test` runs all test suites |

### P1-T09: Set up Go test infrastructure

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T02 |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via running an example test |
| **Description** | Create shared test helpers: `testutil.NewTestDB()` returns temp SQLite with migrations applied (`:memory:` for speed). Create mock interfaces: `MockKeyring`, `MockFaceitClient`. Create golden file test helpers. Set up `go test -race` as default. |
| **Key Files** | `backend/internal/testutil/db.go`, `backend/internal/testutil/mocks.go`, `backend/internal/testutil/golden.go` |
| **Acceptance Criteria** | - `testutil.NewTestDB()` returns a migrated temp SQLite |
| | - Mock interfaces match production interfaces |
| | - `go test -race ./...` passes with example test |
| | - Golden file helpers can write + compare files |

### P1-T10: Set up frontend test infrastructure

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T04 |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via running an example test |
| **Description** | Configure Vitest with React Testing Library. Set up `renderWithProviders()` helper (QueryClient, RouterProvider, ThemeProvider). Create MSW server for Faceit API mocking. Create mock factories for Wails binding functions. Set up Playwright config for E2E. |
| **Key Files** | `frontend/vitest.config.ts`, `frontend/src/test/render.tsx`, `frontend/src/test/msw/handlers.ts`, `frontend/src/test/mocks/bindings.ts`, `e2e/playwright.config.ts` |
| **Acceptance Criteria** | - `pnpm test` runs with example component test passing |
| | - `renderWithProviders()` wraps with all needed providers |
| | - MSW handlers intercept Faceit API patterns |
| | - Mock binding factories return typed test data |
| | - Playwright config points at Wails dev server |

---

## 3. Phase 2: Auth & Demo Pipeline

### P2-T01: Implement loopback OAuth flow

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T02, P1-T09 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write test for temp HTTP listener capturing callback code; test PKCE code verifier/challenge generation. 2. GREEN: Implement temp listener, PKCE helpers, token exchange. 3. REFACTOR: Extract HTTP client, add timeout handling. |
| **Description** | Implement RFC 8252 loopback OAuth: start temp listener on `127.0.0.1:{random_port}`, generate PKCE code verifier/challenge, build Faceit auth URL, open system browser, capture callback, exchange code for tokens. |
| **Key Files** | `backend/internal/auth/oauth.go`, `backend/internal/auth/pkce.go`, `backend/internal/auth/oauth_test.go`, `backend/internal/auth/pkce_test.go` |
| **Acceptance Criteria** | - Temp listener starts on random port and captures auth code |
| | - PKCE code verifier/challenge are RFC 7636 compliant |
| | - Token exchange works against Faceit token endpoint |
| | - Listener shuts down after callback or timeout (30s) |
| | - Unit tests pass for PKCE generation and callback capture |

### P2-T02: Implement keychain token storage

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P1-T09 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write test for store/retrieve/delete tokens using mock keyring. 2. GREEN: Implement keyring wrapper with `zalando/go-keyring`. 3. REFACTOR: Define interface for testing. |
| **Description** | Create a `TokenStore` interface backed by `zalando/go-keyring`. Store refresh token under service name `oversite-faceit-auth`. Access token held in memory only. Mock interface for testing. |
| **Key Files** | `backend/internal/auth/keyring.go`, `backend/internal/auth/keyring_test.go`, `backend/internal/testutil/mocks.go` (add MockKeyring) |
| **Acceptance Criteria** | - Refresh token stored/retrieved from OS keychain |
| | - Access token held in memory, not persisted |
| | - `TokenStore` interface allows mock substitution |
| | - Unit tests pass with mock keyring |

### P2-T03: Create auth service

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T01, P2-T02 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write test for login flow (mock OAuth + mock keyring -> returns user). Test for token refresh. Test for logout (clears keyring). 2. GREEN: Implement AuthService with login, logout, refresh, getCurrentUser. 3. REFACTOR: Add error types for auth failures. |
| **Description** | Create `AuthService` that orchestrates: loopback OAuth -> token exchange -> fetch Faceit profile -> upsert user in SQLite -> store refresh token in keychain. Expose as Wails bindings: `StartLogin()`, `GetCurrentUser()`, `Logout()`, `RefreshProfile()`. |
| **Key Files** | `backend/internal/auth/service.go`, `backend/internal/auth/service_test.go` |
| **Acceptance Criteria** | - Login flow creates/updates user in SQLite |
| | - Logout clears keychain token and in-memory access token |
| | - Token refresh fetches new access token using refresh token |
| | - getCurrentUser returns nil when not logged in |
| | - All tests pass with mock keyring and mock Faceit client |

### P2-T04: Create AuthProvider + login page

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T03, P1-T10 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write component test: login page shows button; clicking triggers StartLogin binding; success redirects to dashboard. 2. GREEN: Implement AuthProvider context, login page, auth guard. 3. REFACTOR: Extract protected route wrapper. |
| **Description** | Create React AuthProvider that calls `GetCurrentUser()` on mount. Login page with Faceit branding and "Login with Faceit" button. Protected route wrapper redirects unauthenticated users to login. |
| **Key Files** | `frontend/src/routes/login.tsx`, `frontend/src/hooks/useAuth.ts`, `frontend/src/components/auth/auth-provider.tsx`, `frontend/src/components/auth/protected-route.tsx`, `frontend/src/routes/login.test.tsx` |
| **Acceptance Criteria** | - Login page renders with Faceit login button |
| | - Clicking login calls `StartLogin()` Wails binding |
| | - Successful login redirects to dashboard |
| | - Protected routes redirect to login when unauthenticated |
| | - Component tests pass with mock bindings |

### P2-T05: Implement demo import binding

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T03, P1-T09 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write test: import valid .dem file creates demo record in SQLite with status=imported. Test: import invalid file returns validation error. 2. GREEN: Implement ImportDemo binding with file validation (magic bytes, size). 3. REFACTOR: Extract validation into reusable function. |
| **Description** | Implement `ImportDemo(path)` Wails binding: validate `.dem` file (check magic bytes `HL2DEMO`, size limits), insert demo record in SQLite with status `imported`, trigger parsing. Implement `ImportFolder(path)` for recursive scan. Implement `OpenFileDialog()` and `OpenFolderDialog()` using Wails runtime dialogs. |
| **Key Files** | `backend/internal/demo/import.go`, `backend/internal/demo/import_test.go`, `backend/internal/demo/validate.go`, `backend/internal/demo/validate_test.go` |
| **Acceptance Criteria** | - Valid `.dem` file creates demo record with correct metadata |
| | - Invalid file (wrong magic bytes, too large) returns descriptive error |
| | - Folder import recursively finds all `.dem` files |
| | - File picker dialogs work on all platforms |
| | - Unit tests pass for validation; integration tests pass for import |

### P2-T06: Implement demo parser core

| | |
|---|---|
| **Complexity** | **XL** |
| **Deps** | P2-T05, P1-T09 |
| **Test Types** | unit, golden |
| **TDD Workflow** | 1. Spike: prototype with demoinfocs-golang to validate API, extract sample data. 2. RED: Write golden file tests comparing parser output against known-good data. 3. GREEN: Implement full parser with event handlers for positions, kills, grenades, bombs, rounds. 4. REFACTOR: Extract per-event-type handlers; add edge case handling (warmup, OT, bots). |
| **Description** | Integrate `markus-wa/demoinfocs-golang` v5. Register handlers for: player positions (every Nth tick), kills, grenade throws/detonations, bomb plant/defuse, round start/end. Extract match metadata (map, duration, tick rate). Handle edge cases: warmup rounds, overtime, bot players, disconnects. This is the same parser logic as the web version -- only the output target changes (direct SQLite insert vs. Redis Streams + Worker). |
| **Key Files** | `backend/internal/demo/parser.go`, `backend/internal/demo/parser_test.go`, `backend/testdata/*.golden` |
| **Acceptance Criteria** | - Parser extracts positions, kills, grenades, bombs, rounds from test demos |
| | - Golden file tests pass for at least 3 different demo files |
| | - Warmup rounds are excluded from parsed data |
| | - Bot players are handled gracefully |
| | - Memory usage stays < 500 MB for 100 MB demo files |
| | - Parse time < 10 seconds for average demo on modern hardware |

### P2-T07: Parse ticks -> batch insert into SQLite

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P2-T06 |
| **Test Types** | integration |
| **TDD Workflow** | 1. RED: Write test: parse demo + insert ticks; verify tick_data table has expected row count and sample values. 2. GREEN: Implement batched transaction inserts (10K rows per transaction). 3. REFACTOR: Tune batch size; add progress reporting via Wails events. |
| **Description** | After parsing, batch-insert tick data into SQLite `tick_data` table. Use transactions with 10,000-row batches for write performance. Emit Wails runtime events for progress reporting. Verify composite PK `(demo_id, tick, steam_id)` handles the ~1.28M rows per demo. |
| **Key Files** | `backend/internal/demo/ingest.go`, `backend/internal/demo/ingest_test.go` |
| **Acceptance Criteria** | - ~1.28M rows inserted for a typical demo |
| | - Batch inserts complete in < 5 seconds |
| | - Progress events emitted during ingestion |
| | - Range query `WHERE demo_id = ? AND tick BETWEEN ? AND ?` returns in < 50ms |
| | - Integration test verifies row count and sample data accuracy |

### P2-T08: Parse events -> insert game_events

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P2-T06 |
| **Test Types** | integration |
| **TDD Workflow** | 1. RED: Write test: parse demo; verify game_events table has correct kill/grenade/bomb events with positions. 2. GREEN: Insert events from parser output. 3. REFACTOR: Normalize event types; add extra_data JSON for type-specific fields. |
| **Description** | Insert game events (kills, grenade throws, grenade detonations, bomb plants, bomb defuses) into `game_events` table. Include attacker/victim steam IDs, weapons, positions, and event-specific extra data as JSON. |
| **Key Files** | `backend/internal/demo/events.go`, `backend/internal/demo/events_test.go` |
| **Acceptance Criteria** | - All event types correctly inserted with positions |
| | - Kill events include headshot flag, weapon, flash assist in extra_data |
| | - Grenade events include throw and landing positions |
| | - Event timestamps (ticks) are accurate |
| | - Integration test verifies event counts and sample data |

### P2-T09: Parse rounds -> insert rounds + player_rounds

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T06 |
| **Test Types** | integration |
| **TDD Workflow** | 1. RED: Write test: parse demo; verify rounds table has correct round count, scores, win reasons. Verify player_rounds has per-player stats. 2. GREEN: Insert round and player_round data. 3. REFACTOR: Handle overtime rounds; validate score progression. |
| **Description** | Insert round summaries and per-player-per-round statistics into `rounds` and `player_rounds` tables. |
| **Key Files** | `backend/internal/demo/rounds.go`, `backend/internal/demo/rounds_test.go` |
| **Acceptance Criteria** | - Round count matches actual demo rounds (excluding warmup) |
| | - Scores progress correctly |
| | - Player K/D/A/damage totals match per-round |
| | - Overtime rounds handled correctly |
| | - Integration test verifies against known demo data |

### P2-T10: Build demo library UI

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T05, P1-T10 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write component test: library renders list of demos; drag-drop zone accepts files; delete shows confirmation. 2. GREEN: Implement library page with list/grid views, drag-drop, status badges, delete. 3. REFACTOR: Extract demo card component; add sort/filter controls. |
| **Description** | Build the demo library page: list and grid view modes, drag-and-drop import zone, demo cards showing map, date, player count, parse status. Delete with confirmation dialog. Status badges (imported, parsing, ready, error). Wails file/folder picker integration. |
| **Key Files** | `frontend/src/routes/demos/index.tsx`, `frontend/src/components/demos/demo-card.tsx`, `frontend/src/components/demos/drop-zone.tsx`, `frontend/src/routes/demos/index.test.tsx` |
| **Acceptance Criteria** | - Library renders demo list from `ListDemos()` binding |
| | - Drag-drop accepts `.dem` files and triggers import |
| | - File/folder picker buttons work |
| | - Delete shows confirmation dialog |
| | - Status badges show correct parse state |
| | - Component tests pass with mock bindings |

### P2-T11: Implement folder import binding

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P2-T05 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write test: given a temp dir with .dem and non-.dem files, ImportFolder returns only .dem paths. 2. GREEN: Implement recursive directory walk with .dem filter. 3. REFACTOR: Add progress event for large folders. |
| **Description** | Implement `ImportFolder(path)` binding that recursively scans a directory for `.dem` files and imports each one. Skip non-`.dem` files. Report progress via Wails events. |
| **Key Files** | `backend/internal/demo/folder.go`, `backend/internal/demo/folder_test.go` |
| **Acceptance Criteria** | - Recursively finds all `.dem` files in directory tree |
| | - Skips non-`.dem` files and directories |
| | - Returns list of imported demos |
| | - Progress events emitted during scan |

---

## 4. Phase 3: Core 2D Viewer

### P3-T01: Set up PixiJS Application + canvas container

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T04 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write test: viewer route renders container div; PixiJS app initializes. 2. GREEN: Implement useEffect-based PixiJS setup with canvas. 3. REFACTOR: Extract cleanup logic; handle resize events. |
| **Description** | Set up PixiJS v8 Application instantiated in `useEffect` (per ADR-0001). React renders a container `<div>`; PixiJS creates and owns the `<canvas>`. Handle window resize. Connect to viewer Zustand store via `subscribe()`. |
| **Key Files** | `frontend/src/components/viewer/canvas.tsx`, `frontend/src/hooks/usePixiApp.ts`, `frontend/src/components/viewer/canvas.test.tsx` |
| **Acceptance Criteria** | - PixiJS canvas renders in the viewer route |
| | - Canvas resizes on window resize |
| | - Cleanup destroys PixiJS Application on unmount |
| | - No React re-renders during PixiJS render loop |

### P3-T02: Implement map layer

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T01 |
| **Test Types** | unit, screenshot |
| **TDD Workflow** | 1. RED: Write unit test for coordinate calibration transform (world -> pixel). 2. GREEN: Implement map sprite loading, calibration data, coordinate transform functions. 3. REFACTOR: Create CalibrationData type; extract map constants. |
| **Description** | Load radar images for each CS2 map. Apply coordinate calibration (origin, scale) to transform world-space coordinates to pixel-space. Support all Active Duty maps. |
| **Key Files** | `frontend/src/lib/maps/calibration.ts`, `frontend/src/lib/maps/calibration.test.ts`, `frontend/src/lib/pixi/map-layer.ts`, `frontend/public/maps/*.png` |
| **Acceptance Criteria** | - Correct radar image loads for each map name |
| | - Coordinate transform produces accurate pixel positions |
| | - Unit tests pass for all supported maps |
| | - Map fills canvas at default zoom level |

### P3-T03: Implement tick data fetching

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P3-T01, P2-T07 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write test for tick buffer: request range, buffer returns data, prefetches next range. 2. GREEN: Implement tick buffer using `GetTicks()` Wails binding with lookahead prefetch. 3. REFACTOR: Tune buffer size and prefetch thresholds. |
| **Description** | Create a client-side tick data buffer that fetches tick ranges via `GetTicks()` Wails binding. Prefetch ahead of playback position. Buffer window: current tick +/- N ticks. SQLite responses are fast (<50ms), so smaller buffers are acceptable vs. the web version. |
| **Key Files** | `frontend/src/lib/viewer/tick-buffer.ts`, `frontend/src/lib/viewer/tick-buffer.test.ts` |
| **Acceptance Criteria** | - Buffer fetches tick data on demand via Wails binding |
| | - Prefetch triggers before buffer runs out during playback |
| | - Buffer handles seek (clears and re-fetches) |
| | - No dropped frames due to data fetching latency |
| | - Unit tests pass for buffer logic |

### P3-T04: Implement player layer

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T02, P3-T03 |
| **Test Types** | unit, screenshot |
| **TDD Workflow** | 1. RED: Write unit test: given tick data, player sprites are positioned correctly. 2. GREEN: Implement player sprites with team colors, names, view angles. 3. REFACTOR: Object pool for sprites; extract rendering constants. |
| **Description** | Render each player as a colored circle with: team color (CT/T), name label, view-angle indicator, health bar (toggle). Dim dead players; show death X. Highlight selected player. |
| **Key Files** | `frontend/src/lib/pixi/player-layer.ts`, `frontend/src/lib/pixi/player-layer.test.ts` |
| **Acceptance Criteria** | - All 10 players rendered at correct positions |
| | - Team colors distinguish CT and T |
| | - View angle cone/line shows look direction |
| | - Dead players dimmed with death X marker |
| | - Unit tests pass for position calculations |

### P3-T05: Implement event layer

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P3-T02, P3-T03 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write tests for each event type's visual representation logic. 2. GREEN: Implement kill lines, grenade circles, bomb icons. 3. REFACTOR: Event renderer registry; extract timing/fade logic. |
| **Description** | Render game events on the map: kill lines (attacker -> victim), grenade effects (smoke circles, flash, HE, molotov), bomb plant/defuse indicators. Events appear at their tick and fade over configurable duration. |
| **Key Files** | `frontend/src/lib/pixi/event-layer.ts`, `frontend/src/lib/pixi/event-layer.test.ts` |
| **Acceptance Criteria** | - Kill events show line + death X |
| | - Smoke shows gray circle with fade |
| | - Flash shows yellow burst |
| | - HE shows red expanding circle |
| | - Molotov shows orange fill |
| | - Bomb plant/defuse icons render correctly |

### P3-T06: Implement playback engine

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P3-T03 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write tests for tick advancement at various speeds; interpolation between ticks; seek behavior. 2. GREEN: Implement requestAnimationFrame loop, tick interpolation, speed multiplier. 3. REFACTOR: Extract timing logic; add frame-independent delta time. |
| **Description** | Core playback loop using `requestAnimationFrame`. Advance ticks at configurable speed (0.25x-4x). Interpolate between ticks for smooth motion. Handle play, pause, seek, and round boundaries. |
| **Key Files** | `frontend/src/lib/viewer/playback-engine.ts`, `frontend/src/lib/viewer/playback-engine.test.ts` |
| **Acceptance Criteria** | - Playback advances at correct speed |
| | - Interpolation produces smooth motion between ticks |
| | - Seek jumps to correct tick immediately |
| | - Pause stops tick advancement |
| | - Frame rate stays at 60 FPS |

### P3-T07: Build playback controls UI

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T06, P1-T10 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write component tests for play/pause toggle, speed selector, timeline scrubber. 2. GREEN: Implement controls with shadcn/ui components. 3. REFACTOR: Extract reusable timeline component. |
| **Description** | Build playback UI: play/pause button, speed selector dropdown (0.25x-4x), timeline scrubber with round boundaries, tick counter, round selector dropdown. All controls update Zustand viewer store. |
| **Key Files** | `frontend/src/components/viewer/playback-controls.tsx`, `frontend/src/components/viewer/timeline.tsx`, `frontend/src/components/viewer/playback-controls.test.tsx` |
| **Acceptance Criteria** | - Play/pause toggles playback state |
| | - Speed selector changes playback speed |
| | - Timeline scrubber seeks to clicked position |
| | - Round boundaries visible on timeline |
| | - Tick counter shows current/total ticks |

### P3-T08: Implement round selector

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P3-T07 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write test: round selector lists all rounds; clicking jumps to round start tick. 2. GREEN: Implement dropdown with round number + score. 3. REFACTOR: Add current-round highlighting. |
| **Description** | Dropdown listing all rounds with score context. Clicking jumps playback to the round's start tick. |
| **Key Files** | `frontend/src/components/viewer/round-selector.tsx`, `frontend/src/components/viewer/round-selector.test.tsx` |
| **Acceptance Criteria** | - All rounds listed with round number and score |
| | - Clicking round jumps to correct start tick |
| | - Current round highlighted during playback |

### P3-T09: Implement zoom and pan

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T01 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write unit tests for zoom transforms (min 0.5x, max 4x) and pan bounds. 2. GREEN: Implement scroll-to-zoom, click-drag pan, mini-map. 3. REFACTOR: Extract viewport manager; add reset-view button. |
| **Description** | Scroll-to-zoom (0.5x to 4x), click-and-drag pan, mini-map in corner showing full map with viewport indicator, reset-view button. |
| **Key Files** | `frontend/src/lib/pixi/viewport.ts`, `frontend/src/lib/pixi/minimap.ts`, `frontend/src/lib/pixi/viewport.test.ts` |
| **Acceptance Criteria** | - Scroll zooms in/out within bounds |
| | - Click-drag pans the view |
| | - Mini-map shows current viewport position |
| | - Reset-view button restores default zoom and position |

### P3-T10: Build scoreboard overlay

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T03, P1-T10 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write test: scoreboard renders player stats for current round accurately. 2. GREEN: Implement toggle-able table overlay with shadcn/ui Table. 3. REFACTOR: Highlight current round row; add column sorting. |
| **Description** | Toggle-able scoreboard overlay showing per-round and match-total stats: Player, K, D, A, ADR, HS%, KAST, Rating. Highlight the current round. |
| **Key Files** | `frontend/src/components/viewer/scoreboard.tsx`, `frontend/src/components/viewer/scoreboard.test.tsx` |
| **Acceptance Criteria** | - Scoreboard shows accurate stats per player |
| | - Toggle button shows/hides overlay |
| | - Current round highlighted |
| | - Stats update when round changes |

### P3-T11: Implement keyboard shortcuts

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P3-T06, P3-T07 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write test: Space toggles play/pause; Left/Right skips 5s; Up/Down changes speed. 2. GREEN: Implement keyboard event handlers. 3. REFACTOR: Extract key bindings to config. |
| **Description** | Keyboard shortcuts: Space (play/pause), Left/Right (skip 5s), Up/Down (speed), Tab (toggle scoreboard), Escape (close overlays). |
| **Key Files** | `frontend/src/hooks/useViewerKeyboard.ts`, `frontend/src/hooks/useViewerKeyboard.test.ts` |
| **Acceptance Criteria** | - All keyboard shortcuts work as specified |
| | - Shortcuts don't fire when typing in input fields |
| | - Unit tests verify key bindings |

### P3-T12: Connect viewer Zustand store to PixiJS

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T04, P3-T05, P3-T06 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write test: store state changes propagate to PixiJS render state. 2. GREEN: Implement Zustand subscribe() bridge. 3. REFACTOR: Minimize subscriptions; use selectors for fine-grained updates. |
| **Description** | Bridge the viewer Zustand store to PixiJS layers using `subscribe()`. Store changes (current tick, speed, selected player, filters) update PixiJS without React re-renders. |
| **Key Files** | `frontend/src/lib/viewer/store-bridge.ts`, `frontend/src/lib/viewer/store-bridge.test.ts` |
| **Acceptance Criteria** | - Store state changes reflected in PixiJS render |
| | - No React re-renders triggered by store-to-PixiJS bridge |
| | - Selective subscriptions minimize unnecessary updates |

---

## 5. Phase 4: Faceit & Heatmaps

### P4-T01: Implement Faceit API client

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T03, P1-T09 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write tests for GetProfile, GetMatches, GetEloHistory with mock HTTP responses. 2. GREEN: Implement HTTP client with auth header injection. 3. REFACTOR: Add rate limiting; extract response types. |
| **Description** | Go HTTP client for Faceit Data API: get player profile, match history (paginated), ELO history. Inject access token from auth service. Handle rate limiting (429) with exponential backoff. |
| **Key Files** | `backend/internal/faceit/client.go`, `backend/internal/faceit/client_test.go`, `backend/internal/faceit/types.go` |
| **Acceptance Criteria** | - GetProfile returns typed Faceit profile |
| | - GetMatches returns paginated match list |
| | - GetEloHistory returns ELO data points |
| | - Rate limiting handled with backoff |
| | - Unit tests pass with mock HTTP responses |

### P4-T02: Implement Faceit sync service

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P4-T01 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write test: sync fetches new matches, upserts into SQLite, skips already-synced. 2. GREEN: Implement sync service with delta detection. 3. REFACTOR: Add progress events; handle partial failures. |
| **Description** | In-process Faceit match sync (replaces web version's Redis Streams worker). Fetch recent matches, compare with SQLite, upsert new ones. No background job -- runs synchronously when triggered. Exposed as `SyncMatches()` Wails binding. |
| **Key Files** | `backend/internal/faceit/sync.go`, `backend/internal/faceit/sync_test.go` |
| **Acceptance Criteria** | - New matches inserted into SQLite |
| | - Existing matches skipped (no duplicates) |
| | - ELO before/after calculated correctly |
| | - Progress events emitted during sync |
| | - Tests pass with mock Faceit client + temp SQLite |

### P4-T03: Build Faceit dashboard page

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P4-T01, P1-T10 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write test: dashboard renders profile card, ELO chart, win/loss streak. 2. GREEN: Implement page with TanStack Query wrapping Wails bindings. 3. REFACTOR: Extract chart component; add time range selector. |
| **Description** | Faceit dashboard showing: profile card (avatar, nickname, level, ELO, country), ELO history line chart (30/90/180/all time), win/loss streak indicator. |
| **Key Files** | `frontend/src/routes/dashboard.tsx`, `frontend/src/components/faceit/profile-card.tsx`, `frontend/src/components/faceit/elo-chart.tsx`, `frontend/src/routes/dashboard.test.tsx` |
| **Acceptance Criteria** | - Profile card shows correct Faceit data |
| | - ELO chart renders with selectable time ranges |
| | - Win/loss streak displayed |
| | - Component tests pass with mock bindings |

### P4-T04: Build match history list

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P4-T02, P1-T10 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write test: match list renders paginated entries; clicking match opens demo if available. 2. GREEN: Implement match list with filters. 3. REFACTOR: Extract match card; add infinite scroll. |
| **Description** | Paginated list of Faceit matches: map, score, K/D/A, ELO change, date. Filters: map, result (W/L), date range. Click to open demo in viewer (if imported). |
| **Key Files** | `frontend/src/components/faceit/match-list.tsx`, `frontend/src/components/faceit/match-card.tsx`, `frontend/src/components/faceit/match-list.test.tsx` |
| **Acceptance Criteria** | - Match list renders with pagination |
| | - Filters work (map, result, date range) |
| | - ELO change shows +/- delta |
| | - Click opens demo viewer (if available) |

### P4-T05: Implement demo download from Faceit

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P4-T02, P2-T05 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write test: given a Faceit match with demo URL, download file and trigger import. 2. GREEN: Implement download with progress events. 3. REFACTOR: Add retry logic; handle large files. |
| **Description** | Download `.dem` files from Faceit match data URLs. Save to a configurable local directory. Trigger import + parse after download. Exposed as `ImportMatchDemo(matchId)` binding. |
| **Key Files** | `backend/internal/faceit/download.go`, `backend/internal/faceit/download_test.go` |
| **Acceptance Criteria** | - Demo downloaded from Faceit URL to local directory |
| | - Progress events emitted during download |
| | - Import triggered automatically after download |
| | - Retry on transient failures |

### P4-T06: Implement heatmap data binding

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T07, P2-T08 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write test: given demo IDs and filters, binding returns aggregated position/event data. 2. GREEN: Implement SQLite aggregation queries. 3. REFACTOR: Optimize query performance; add caching. |
| **Description** | `GetHeatmapData()` binding: query tick_data and game_events tables with filters (demo IDs, map, side, weapon, player). Return aggregated position data suitable for KDE rendering. |
| **Key Files** | `backend/internal/heatmap/service.go`, `backend/internal/heatmap/service_test.go` |
| **Acceptance Criteria** | - Returns position data filtered by criteria |
| | - Multi-demo aggregation works correctly |
| | - Query performance < 500ms for 10-demo aggregate |
| | - Integration test verifies against known demo data |

### P4-T07: Implement client-side KDE rendering

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P4-T06, P3-T02 |
| **Test Types** | unit, screenshot |
| **TDD Workflow** | 1. RED: Write unit test for KDE algorithm (known input -> expected density output). 2. GREEN: Implement KDE on PixiJS canvas as color gradient overlay. 3. REFACTOR: Optimize with WebGL shader or offscreen canvas. |
| **Description** | Kernel Density Estimation rendered as color gradient overlay on the map image using PixiJS. Overlay updates when filters change. |
| **Key Files** | `frontend/src/lib/pixi/heatmap-layer.ts`, `frontend/src/lib/pixi/kde.ts`, `frontend/src/lib/pixi/kde.test.ts` |
| **Acceptance Criteria** | - KDE produces correct density gradient |
| | - Overlay renders on map at correct positions |
| | - Performance acceptable (< 2s render for single demo) |
| | - Unit test verifies KDE algorithm correctness |

### P4-T08: Build heatmap filter controls

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P4-T07, P1-T10 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write test: filter controls render; changing filters triggers re-fetch and re-render. 2. GREEN: Implement filter panel with shadcn/ui. 3. REFACTOR: Add demo multi-select; extract filter state. |
| **Description** | Filter panel: map selector, side (CT/T/both), weapon category, player name, demo multi-select. Changing filters re-fetches heatmap data and re-renders overlay. |
| **Key Files** | `frontend/src/components/heatmap/filter-panel.tsx`, `frontend/src/routes/heatmaps.tsx`, `frontend/src/components/heatmap/filter-panel.test.tsx` |
| **Acceptance Criteria** | - All filter controls render and function |
| | - Changing a filter triggers data re-fetch |
| | - Heatmap overlay updates after re-fetch |
| | - Demo multi-select allows aggregation |

### P4-T09: Build per-demo stats view

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T09, P1-T10 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write test: stats view renders correct K/D/A/ADR/HS%/KAST for demo. 2. GREEN: Implement stats page querying player_rounds data. 3. REFACTOR: Add charts; improve layout. |
| **Description** | Per-demo statistics page: K/D/A, ADR, HS%, KAST, Rating per player. Weapon breakdown. Round-by-round performance. |
| **Key Files** | `frontend/src/components/viewer/stats-panel.tsx`, `frontend/src/components/viewer/stats-panel.test.tsx` |
| **Acceptance Criteria** | - Stats match demo data accurately |
| | - All stat columns render correctly |
| | - Weapon breakdown shows kill distribution |

---

## 6. Phase 5: Strategy Board & Lineups

### P5-T01: Implement drawing canvas

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P3-T02 |
| **Test Types** | unit, screenshot |
| **TDD Workflow** | 1. RED: Write test for canvas initialization with map background. 2. GREEN: Implement PixiJS or Canvas 2D drawing surface with map layer. 3. REFACTOR: Extract layer management; add tool state machine. |
| **Description** | Drawing canvas with map background. Supports layered rendering: background (map image), strategy layer (drawings), annotation layer (text). Tool state machine for switching between drawing modes. |
| **Key Files** | `frontend/src/components/strat/strat-canvas.tsx`, `frontend/src/lib/strat/canvas-manager.ts`, `frontend/src/lib/strat/canvas-manager.test.ts` |
| **Acceptance Criteria** | - Canvas renders with map background |
| | - Layer system supports multiple drawing layers |
| | - Tool state machine switches between modes |
| | - Canvas resizes with window |

### P5-T02: Implement drawing tools

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P5-T01 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write tests for each tool's draw logic (freehand produces path points, line creates two endpoints, etc.). 2. GREEN: Implement tools: freehand, line, arrow, rectangle, circle, text. 3. REFACTOR: Extract tool interface; share color/width state. |
| **Description** | Drawing tools: freehand (path), line, arrow, rectangle, circle, text label. Color picker with preset team colors + custom. Line width selector. Eraser tool. |
| **Key Files** | `frontend/src/lib/strat/tools/*.ts`, `frontend/src/lib/strat/tools/*.test.ts`, `frontend/src/components/strat/toolbar.tsx` |
| **Acceptance Criteria** | - Each tool produces correct geometric output |
| | - Color picker and line width work |
| | - Eraser removes elements |
| | - Unit tests pass for each tool's logic |

### P5-T03: Implement strategy primitives

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T01 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write tests for player token drag, grenade marker placement. 2. GREEN: Implement draggable tokens and markers. 3. REFACTOR: Extract primitive types; add snap-to-grid option. |
| **Description** | Strategy-specific elements: player tokens (draggable, labeled CT1-CT5/T1-T5), grenade trajectory lines, smoke/molotov/flash markers, timing waypoints. |
| **Key Files** | `frontend/src/lib/strat/primitives.ts`, `frontend/src/lib/strat/primitives.test.ts` |
| **Acceptance Criteria** | - Player tokens draggable with team labels |
| | - Grenade markers render with type icons |
| | - Timing waypoints show execute order |

### P5-T04: Implement undo/redo

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T02 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write tests: draw -> undo reverts; redo restores; branching history works. 2. GREEN: Implement command stack with undo/redo. 3. REFACTOR: Limit stack depth; handle edge cases. |
| **Description** | In-memory command stack for undo/redo (replacing Yjs UndoManager from web version). Ctrl+Z/Cmd+Z to undo, Ctrl+Shift+Z/Cmd+Shift+Z to redo. Branching: new action after undo discards redo stack. |
| **Key Files** | `frontend/src/lib/strat/history.ts`, `frontend/src/lib/strat/history.test.ts` |
| **Acceptance Criteria** | - Undo reverts last action |
| | - Redo restores undone action |
| | - Branching history works correctly |
| | - Keyboard shortcuts trigger undo/redo |

### P5-T05: Implement board persistence

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T01, P1-T03 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write test: save board state as JSON to SQLite; load it back; verify equality. 2. GREEN: Implement serialize/deserialize + SaveBoard/GetBoard bindings. 3. REFACTOR: Add autosave with debouncing. |
| **Description** | Serialize board state (all drawings, tokens, annotations) to JSON. Save to `strategy_boards.board_state` via `SaveBoard()` binding. Autosave on changes (debounced 1s). Load on board open. |
| **Key Files** | `backend/internal/strat/service.go`, `backend/internal/strat/service_test.go`, `frontend/src/hooks/useAutoSave.ts` |
| **Acceptance Criteria** | - Board state round-trips through JSON correctly |
| | - Autosave triggers on changes (debounced) |
| | - Board survives app restart |
| | - Integration test verifies save/load cycle |

### P5-T06: Build board list + create/delete UI

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P5-T05, P1-T10 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write test: board list renders; create dialog works; delete shows confirmation. 2. GREEN: Implement list page + create/delete flows. 3. REFACTOR: Add board preview thumbnails. |
| **Description** | Strategy board list page: grid of boards with map name and title. Create dialog (select map, enter title). Delete with confirmation. |
| **Key Files** | `frontend/src/routes/strats/index.tsx`, `frontend/src/routes/strats/index.test.tsx` |
| **Acceptance Criteria** | - Board list renders from `ListBoards()` binding |
| | - Create dialog produces new board |
| | - Delete shows confirmation and removes board |

### P5-T07: Implement PNG export

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P5-T01 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write test: export produces a valid PNG data URL. 2. GREEN: Implement canvas-to-PNG export. 3. REFACTOR: Add resolution options. |
| **Description** | Export the strategy board canvas as a PNG image. Use PixiJS/Canvas `toDataURL()` or `toBlob()`. Trigger native save dialog via Wails. |
| **Key Files** | `frontend/src/lib/strat/export.ts`, `frontend/src/lib/strat/export.test.ts` |
| **Acceptance Criteria** | - PNG export captures full board state |
| | - Save dialog opens with suggested filename |
| | - Exported image matches canvas content |

### P5-T08: Implement JSON import/export

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P5-T05 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write test: export board to JSON string; import from JSON creates new board. 2. GREEN: Implement ExportBoardJSON/ImportBoardJSON bindings. 3. REFACTOR: Add validation for imported JSON. |
| **Description** | Export board as JSON file (for sharing via Discord, etc.). Import board from JSON file. Replaces web version's share links. |
| **Key Files** | `backend/internal/strat/export.go`, `backend/internal/strat/export_test.go` |
| **Acceptance Criteria** | - JSON export produces valid, importable JSON |
| | - Import creates new board from JSON |
| | - Invalid JSON shows descriptive error |
| | - Round-trip: export -> import produces identical board |

### P5-T09: Add grenade extraction to demo parser

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P2-T06 |
| **Test Types** | unit, golden |
| **TDD Workflow** | 1. RED: Write golden file test for grenade extraction (known throws from test demo). 2. GREEN: Add grenade throw/detonate event handlers to parser. 3. REFACTOR: Correlate throw with landing position. |
| **Description** | Extend demo parser to extract grenade throws: thrower position, aim angles, grenade type, landing/detonation position. Correlate throw events with detonation events. Insert into `grenade_lineups` table. |
| **Key Files** | `backend/internal/demo/grenades.go`, `backend/internal/demo/grenades_test.go` |
| **Acceptance Criteria** | - Grenade throws extracted with correct positions and angles |
| | - Throw correlated with landing position |
| | - All grenade types handled (smoke, flash, HE, molotov) |
| | - Golden file tests pass |

### P5-T10: Build lineup catalog page

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T09, P1-T10 |
| **Test Types** | component |
| **TDD Workflow** | 1. RED: Write test: catalog renders lineups; filters work; clicking shows detail. 2. GREEN: Implement catalog with grid layout, filters, search. 3. REFACTOR: Add 2D mini-preview for each lineup. |
| **Description** | Lineup catalog: browse by map, grenade type, site. Each entry shows 2D preview (throw position + landing on mini-map), description, tags. Search and filter. Link to source tick in demo viewer. |
| **Key Files** | `frontend/src/routes/lineups/index.tsx`, `frontend/src/components/lineups/lineup-card.tsx`, `frontend/src/routes/lineups/index.test.tsx` |
| **Acceptance Criteria** | - Catalog renders lineups from `ListLineups()` binding |
| | - Filters work (map, grenade type) |
| | - Mini-preview shows throw/landing positions on map |
| | - "View in Demo" links to viewer at correct tick |

### P5-T11: Implement lineup CRUD + favorites

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P5-T09, P1-T03 |
| **Test Types** | unit, integration |
| **TDD Workflow** | 1. RED: Write tests for UpdateLineup, DeleteLineup, ToggleFavorite. 2. GREEN: Implement CRUD bindings + SQLite queries. 3. REFACTOR: Add tag management; bulk operations. |
| **Description** | Lineup management: edit title/description/tags, delete, toggle favorite. Custom notes per lineup. Tag-based filtering. |
| **Key Files** | `backend/internal/lineup/service.go`, `backend/internal/lineup/service_test.go` |
| **Acceptance Criteria** | - Update changes title, description, tags |
| | - Delete removes lineup |
| | - Toggle favorite flips is_favorite flag |
| | - Tags stored as JSON array; filterable |
| | - Integration tests pass with temp SQLite |

---

## 7. Phase 6: Polish & Distribute

### P6-T01: Performance profiling and optimization (frontend)

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P3-T12, P4-T07 |
| **Test Types** | unit (benchmarks) |
| **TDD Workflow** | Profile first, optimize second. Add benchmark tests for critical render paths. |
| **Description** | Profile PixiJS rendering (Chrome DevTools Performance tab). Optimize: sprite pooling, draw call batching, LOD at low zoom, texture atlas for player sprites. Ensure stable 60 FPS with all layers active. Profile heatmap KDE rendering. |
| **Key Files** | Various `frontend/src/lib/pixi/*.ts` files |
| **Acceptance Criteria** | - Viewer maintains 60 FPS with 10 players + events + effects |
| | - Heatmap renders in < 2s (single demo) |
| | - No memory leaks on repeated demo loads |

### P6-T02: Performance profiling and optimization (backend)

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P2-T07, P4-T06 |
| **Test Types** | unit (benchmarks) |
| **TDD Workflow** | Benchmark critical SQLite queries. Profile demo parsing. Optimize based on data. |
| **Description** | Profile demo parsing time and memory. Benchmark SQLite queries: tick range scans, event queries, heatmap aggregation. Optimize: query plans (EXPLAIN), index usage, transaction batch sizes. Ensure < 10s parse, < 50ms tick query. |
| **Key Files** | Various `backend/internal/**/*_test.go` (benchmark functions) |
| **Acceptance Criteria** | - Demo parse < 10s for 100 MB demo |
| | - Tick range query < 50ms |
| | - Heatmap aggregation < 500ms for 10 demos |
| | - Memory usage < 500 MB during parsing |

### P6-T03: Cross-platform WebView testing

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P3-T01 |
| **Test Types** | e2e |
| **TDD Workflow** | N/A -- manual testing + automated E2E on each platform |
| **Description** | Test PixiJS WebGL rendering on all three WebView engines: WebKit (macOS), WebView2 (Windows), WebKitGTK (Linux). Verify: canvas rendering, scroll/zoom, keyboard shortcuts, file dialogs, OAuth browser launch. |
| **Key Files** | `e2e/*.spec.ts` |
| **Acceptance Criteria** | - PixiJS renders correctly on all three platforms |
| | - No WebGL capability gaps on minimum OS versions |
| | - File dialogs work on each platform |
| | - OAuth loopback flow works on each platform |

### P6-T04: Implement auto-update checker

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T01 |
| **Test Types** | unit |
| **TDD Workflow** | 1. RED: Write test: given a GitHub Releases response with newer version, returns UpdateInfo. 2. GREEN: Implement version check + download link extraction. 3. REFACTOR: Add check frequency setting; dismissible notification. |
| **Description** | On startup, check GitHub Releases API for newer version. Show non-intrusive notification if available. User-initiated download (open browser to release page). Configurable check frequency. |
| **Key Files** | `backend/internal/app/updater.go`, `backend/internal/app/updater_test.go` |
| **Acceptance Criteria** | - Detects newer version via GitHub Releases API |
| | - Shows notification with version info |
| | - User can dismiss or open download page |
| | - Check frequency respects user setting |

### P6-T05: Create macOS build + .dmg

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T01 |
| **Test Types** | N/A (packaging) |
| **TDD Workflow** | N/A -- verify via manual install test |
| **Description** | Configure Wails build for macOS. Create `.dmg` installer with drag-to-Applications layout. Set `Info.plist` metadata (version, bundle ID, icon). |
| **Key Files** | `build/darwin/*`, `wails.json` (macOS build config) |
| **Acceptance Criteria** | - `wails build -platform darwin/universal` produces `.app` bundle |
| | - `.dmg` contains app + Applications alias |
| | - App icon renders correctly |
| | - App launches on macOS 12+ |

### P6-T06: Create Windows build + installer

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T01 |
| **Test Types** | N/A (packaging) |
| **TDD Workflow** | N/A -- verify via manual install test |
| **Description** | Configure Wails build for Windows. Create NSIS installer with install/uninstall, start menu shortcut, optional desktop shortcut. Set version info and icon in manifest. Ensure WebView2 bootstrap is included. |
| **Key Files** | `build/windows/*`, `wails.json` (Windows build config) |
| **Acceptance Criteria** | - `wails build -platform windows/amd64` produces `.exe` |
| | - NSIS installer installs/uninstalls cleanly |
| | - WebView2 bootstrapper runs if not present |
| | - App launches on Windows 10 1903+ |

### P6-T07: Create Linux build + AppImage

| | |
|---|---|
| **Complexity** | M |
| **Deps** | P1-T01 |
| **Test Types** | N/A (packaging) |
| **TDD Workflow** | N/A -- verify via manual install test |
| **Description** | Configure Wails build for Linux. Create `.AppImage` for universal distribution. Include `.desktop` file and icon. Verify WebKitGTK dependency documented. |
| **Key Files** | `build/linux/*`, `wails.json` (Linux build config) |
| **Acceptance Criteria** | - `wails build -platform linux/amd64` produces binary |
| | - `.AppImage` runs on Ubuntu 22.04+ |
| | - WebKitGTK dependency clearly documented |

### P6-T08: Set up code signing

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P6-T05, P6-T06 |
| **Test Types** | N/A (infrastructure) |
| **TDD Workflow** | N/A -- verify via successful notarization/signing |
| **Description** | macOS: Apple Developer certificate + notarization (codesign + notarytool). Windows: code signing certificate (EV or OV). Integrate signing into CI build pipeline. |
| **Key Files** | `.github/workflows/release.yml` |
| **Acceptance Criteria** | - macOS app passes Gatekeeper (notarized) |
| | - Windows app passes SmartScreen (signed) |
| | - CI pipeline signs automatically on release |

### P6-T09: End-to-end testing of critical paths

| | |
|---|---|
| **Complexity** | L |
| **Deps** | P4-T09, P5-T11 |
| **Test Types** | e2e |
| **TDD Workflow** | Write E2E tests for critical user flows: import demo, watch playback, view heatmap, draw strategy, browse lineups. |
| **Description** | Playwright E2E tests against Wails dev instance. Critical paths: import demo + verify library, open viewer + play/pause/seek, filter heatmap, create strategy board + draw, browse lineups. |
| **Key Files** | `e2e/demo-import.spec.ts`, `e2e/viewer.spec.ts`, `e2e/heatmap.spec.ts`, `e2e/strat-board.spec.ts`, `e2e/lineups.spec.ts` |
| **Acceptance Criteria** | - All critical path E2E tests pass |
| | - Tests run in CI |
| | - Coverage: import, viewer, heatmap, strat board, lineups |

### P6-T10: Write README.md and contributing guide

| | |
|---|---|
| **Complexity** | S |
| **Deps** | P6-T05, P6-T06, P6-T07 |
| **Test Types** | N/A (documentation) |
| **TDD Workflow** | N/A |
| **Description** | Write project README with: feature overview, screenshots, download links, development setup, building from source, contributing guidelines. |
| **Key Files** | `README.md`, `CONTRIBUTING.md` |
| **Acceptance Criteria** | - README covers installation, features, development |
| | - Build from source instructions work |
| | - Contributing guide covers TDD workflow and PR process |

---

## 8. Critical Path Analysis

### Longest Path

```
P1-T01 → P1-T02 → P1-T03 → P1-T09 → P2-T05 → P2-T06 → P2-T07 → P3-T03 → P3-T06 → P6-T01
  M         M         M         M         M         XL        L         L         L        L
```

**Critical path items:**

| Task | Why Critical |
|------|-------------|
| **P1-T09** (Go Test Infrastructure) | Test infrastructure must be ready before any TDD task in P2+ |
| **P2-T06** (Demo Parser Core) | XL complexity; all viewer features depend on parsed data; highest risk of delays |
| **P2-T07** (Tick Data Ingestion) | Viewer can't render without tick data in SQLite |
| **P3-T03** (Tick Data Fetching) | Client-side buffer needed before any rendering |
| **P3-T06** (Playback Engine) | Core viewer functionality; all other viewer features depend on it |

### Bottleneck: P2-T06 Demo Parser

This single task is the project's biggest risk. The same mitigation applies as the web version:

1. **Start a spike early**: Before Phase 2, spend a day prototyping with `demoinfocs-golang`
2. **Test with real demos**: Collect 5+ CS2 demo files of varying size and complexity
3. **Incremental extraction**: Parse positions first, add events second, add edge cases third
4. **Benchmark continuously**: Track parse time and memory usage
5. **Golden file tests early**: Write golden file tests against spike output before full implementation

---

## 9. Risk Register

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|------------|
| R1 | **demoinfocs-golang doesn't support latest CS2 demo format** | Medium | Critical | Monitor library's GitHub issues; contribute patches; keep parser modular |
| R2 | **System WebView inconsistencies across platforms** | Medium | High | Test PixiJS on all three WebView engines early (P3); maintain platform-specific workarounds list; set minimum OS versions |
| R3 | **modernc.org/sqlite performance insufficient for tick data** | Low | High | Benchmark early; tune batch size and transaction scope; fall back to mattn/go-sqlite3 (CGo) if needed |
| R4 | **PixiJS 60 FPS not achievable in system WebView** | Low | High | Object pooling; sprite batching; LOD; profile WebGL on each platform early |
| R5 | **Faceit API rate limiting disrupts sync** | Medium | Medium | Aggressive local caching; exponential backoff; batch API calls |
| R6 | **Map coordinate calibration inaccurate** | Low | High | Verify against known positions; allow manual adjustment; community-sourced data |
| R7 | **Code signing certificates expensive or unavailable** | Medium | Medium | macOS Developer Program ($99/yr); Windows EV cert ($200-400/yr); budget for certs; consider unsigned builds for early testers |
| R8 | **Auto-updater security (man-in-the-middle)** | Low | High | HTTPS-only update checks; verify signature on downloaded binaries; use GitHub Releases (trusted CDN) |
| R9 | **TDD overhead slows early velocity** | Medium | Medium | Start with highest-value tests (parser golden files, auth); defer low-value tests; keep unit tests < 30s |
| R10 | **Loopback OAuth blocked by firewalls** | Low | Medium | Clear error message + troubleshooting guide; configurable port range; future fallback to manual token entry |

---

## 10. Development Environment Setup

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Backend development |
| Node.js | 20 LTS | Frontend development |
| pnpm | 9+ | Package manager |
| Wails CLI | v2 (latest) | Desktop app framework |
| Git | 2.40+ | Version control (worktree support) |
| Playwright | Latest | E2E testing |

**Platform-specific:**
- **macOS**: Xcode Command Line Tools (for WebKit)
- **Windows**: WebView2 Runtime (usually pre-installed on Win 10+)
- **Linux**: `libwebkit2gtk-4.1-dev`, `libgtk-3-dev`, `build-essential`

### Clone to Running (First Time)

```bash
# 1. Clone bare repo + create worktree
git clone --bare git@github.com:ok2ju/oversite.git oversite
cd oversite
git worktree add ../oversite-main main
cd ../oversite-main

# 2. Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 3. Install frontend dependencies
cd frontend && pnpm install && cd ..

# 4. Start development (hot reload for Go + frontend)
wails dev
# or: make dev

# 5. App window opens automatically
# SQLite database created in OS app data directory on first run
```

### Common Development Commands

```bash
make dev              # Start Wails dev mode (hot reload)
make build            # Build production binary
make test             # Run all tests (Go + TS)
make test-unit        # Go + TS unit tests only
make test-e2e         # Playwright E2E tests
make lint             # Run all linters (Go + TS)
make typecheck        # TypeScript type checking
make sqlc             # Regenerate sqlc Go code
make migrate-up       # Run pending migrations
make migrate-down     # Rollback last migration
make clean            # Remove build artifacts
```

---

## 11. Sprint Pairing Recommendations

### Natural Sprint Groupings (Solo Dev)

| Sprint | Tasks | Focus |
|--------|-------|-------|
| 1 | P1-T01 through P1-T10 | Desktop foundation + test infrastructure |
| 2 | P2-T01 through P2-T04 | Auth (OAuth + keychain + UI) |
| 3 | P2-T05 through P2-T09 | Demo import + parser + ingestion |
| 4 | P2-T10, P2-T11, P3-T01, P3-T02 | Demo library UI + viewer setup |
| 5 | P3-T03 through P3-T06 | Viewer core rendering + playback |
| 6 | P3-T07 through P3-T12 | Viewer UI + polish |
| 7 | P4-T01 through P4-T05 | Faceit integration |
| 8 | P4-T06 through P4-T09 | Heatmaps + analytics |
| 9 | P5-T01 through P5-T08 | Strategy board |
| 10 | P5-T09 through P5-T11 | Lineups |
| 11 | P6-T01 through P6-T10 | Polish + distribute |

### Parallel Tracks (After P2 Complete)

**Track A: Viewer Rendering**
- P3-T03 -> P3-T04 -> P3-T05 -> P3-T06 -> P3-T12

**Track B: Viewer UI**
- P3-T07 -> P3-T10 -> P3-T11

**Track C: Faceit Backend** (can start early)
- P4-T01 -> P4-T02 -> P4-T05

### Parallel Tracks (After P3 Complete)

**Track A: Heatmaps**
- P4-T06 -> P4-T07 -> P4-T08

**Track B: Faceit Dashboard**
- P4-T03 -> P4-T04 -> P4-T09

**Track C: Strategy Board Foundation**
- P5-T01 -> P5-T02 -> P5-T04

---

*Cross-references: [PRD.md](PRD.md) for feature requirements, [ARCHITECTURE.md](ARCHITECTURE.md) for system design, [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for phase milestones.*
