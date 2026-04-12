# Oversite -- Implementation Plan

> **Version:** 1.0
> **Last Updated:** 2026-03-31

---

## Table of Contents

1. [Git Worktree Strategy](#1-git-worktree-strategy)
2. [TDD Methodology](#2-tdd-methodology)
3. [Phase Overview](#3-phase-overview)
4. [Phase 1: Foundation](#4-phase-1-foundation)
5. [Phase 2: Auth & Demo Pipeline](#5-phase-2-auth--demo-pipeline)
6. [Phase 3: Core 2D Viewer](#6-phase-3-core-2d-viewer)
7. [Phase 4: Faceit & Heatmaps](#7-phase-4-faceit--heatmaps)
8. [Phase 5: Strategy Board & Lineups](#8-phase-5-strategy-board--lineups)
9. [Phase 6: Polish & Deploy](#9-phase-6-polish--deploy)
10. [Dependency Graph](#10-dependency-graph)

---

## 1. Git Worktree Strategy

The repository uses a **bare clone + worktree** model. The bare repo at `oversite/` contains no working tree; all development happens in worktrees.

### Branch Naming Convention

| Type | Pattern | Example |
|------|---------|---------|
| Main | `main` | `main` |
| Phase | `phase/{n}-{name}` | `phase/1-foundation` |
| Feature | `feat/{phase}-{short-desc}` | `feat/p2-demo-parser` |
| Bug fix | `fix/{short-desc}` | `fix/tick-data-offset` |
| Docs | `docs/{short-desc}` | `docs/api-spec` |

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
- No long-lived develop branch; `main` is always deployable after Phase 1

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

1. **RED**: Write a test that defines the expected behavior. Run it -- it must fail. This confirms the test actually tests something.
2. **GREEN**: Write the minimum code to make the test pass. No more.
3. **REFACTOR**: Clean up the implementation and the test. All tests must remain green.
4. **COMMIT**: Commit after each green-to-refactor cycle.

### 2.2 TDD Applicability by Layer

Not every layer uses classical TDD. The approach is adapted to the nature of the code:

| Layer | TDD Approach | Rationale |
|-------|-------------|-----------|
| Go services | Classical TDD | Pure business logic; interfaces enable mocking |
| Go HTTP handlers | TDD with httptest | Request/response tests written first |
| Go demo parser | Golden-file TDD + spike | Spike validates external library, then golden tests |
| sqlc queries | TDD with testcontainers | Write expected-result tests against real DB first |
| Zustand stores | Classical TDD | Pure state + actions, no I/O |
| React components | TDD with React Testing Library | Render + interaction tests written first |
| TanStack Query hooks | TDD with MSW | Hook behavior tests with mock server first |
| PixiJS rendering | Test-alongside | TDD the logic (transforms, interpolation); screenshot-test the visuals |
| Yjs collaboration | TDD with in-memory docs | Convergence tests written first |
| E2E flows | Test-alongside | Written after features work, not strict Red-Green-Refactor |

### 2.3 Test Infrastructure Requirements per Phase

| Phase | Test Infrastructure Added | CI Gate |
|-------|--------------------------|---------|
| **P1** | testcontainers-go base, Vitest + RTL + MSW, Playwright config | `go test` + `pnpm test` pass |
| **P2** | Demo fixture files, golden file framework, auth mock helpers | Golden tests pass; integration tests pass |
| **P3** | PixiJS screenshot test pipeline, coordinate fixture data | Screenshot comparison stable |
| **P4** | Faceit API mock handlers (MSW), KDE test fixtures | Mock API tests pass |
| **P5** | Yjs in-memory test helpers, WebSocket test client | Convergence tests pass |
| **P6** | Full E2E test suite, coverage reporting | Coverage targets met; all E2E pass |

### 2.4 Testing Milestones Convention

Every phase includes a **testing milestone** (`Px-MT`) that must be met before the phase is considered complete. These milestones verify that the TDD process was followed and that test coverage is adequate for the phase's deliverables.

---

## 3. Phase Overview

```
Phase 1          Phase 2             Phase 3           Phase 4          Phase 5            Phase 6
Foundation       Auth & Demo         Core 2D           Faceit &         Strategy Board     Polish &
                 Pipeline            Viewer             Heatmaps         & Lineups          Deploy
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Monorepo         Faceit OAuth        PixiJS setup       Faceit API       WebSocket hub      Performance
Docker Compose   Upload endpoint     Map rendering      client           Yjs integration    Security
Go scaffold      Demo parser         Player rendering   Auto-fetch       Drawing tools      Responsive
Next.js scaffold Redis Streams       Event rendering    Heatmap KDE      Grenade catalog    Docs
DB migrations    MinIO storage       Playback controls  Stats charts     Lineup CRUD        Prod Docker
CI pipeline      Job processing      Scoreboard         Dashboard UI     Sharing            Deployment
```

| Phase | Name | Key Milestone | Dependencies |
|-------|------|---------------|-------------|
| **P1** | Foundation | `docker compose up` runs all services; CI passes | None |
| **P2** | Auth & Demo Pipeline | Upload `.dem` → parsed data in DB | P1 |
| **P3** | Core 2D Viewer | Play back a demo in the browser at 60 FPS | P2 |
| **P4** | Faceit & Heatmaps | Faceit stats dashboard + KDE heatmaps render | P2, P3 |
| **P5** | Strategy Board & Lineups | Real-time collaborative drawing + lineup catalog | P3 |
| **P6** | Polish & Deploy | Production-ready deployment | P4, P5 |

---

## 4. Phase 1: Foundation

**Goal**: Monorepo scaffolding, Docker Compose environment, database schema, CI pipeline. After this phase, `docker compose up` brings up all services, and the CI pipeline runs lint + test + build.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P1-M1 | Monorepo structure created | Directory layout matches ARCHITECTURE.md spec |
| P1-M2 | Docker Compose runs | All 8 containers start and pass health checks |
| P1-M3 | Database migrated | All tables + hypertable created via migration tool |
| P1-M4 | Go scaffold works | `GET /healthz` returns 200 |
| P1-M5 | Next.js scaffold works | Home page renders with shadcn/ui components |
| P1-M6 | CI pipeline passes | Lint, test, build all green |
| P1-MT | Test infrastructure works | `go test ./...` runs; `pnpm test` runs; CI runs both; testcontainers works in CI |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P1-T01 | Initialize monorepo structure | S |
| P1-T02 | Set up Docker Compose (all 8 services) | M |
| P1-T03 | Set up nginx reverse proxy config | S |
| P1-T04 | Scaffold Go backend (cmd, internal packages, chi router) | M |
| P1-T05 | Create database migrations (full schema) | M |
| P1-T06 | Configure sqlc and generate Go code | M |
| P1-T07 | Scaffold Next.js frontend (App Router, shadcn/ui, Tailwind) | M |
| P1-T08 | Set up Zustand stores (skeleton) | S |
| P1-T09 | Set up CI pipeline (lint, test, build) | M |
| P1-T10 | Create root Makefile with dev commands | S |
| P1-T11 | Set up Go test infrastructure (testcontainers, test helpers, mock interfaces, CI integration stage) | M |
| P1-T12 | Set up frontend test infrastructure (Vitest config, RTL, MSW, Playwright, test helpers) | M |

---

## 5. Phase 2: Auth & Demo Pipeline

**Goal**: Users can log in with Faceit, upload `.dem` files, and have them parsed into queryable data. This is the critical foundation for all features.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P2-M1 | Faceit OAuth works | User can log in, session created, profile displayed |
| P2-M2 | Demo upload works | File uploaded to MinIO, demo record in DB |
| P2-M3 | Demo parsing works | Parse job processes, tick data + events in DB |
| P2-M4 | Demo library works | UI lists user's demos with status |
| P2-MT | Tests pass | Parser golden file tests pass; auth integration tests pass; upload handler tests pass |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P2-T01 | Implement Faceit OAuth 2.0 + PKCE flow | M |
| P2-T02 | Implement Redis session management | M |
| P2-T03 | Create auth middleware (Go) | S |
| P2-T04 | Create AuthProvider + login page (Next.js) | M |
| P2-T05 | Set up MinIO buckets and S3 client | S |
| P2-T06 | Implement demo upload endpoint (validation, MinIO storage) | M |
| P2-T07 | Set up Redis Streams job queue (producer + consumer) | M |
| P2-T08 | **Implement demo parser core** (demoinfocs-golang integration) | **XL** |
| P2-T09 | Parse ticks → batch insert into TimescaleDB hypertable | L |
| P2-T10 | Parse events → insert game_events (kills, grenades, bombs) | L |
| P2-T11 | Parse rounds → insert rounds + player_rounds | M |
| P2-T12 | Build demo library UI (list, status, delete) | M |

### Critical Path Note

**P2-T08 (Demo Parser Core)** is the highest-risk, highest-complexity task in the entire project. The `demoinfocs-golang` library requires careful integration:

- Registering event handlers for all needed game events
- Extracting player state at configurable tick intervals (every Nth tick for storage efficiency)
- Handling CS2 demo format edge cases (warmup rounds, overtime, bot players)
- Memory management for large demos (100 MB+ files)

Budget extra time here. Consider a spike/prototype before the full implementation.

---

## 6. Phase 3: Core 2D Viewer

**Goal**: Users can watch a parsed demo in a 2D top-down view with full playback controls. This is the flagship feature.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P3-M1 | Map renders | Correct radar image displayed, scaled to canvas |
| P3-M2 | Players render | All 10 players shown with correct positions each tick |
| P3-M3 | Events render | Kills, grenades, bomb events visible on map |
| P3-M4 | Playback works | Play/pause, speed, seek, round select all functional |
| P3-M5 | Scoreboard works | Accurate stats overlay for current round |
| P3-MT | Tests pass | Playback engine unit tests pass; coordinate transform tests pass; Playwright screenshots stable |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P3-T01 | Set up PixiJS Application + canvas container | M |
| P3-T02 | Implement map layer (radar images, coordinate calibration) | M |
| P3-T03 | Implement tick data fetching (API + client-side buffer) | L |
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

**Goal**: Faceit stats dashboard with ELO history + interactive KDE heatmaps for kills and positioning.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P4-M1 | Faceit profile + ELO chart | Dashboard shows profile, ELO line chart |
| P4-M2 | Match history | Paginated match list with links to demos |
| P4-M3 | Auto-fetch | Background job syncs Faceit matches on login |
| P4-M4 | Kill heatmap | KDE overlay renders correctly for a demo |
| P4-M5 | Aggregated heatmap | Multi-demo heatmap with filters |
| P4-MT | Tests pass | Faceit client mock tests pass; KDE algorithm unit tests pass; heatmap endpoint integration tests pass |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P4-T01 | Implement Faceit API client (Go HTTP client) | M |
| P4-T02 | Implement Faceit sync worker job | M |
| P4-T03 | Build Faceit dashboard page (profile, ELO chart) | M |
| P4-T04 | Build match history list (pagination, filters) | M |
| P4-T05 | Implement demo auto-import from Faceit matches | L |
| P4-T06 | Implement heatmap data endpoint (aggregation query) | M |
| P4-T07 | Implement client-side KDE rendering on PixiJS canvas | L |
| P4-T08 | Build heatmap filter controls (map, side, weapon, player) | M |
| P4-T09 | Build per-demo stats view | M |
| P4-T10 | Build cross-demo trend charts | M |

---

## 8. Phase 5: Strategy Board & Lineups

**Goal**: Real-time collaborative strategy drawing with Yjs + WebSocket, and a grenade lineup catalog auto-populated from demos.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P5-M1 | WS server running | WebSocket connections established and authenticated |
| P5-M2 | Yjs sync works | Two browsers see each other's drawings in < 200ms |
| P5-M3 | Drawing tools complete | Freehand, shapes, arrows, text, player tokens, eraser |
| P5-M4 | Persistence works | Board state survives all clients disconnecting |
| P5-M5 | Lineup catalog works | Auto-extracted lineups browsable by map/type |
| P5-M6 | Personal collection | Users can save, tag, favorite lineups |
| P5-MT | Tests pass | WebSocket hub unit tests pass; Yjs convergence tests pass; drawing tool logic tests pass |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P5-T01 | Implement WebSocket server (gorilla/websocket, room management) | L |
| P5-T02 | Implement Yjs relay protocol in Go | L |
| P5-T03 | Set up Yjs client (y-websocket provider, awareness) | M |
| P5-T04 | Implement drawing canvas (PixiJS or Canvas 2D) | L |
| P5-T05 | Implement drawing tools (freehand, line, arrow, rect, circle, text) | L |
| P5-T06 | Implement strategy primitives (player tokens, grenade markers) | M |
| P5-T07 | Implement undo/redo (Yjs UndoManager) | M |
| P5-T08 | Implement board persistence (Yjs state ↔ PostgreSQL) | M |
| P5-T09 | Build board list + create/delete UI | S |
| P5-T10 | Implement sharing (token generation, share modes) | M |
| P5-T11 | Implement PNG export | S |
| P5-T12 | Add grenade extraction to demo parser | M |
| P5-T13 | Build lineup catalog page (browse, filter, search) | M |
| P5-T14 | Implement lineup CRUD + favorites | M |

---

## 9. Phase 6: Polish & Deploy

**Goal**: Performance optimization, security hardening, responsive design, documentation, and production Docker setup.

### Milestones

| ID | Milestone | Done When |
|----|-----------|-----------|
| P6-M1 | Performance targets met | 60 FPS viewer, < 200ms API p95, < 30s parse |
| P6-M2 | Security audit passed | OWASP top 10 mitigated, rate limiting in place |
| P6-M3 | Responsive UI | Usable on tablet (1024px+), functional on mobile |
| P6-M4 | Documentation complete | README, API docs, contributing guide |
| P6-M5 | Production deployment | `docker compose -f docker-compose.prod.yml up` works |
| P6-MT | Coverage targets met | E2E suite passes all critical paths; 80% Go / 75% frontend coverage; performance benchmarks pass |

### Tasks

| Task | Description | Complexity |
|------|-------------|-----------|
| P6-T01 | Performance profiling and optimization (frontend) | L |
| P6-T02 | Performance profiling and optimization (backend) | L |
| P6-T03 | Security hardening (CSRF, rate limiting, input validation) | M |
| P6-T04 | Add TLS configuration to nginx | S |
| P6-T05 | Implement responsive layouts | M |
| P6-T06 | Write README.md | S |
| P6-T07 | Write API documentation | M |
| P6-T08 | Create production Docker Compose (env vars, secrets, resource limits) | M |
| P6-T09 | Add error tracking setup (Sentry or equivalent) | S |
| P6-T10 | End-to-end testing of critical paths | L |

---

## 10. Dependency Graph

```
P1 (Foundation)
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
                                        P6 (Polish & Deploy)
```

### Cross-Phase Dependencies

| Dependency | Reason |
|------------|--------|
| P3 depends on P2 | Viewer needs parsed tick/event data |
| P4 depends on P2 | Heatmaps need parsed game events |
| P4 depends on P3 | Heatmap canvas reuses PixiJS map layer |
| P5 depends on P3 | Strat board reuses map rendering + coordinate system |
| P5-T12 depends on P2-T08 | Grenade extraction extends the demo parser |
| P6 depends on P4, P5 | Polish and deploy all features |
| P2+ depends on P1-T11, P1-T12 | All TDD tasks require test infrastructure to write tests first |

### Parallel Work Opportunities

Within P1, **P1-T11** and **P1-T12** (test infrastructure) can be done in parallel with **P1-T04 through P1-T10**.

Once P2 is complete, P3 can begin immediately. Within P3, once the basic viewer works (P3-M2), the following can begin in parallel:

- **P4-T01 to P4-T04**: Faceit API client and dashboard (no viewer dependency)
- **P5-T01 to P5-T03**: WebSocket server + Yjs relay (independent infrastructure)

---

*Cross-references: [PRD.md](PRD.md) for feature requirements, [ARCHITECTURE.md](ARCHITECTURE.md) for system design, [TASK_BREAKDOWN.md](TASK_BREAKDOWN.md) for granular tasks with acceptance criteria.*
