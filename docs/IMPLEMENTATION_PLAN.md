# Oversite -- Implementation Plan

> **Version:** 2.0
> **Last Updated:** 2026-04-12

---

## Table of Contents

1. [Git Worktree Strategy](#1-git-worktree-strategy)
2. [TDD Methodology](#2-tdd-methodology)
3. [Phase Overview](#3-phase-overview)
4. [Phase 1: Desktop Foundation](#4-phase-1-desktop-foundation)
5. [Phase 2: Auth & Demo Pipeline](#5-phase-2-auth--demo-pipeline)
6. [Phase 3: Core 2D Viewer](#6-phase-3-core-2d-viewer)
7. [Phase 4: Faceit & Heatmaps](#7-phase-4-faceit--heatmaps)
8. [Phase 5: Strategy Board & Lineups](#8-phase-5-strategy-board--lineups)
9. [Phase 6: Polish & Distribute](#9-phase-6-polish--distribute)
10. [Dependency Graph](#10-dependency-graph)

---

## 1. Git Worktree Strategy

The repository uses a **bare clone + worktree** model. The bare repo at `oversite/` contains no working tree; all development happens in worktrees.

### Branch Naming Convention

| Type | Pattern | Example |
|------|---------|---------|
| Main | `main` | `main` |
| Phase | `phase/{n}-{name}` | `phase/1-desktop-foundation` |
| Feature | `feat/{phase}-{short-desc}` | `feat/p2-demo-parser` |
| Bug fix | `fix/{short-desc}` | `fix/tick-data-offset` |
| Docs | `docs/{short-desc}` | `docs/desktop-pivot` |

### Worktree Workflow

```bash
# Create worktree for a feature
git worktree add ../oversite-feat-demo-parser feat/p2-demo-parser

# Work in the worktree
cd ../oversite-feat-demo-parser

# When done, merge and clean up
git checkout main && git merge feat/p2-demo-parser
git worktree remove ../oversite-feat-demo-parser
```

### Merge Strategy

- Feature branches merge into `main` via squash merge (clean history)
- Phase branches are bookmarks -- tag `main` at each phase milestone
- No long-lived develop branch; `main` is always buildable after Phase 1

---

## 2. TDD Methodology

The project follows **Test-Driven Development** from the first line of code. Every feature is built using the Red-Green-Refactor cycle.

### 2.1 Red-Green-Refactor Workflow

```
    ┌─────────────────────────────────────┐
    │                                     │
    ▼                                     │
┌───────┐      ┌───────┐      ┌──────────┴──┐
│  RED  │─────▶│ GREEN │─────▶│  REFACTOR   │
│       │      │       │      │             │
│ Write │      │ Write │      │ Clean up,   │
│failing│      │minimal│      │ DRY, rename │
│ test  │      │ code  │      │ — tests     │
│       │      │to pass│      │ stay green  │
└───────┘      └───────┘      └─────────────┘
```

1. **RED**: Write a test that defines the expected behavior. Run it -- it must fail.
2. **GREEN**: Write the minimum code to make the test pass. No more.
3. **REFACTOR**: Clean up the implementation and the test. All tests must remain green.
4. **COMMIT**: Commit after each green-to-refactor cycle.

### 2.2 TDD Applicability by Layer

| Layer | TDD Approach | Rationale |
|-------|-------------|-----------|
| Go services | Classical TDD | Pure business logic; interfaces enable mocking |
| Go Wails bindings | TDD with direct calls | Test service methods directly, not HTTP |
| Go demo parser | Golden-file TDD + spike | Spike validates external library, then golden tests |
| sqlc queries | TDD with temp SQLite | `:memory:` or temp file SQLite per test |
| Zustand stores | Classical TDD | Pure state + actions, no I/O |
| React components | TDD with React Testing Library | Render + interaction tests written first |
| TanStack Query hooks | TDD with mock bindings | Mock Wails-generated TS functions |
| PixiJS rendering | Test-alongside | TDD the logic (transforms, interpolation); screenshot-test the visuals |
| E2E flows | Test-alongside | Written after features work, not strict Red-Green-Refactor |

### 2.3 Test Infrastructure Requirements per Phase

| Phase | Test Infrastructure Added | CI Gate |
|-------|--------------------------|---------|
| **P1** | Temp SQLite test helpers, Vitest + RTL + MSW, Playwright config | `go test` + `pnpm test` pass |
| **P2** | Demo fixture files, golden file framework, mock keyring | Golden tests pass; auth tests pass |
| **P3** | PixiJS screenshot test pipeline, coordinate fixture data | Screenshot comparison stable |
| **P4** | Faceit API mock handlers (MSW), KDE test fixtures | Mock API tests pass |
| **P5** | Drawing tool logic test fixtures | Drawing logic tests pass |
| **P6** | Full E2E test suite, coverage reporting, cross-platform CI | Coverage targets met; all E2E pass |

### 2.4 Testing Milestones Convention

Every phase includes a **testing milestone** (`Px-MT`) that must be met before the phase is considered complete.

---

## 3. Phase Overview

```
Phase 1          Phase 2             Phase 3           Phase 4          Phase 5            Phase 6
Desktop          Auth & Demo         Core 2D           Faceit &         Strategy Board     Polish &
Foundation       Pipeline            Viewer             Heatmaps         & Lineups          Distribute
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Wails scaffold   Loopback OAuth      PixiJS setup       Faceit API       Drawing canvas     Cross-platform
SQLite setup     Keychain tokens     Map rendering      client           Drawing tools      Auto-updater
Vite + React     Demo import         Player rendering   Auto-fetch       Strat primitives   Installers
react-router     Demo parser         Event rendering    Heatmap KDE      Grenade catalog    Code signing
sqlc (SQLite)    Parse → SQLite      Playback controls  Stats charts     Lineup CRUD        Performance
CI pipeline      Demo library UI     Scoreboard         Dashboard UI     Board mgmt         Docs
```

| Phase | Name | Key Milestone | Dependencies |
|-------|------|---------------|-------------|
| **P1** | Desktop Foundation | `wails dev` runs app; CI passes | None |
| **P2** | Auth & Demo Pipeline | Import local `.dem` -> parsed data in SQLite | P1 |
| **P3** | Core 2D Viewer | Play back a demo at 60 FPS in the app | P2 |
| **P4** | Faceit & Heatmaps | Faceit stats dashboard + KDE heatmaps render | P2, P3 |
| **P5** | Strategy Board & Lineups | Drawing tools + lineup catalog | P3 |
| **P6** | Polish & Distribute | Cross-platform builds, installer, auto-updater | P4, P5 |

---

## 4. Phase 1: Desktop Foundation

**Goal**: Wails project scaffolding, SQLite database, Vite + React frontend, CI pipeline. After this phase, `wails dev` launches the app with hot reload, and CI runs lint + test + build.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P1-M1 | Wails project structure | `wails dev` launches app with WebView window |
| P1-M2 | SQLite database works | Migrations create all tables; sqlc generates Go code |
| P1-M3 | Frontend scaffold | Vite + React + react-router renders shell with sidebar |
| P1-M4 | UI framework | shadcn/ui + Tailwind configured; theme switching works |
| P1-M5 | CI pipeline passes | Lint, test, build all green |
| P1-MT | Test infrastructure works | `go test ./...` runs with temp SQLite; `pnpm test` runs; CI runs both |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P1-T01 | Initialize Wails project (Go module, wails.json, project structure) | M |
| P1-T02 | Set up SQLite with modernc.org/sqlite (connection, WAL mode, migrations) | M |
| P1-T03 | Configure sqlc for SQLite dialect and generate Go code | M |
| P1-T04 | Scaffold Vite + React frontend (react-router-dom, app shell) | M |
| P1-T05 | Configure shadcn/ui + Tailwind CSS (theme, components) | S |
| P1-T06 | Set up Zustand stores (skeleton: viewer, strat, ui, faceit, demo) | S |
| P1-T07 | Set up CI pipeline (lint, test, build for Go + TS) | M |
| P1-T08 | Create root Makefile with dev commands | S |
| P1-T09 | Set up Go test infrastructure (temp SQLite helper, mock interfaces, test utils) | M |
| P1-T10 | Set up frontend test infrastructure (Vitest config, RTL, MSW, mock bindings) | M |

---

## 5. Phase 2: Auth & Demo Pipeline

**Goal**: Users can log in with Faceit via loopback OAuth, import local `.dem` files, and have them parsed into queryable SQLite data.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P2-M1 | Faceit OAuth works | User can log in via system browser; tokens stored in keychain |
| P2-M2 | Demo import works | Local `.dem` file imported; demo record in SQLite |
| P2-M3 | Demo parsing works | Parser runs in-process; tick data + events in SQLite |
| P2-M4 | Demo library works | UI lists user's demos with status; drag-drop works |
| P2-MT | Tests pass | Parser golden file tests pass; auth tests pass; import tests pass |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P2-T01 | Implement loopback OAuth flow (temp HTTP listener, PKCE, browser open) | M |
| P2-T02 | Implement keychain token storage (go-keyring integration) | S |
| P2-T03 | Create auth service (login, logout, token refresh, get current user) | M |
| P2-T04 | Create AuthProvider + login page (React) | M |
| P2-T05 | Implement demo import binding (file validation, SQLite insert) | M |
| P2-T06 | **Implement demo parser core** (demoinfocs-golang integration) | **XL** |
| P2-T07 | Parse ticks -> batch insert into SQLite (10K-row transactions) | L |
| P2-T08 | Parse events -> insert game_events (kills, grenades, bombs) | L |
| P2-T09 | Parse rounds -> insert rounds + player_rounds | M |
| P2-T10 | Build demo library UI (list, grid, drag-drop, status, delete) | M |
| P2-T11 | Implement folder import binding (recursive .dem scan) | S |

### Critical Path Note

**P2-T06 (Demo Parser Core)** is the highest-risk, highest-complexity task. The `demoinfocs-golang` library requires careful integration. This task carries over from the web version -- the parser logic is identical, only the output target changes (SQLite transactions instead of PostgreSQL via worker).

---

## 6. Phase 3: Core 2D Viewer

**Goal**: Users can watch a parsed demo in a 2D top-down view with full playback controls. Data comes from SQLite via Wails bindings instead of REST API.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P3-M1 | Map renders | Correct radar image displayed, scaled to canvas |
| P3-M2 | Players render | All 10 players shown with correct positions each tick |
| P3-M3 | Events render | Kills, grenades, bomb events visible on map |
| P3-M4 | Playback works | Play/pause, speed, seek, round select all functional |
| P3-M5 | Scoreboard works | Accurate stats overlay for current round |
| P3-MT | Tests pass | Playback engine unit tests pass; coordinate transform tests pass |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P3-T01 | Set up PixiJS Application + canvas container | M |
| P3-T02 | Implement map layer (radar images, coordinate calibration) | M |
| P3-T03 | Implement tick data fetching (Wails binding + client-side buffer) | L |
| P3-T04 | Implement player layer (circles, names, view angles) | M |
| P3-T05 | Implement event layer (kills, grenades, bombs) | L |
| P3-T06 | Implement playback engine (tick interpolation, speed control) | L |
| P3-T07 | Build playback controls UI (play/pause, speed, timeline) | M |
| P3-T08 | Implement round selector | S |
| P3-T09 | Implement zoom and pan (+ mini-map) | M |
| P3-T10 | Build scoreboard overlay | M |
| P3-T11 | Implement keyboard shortcuts | S |
| P3-T12 | Connect viewer Zustand store to PixiJS render loop | M |

---

## 7. Phase 4: Faceit & Heatmaps

**Goal**: Faceit stats dashboard with ELO history + interactive KDE heatmaps. Faceit sync runs in-process (no worker/queue).

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P4-M1 | Faceit profile + ELO chart | Dashboard shows profile, ELO line chart |
| P4-M2 | Match history | Paginated match list with links to demos |
| P4-M3 | Match sync | In-process sync fetches Faceit matches on demand |
| P4-M4 | Kill heatmap | KDE overlay renders correctly for a demo |
| P4-M5 | Aggregated heatmap | Multi-demo heatmap with filters |
| P4-MT | Tests pass | Faceit client mock tests pass; KDE algorithm unit tests pass |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P4-T01 | Implement Faceit API client (Go HTTP client) | M |
| P4-T02 | Implement Faceit sync service (in-process, no worker) | M |
| P4-T03 | Build Faceit dashboard page (profile, ELO chart) | M |
| P4-T04 | Build match history list (pagination, filters) | M |
| P4-T05 | Implement demo download from Faceit matches | L |
| P4-T06 | Implement heatmap data binding (aggregation query) | M |
| P4-T07 | Implement client-side KDE rendering on PixiJS canvas | L |
| P4-T08 | Build heatmap filter controls (map, side, weapon, player) | M |
| P4-T09 | Build per-demo stats view | M |

---

## 8. Phase 5: Strategy Board & Lineups

**Goal**: Single-user strategy drawing tools with local persistence, and a grenade lineup catalog auto-populated from demos.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P5-M1 | Drawing canvas | Map background with drawing surface renders |
| P5-M2 | Drawing tools complete | Freehand, shapes, arrows, text, player tokens, eraser |
| P5-M3 | Persistence works | Board state saved to SQLite; survives app restart |
| P5-M4 | Lineup catalog works | Auto-extracted lineups browsable by map/type |
| P5-M5 | Personal collection | Users can save, tag, favorite lineups |
| P5-MT | Tests pass | Drawing tool logic tests pass; lineup CRUD tests pass |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P5-T01 | Implement drawing canvas (PixiJS or Canvas 2D, map background) | L |
| P5-T02 | Implement drawing tools (freehand, line, arrow, rect, circle, text) | L |
| P5-T03 | Implement strategy primitives (player tokens, grenade markers) | M |
| P5-T04 | Implement undo/redo (in-memory command stack) | M |
| P5-T05 | Implement board persistence (JSON state <-> SQLite) | M |
| P5-T06 | Build board list + create/delete UI | S |
| P5-T07 | Implement PNG export | S |
| P5-T08 | Implement JSON import/export (board sharing) | S |
| P5-T09 | Add grenade extraction to demo parser | M |
| P5-T10 | Build lineup catalog page (browse, filter, search) | M |
| P5-T11 | Implement lineup CRUD + favorites | M |

---

## 9. Phase 6: Polish & Distribute

**Goal**: Cross-platform testing, auto-updater, installers, code signing, performance optimization, and documentation.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P6-M1 | Performance targets met | 60 FPS viewer, < 10s parse, < 3s startup |
| P6-M2 | Cross-platform verified | App works on macOS, Windows, Linux |
| P6-M3 | Installers built | .dmg, .exe, .AppImage produced by CI |
| P6-M4 | Auto-updater works | App detects new version and offers download |
| P6-M5 | Documentation complete | README, contributing guide |
| P6-MT | Coverage targets met | E2E suite passes all critical paths; 80% Go / 75% frontend coverage |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P6-T01 | Performance profiling and optimization (frontend -- PixiJS, rendering) | L |
| P6-T02 | Performance profiling and optimization (backend -- SQLite queries, parsing) | L |
| P6-T03 | Cross-platform WebView testing (PixiJS on WebKit, WebView2, WebKitGTK) | M |
| P6-T04 | Implement auto-update checker (GitHub Releases or custom endpoint) | M |
| P6-T05 | Create macOS build + .dmg packaging | M |
| P6-T06 | Create Windows build + .exe installer (NSIS) | M |
| P6-T07 | Create Linux build + .AppImage | M |
| P6-T08 | Set up code signing (macOS notarization, Windows signing) | L |
| P6-T09 | End-to-end testing of critical paths (Playwright) | L |
| P6-T10 | Write README.md and contributing guide | S |

---

## 10. Dependency Graph

```
P1 (Desktop Foundation)
  │
  └──▶ P2 (Auth & Demo Pipeline)
         │
         ├──▶ P3 (Core 2D Viewer)
         │      │
         │      ├──▶ P4 (Faceit & Heatmaps) ──┐
         │      │                               │
         │      └──▶ P5 (Strategy & Lineups) ──┤
         │                                      │
         └──────────────────────────────────────┘
                                                │
                                                ▼
                                        P6 (Polish & Distribute)
```

### Cross-Phase Dependencies

| Dependency | Reason |
|------------|--------|
| P3 depends on P2 | Viewer needs parsed tick/event data in SQLite |
| P4 depends on P2 | Heatmaps need parsed game events |
| P4 depends on P3 | Heatmap canvas reuses PixiJS map layer |
| P5 depends on P3 | Strat board reuses map rendering + coordinate system |
| P5-T09 depends on P2-T06 | Grenade extraction extends the demo parser |
| P6 depends on P4, P5 | Polish and distribute all features |
| P2+ depends on P1-T09, P1-T10 | All TDD tasks require test infrastructure |

### Parallel Work Opportunities

Within P1, **P1-T09** and **P1-T10** (test infrastructure) can be done in parallel with **P1-T01 through P1-T08**.

Once P2 is complete, P3 can begin immediately. Within P3, once the basic viewer works (P3-M2):

- **P4-T01 to P4-T02**: Faceit API client (no viewer dependency)
- **P5-T01 to P5-T02**: Drawing canvas infrastructure (independent of viewer data flow)

---

*Cross-references: [PRD.md](PRD.md) for feature requirements, [ARCHITECTURE.md](ARCHITECTURE.md) for system design, [TASK_BREAKDOWN.md](TASK_BREAKDOWN.md) for granular tasks with acceptance criteria.*
