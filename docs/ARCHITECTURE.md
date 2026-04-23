# Oversite -- Architecture Documentation

> **Version:** 2.0
> **Last Updated:** 2026-04-12
> **Format:** arc42

---

## Table of Contents

1. [Introduction & Goals](#1-introduction--goals)
2. [System Context (C4 Level 1)](#2-system-context-c4-level-1)
3. [Application Structure (C4 Level 2)](#3-application-structure-c4-level-2)
4. [Component Diagrams (C4 Level 3)](#4-component-diagrams-c4-level-3)
5. [Data Flow Diagrams](#5-data-flow-diagrams)
6. [Wails Bindings Specification](#6-wails-bindings-specification)
7. [Database Schema](#7-database-schema)
8. [Local Storage Layout](#8-local-storage-layout)
9. [Cross-Cutting Concerns](#9-cross-cutting-concerns)
10. [Project Directory Structure](#10-project-directory-structure)
11. [Testing Architecture](#11-testing-architecture)

---

## 1. Introduction & Goals

### 1.1 Requirements Overview

Oversite is a desktop 2D demo viewer and analytics platform for CS2 Faceit players. It runs as a single native binary using Wails (Go backend + system WebView frontend).

| Priority | Quality Goal | Motivation |
|----------|-------------|------------|
| 1 | **Performance** | 60 FPS canvas rendering; < 10s demo parse from local disk; < 50ms tick query |
| 2 | **Simplicity** | Single binary, single process, no external services except Faceit API |
| 3 | **Developer Experience** | Monorepo with hot reload, type-safe SQL, Wails dev mode |
| 4 | **Cross-Platform** | macOS, Windows, Linux from a single codebase |

### 1.2 Stakeholders

| Role | Concern |
|------|---------|
| Solo developer | Productive monorepo DX; manageable complexity |
| End users (Faceit players) | Fast, reliable demo review on their desktop |
| Future contributors | Clear architecture boundaries; documented bindings |

---

## 2. System Context (C4 Level 1)

```
┌─────────────────────────────────────────────────────────┐
│                    External Systems                      │
│                                                         │
│  ┌──────────────┐                   ┌──────────────┐   │
│  │  Faceit API   │                   │ Local         │   │
│  │  (OAuth +     │                   │ Filesystem    │   │
│  │   Data API)   │                   │ (.dem files)  │   │
│  └──────┬───────┘                   └──────┬───────┘   │
│         │                                  │           │
└─────────┼──────────────────────────────────┼───────────┘
          │                                  │
          ▼                                  ▼
┌─────────────────────────────────────────────────────────┐
│                                                         │
│              O V E R S I T E  (Desktop)                  │
│                                                         │
│    Native desktop app for CS2 demo review, analytics,   │
│    strategy planning, and Faceit stats tracking.        │
│                                                         │
│    Single binary: Go backend + WebView frontend         │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### External System Interfaces

| System | Protocol | Purpose |
|--------|----------|---------|
| **Faceit OAuth** | HTTPS (OAuth 2.0 + PKCE) | User authentication via loopback redirect |
| **Faceit Data API** | HTTPS REST | Player stats, match history, ELO data |
| **Local Filesystem** | OS file I/O | Read `.dem` files, SQLite database, app data |

---

## 3. Application Structure (C4 Level 2)

```
┌─────────────────────────────────────────────────────────────┐
│                    Wails Runtime (Single Process)             │
│                                                             │
│  ┌────────────────────────────┐  ┌────────────────────────┐ │
│  │      Go Backend             │  │    System WebView       │ │
│  │                            │  │                        │ │
│  │  ┌──────────────────────┐  │  │  ┌──────────────────┐  │ │
│  │  │  App (Wails Bindings) │  │  │  │  React SPA       │  │ │
│  │  │                      │◀─┼──┼──│  (Vite + PixiJS)  │  │ │
│  │  │  - DemoService       │──┼──┼─▶│                  │  │ │
│  │  │  - FaceitService     │  │  │  │  - Viewer Page   │  │ │
│  │  │  - AuthService       │  │  │  │  - Heatmap Page  │  │ │
│  │  │  - StoreService      │  │  │  │  - Strat Board   │  │ │
│  │  │  - HeatmapService    │  │  │  │  - Dashboard     │  │ │
│  │  │  - StratService      │  │  │  │  - Lineup Page   │  │ │
│  │  │  - LineupService     │  │  │  │  - Demo Library  │  │ │
│  │  └──────────┬───────────┘  │  │  └──────────────────┘  │ │
│  │             │              │  │                        │ │
│  │  ┌──────────▼───────────┐  │  └────────────────────────┘ │
│  │  │  SQLite (WAL mode)    │  │                             │
│  │  │  modernc.org/sqlite   │  │                             │
│  │  └──────────────────────┘  │                             │
│  │                            │                             │
│  │  ┌──────────────────────┐  │                             │
│  │  │  Demo Parser          │  │                             │
│  │  │  (demoinfocs-golang)  │  │                             │
│  │  └──────────────────────┘  │                             │
│  │                            │                             │
│  └────────────────────────────┘                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
         │                              │
         ▼                              ▼
    ┌──────────┐                 ┌──────────────┐
    │ Faceit   │                 │ Local        │
    │ API      │                 │ Filesystem   │
    │ (HTTPS)  │                 │ (.dem, .db)  │
    └──────────┘                 └──────────────┘
```

### Component Communication

| From | To | Mechanism | Notes |
|------|----|-----------|-------|
| React SPA | Go Backend | Wails bindings | Auto-generated TS functions from Go methods |
| Go Backend | SQLite | `modernc.org/sqlite` | sqlc-generated queries; WAL mode |
| Go Backend | Filesystem | `os` package | Read `.dem` files, manage app data dir |
| Go Backend | Faceit API | `net/http` | REST calls for profile, matches, ELO |
| Go Backend | OS Keychain | `zalando/go-keyring` | Store/retrieve OAuth tokens |
| Demo Parser | SQLite | Transaction batches | 10K-row batched inserts for tick data |

---

## 4. Component Diagrams (C4 Level 3)

### 4.1 Go Backend Components

```
┌─────────────────────────────────────────────────────┐
│                   App (Wails Bindings)                │
│                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │ DemoService   │  │ FaceitService│  │AuthService│  │
│  │               │  │              │  │           │  │
│  │ - ImportDemo  │  │ - GetProfile │  │ - Login   │  │
│  │ - ImportDir   │  │ - GetElo     │  │ - Logout  │  │
│  │ - ListDemos   │  │ - GetMatches │  │ - Refresh │  │
│  │ - GetTicks    │  │ - Sync       │  │           │  │
│  │ - GetEvents   │  │ - ImportDemo │  └───────────┘  │
│  └──────┬───────┘  └──────┬───────┘                 │
│         │                 │                          │
│  ┌──────▼─────────────────▼──────────────────────┐   │
│  │              StoreService (sqlc/SQLite)         │   │
│  │                                                │   │
│  │  - DemoQueries    - RoundQueries               │   │
│  │  - TickQueries    - EventQueries               │   │
│  │  - FaceitQueries  - LineupQueries              │   │
│  │  - BoardQueries   - UserQueries                │   │
│  └────────────────────────────────────────────────┘   │
│                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │ HeatmapSvc   │  │ StratService │  │LineupSvc  │  │
│  │               │  │              │  │           │  │
│  │ - GetData     │  │ - CRUD       │  │ - CRUD    │  │
│  │ - Aggregate   │  │ - Export/    │  │ - Favorite│  │
│  │               │  │   Import JSON│  │ - Extract │  │
│  └──────────────┘  └──────────────┘  └───────────┘  │
│                                                     │
│  ┌──────────────┐  ┌──────────────┐                 │
│  │ Demo Parser   │  │ Faceit Client│                 │
│  │ (demoinfocs)  │  │ (HTTP)       │                 │
│  └──────────────┘  └──────────────┘                 │
│                                                     │
│  ┌──────────────┐                                   │
│  │ Keyring       │                                   │
│  │ (go-keyring)  │                                   │
│  └──────────────┘                                   │
└─────────────────────────────────────────────────────┘
```

### 4.2 React Frontend Components

```
┌──────────────────────────────────────────────────────────┐
│                  React SPA (Vite)                          │
│                                                          │
│  ┌─────────────────────────────────────────────────────┐ │
│  │                    App Shell                         │ │
│  │  Sidebar  │  Header  │  Content Area (Outlet)       │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌───────────┐  ┌───────────┐  ┌───────────────────────┐ │
│  │  Pages     │  │  Stores    │  │  Providers            │ │
│  │ (react-    │  │ (Zustand)  │  │                       │ │
│  │  router)   │  │            │  │ - AuthProvider        │ │
│  │            │  │ - viewer   │  │ - QueryProvider       │ │
│  │ - Viewer   │  │ - strat    │  │ - ThemeProvider       │ │
│  │ - Heatmap  │  │ - ui       │  │ - RouterProvider      │ │
│  │ - Strats   │  │ - faceit   │  │                       │ │
│  │ - Dashboard│  │ - demo     │  └───────────────────────┘ │
│  │ - Lineups  │  │            │                           │
│  │ - DemoLib  │  └─────┬─────┘                           │
│  └─────┬─────┘        │                                  │
│        │              │                                   │
│  ┌─────▼──────────────▼──────────────────────────────┐   │
│  │                 Canvas Layer                        │   │
│  │                                                    │   │
│  │  ┌──────────────┐  ┌──────────────┐               │   │
│  │  │  PixiJS App   │  │  Strat Canvas │               │   │
│  │  │  (Viewer)     │  │  (Drawing)    │               │   │
│  │  │               │  │               │               │   │
│  │  │ - MapLayer    │  │ - DrawLayer   │               │   │
│  │  │ - PlayerLayer │  │ - TokenLayer  │               │   │
│  │  │ - EventLayer  │  │ - ToolLayer   │               │   │
│  │  │ - UILayer     │  │               │               │   │
│  │  └──────────────┘  └──────────────┘               │   │
│  └────────────────────────────────────────────────────┘   │
│                                                          │
│  ┌────────────────────────────────────────────────────┐   │
│  │                  UI Components (shadcn/ui)          │   │
│  │  Button │ Dialog │ Tabs │ Select │ Slider │ ...    │   │
│  └────────────────────────────────────────────────────┘   │
│                                                          │
│  ┌────────────────────────────────────────────────────┐   │
│  │           Wails JS Bindings (auto-generated)        │   │
│  │  wailsjs/go/main/App.ts                            │   │
│  └────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

### Key Frontend Patterns

| Pattern | Implementation |
|---------|---------------|
| **PixiJS outside React** | PixiJS Application instantiated in a `useEffect`; React renders a container `<div>`, PixiJS manages its own render loop. Zustand store bridges React UI controls to PixiJS state. (See [ADR-0001](adr/0001-pixijs-outside-react.md)) |
| **Zustand stores** | Separate stores per domain: `viewerStore` (playback state, current tick), `stratStore` (board state), `uiStore` (sidebar, modals), `faceitStore` (profile, matches), `demoStore` (library state). |
| **TanStack Query** | Wraps Wails binding calls. Stale-while-revalidate for demo lists, Faceit data. Invalidation on import/delete. |
| **react-router-dom** | Client-side routing; replaces Next.js App Router. Outlet-based layout with sidebar navigation. |

#### Dashboard composition

The `/dashboard` route is intentionally lean: it renders only the Faceit **ProfileHero** (avatar, level, ELO, progress to next tier) and **RecentMatches** (the match history list) in a single column. Deeper stats (per-map, per-weapon, rolling form) live on per-demo analytics surfaces — primarily **Match Details** (`/matches/:demoId`, reached by clicking a match row) and the 2D Viewer. Earlier PerformanceGrid / RecentForm / MapPerformance / Weapons widgets were removed because they duplicated data shown elsewhere or were placeholder-only.

---

## 5. Data Flow Diagrams

### 5.1 Demo Import & Parse

```
User                    React SPA          Go Backend            SQLite           Filesystem
 │                        │                    │                    │                │
 │  Drag-drop .dem file   │                    │                    │                │
 │───────────────────────▶│                    │                    │                │
 │                        │  ImportDemo(path)  │                    │                │
 │                        │───────────────────▶│                    │                │
 │                        │                    │  Validate file     │                │
 │                        │                    │  (magic bytes,     │                │
 │                        │                    │   size check)      │                │
 │                        │                    │                    │                │
 │                        │                    │  Read .dem file    │                │
 │                        │                    │────────────────────────────────────▶│
 │                        │                    │          file data                  │
 │                        │                    │◀────────────────────────────────────│
 │                        │                    │                    │                │
 │                        │                    │  INSERT demo       │                │
 │                        │                    │  (status=parsing)  │                │
 │                        │                    │───────────────────▶│                │
 │                        │                    │                    │                │
 │                        │                    │  Parse demo        │                │
 │                        │                    │  (demoinfocs)      │                │
 │                        │                    │                    │                │
 │                        │                    │  BEGIN transaction │                │
 │                        │                    │  Batch INSERT      │                │
 │                        │                    │  tick_data (10K    │                │
 │                        │                    │  rows per batch)   │                │
 │                        │                    │───────────────────▶│                │
 │                        │                    │                    │                │
 │                        │                    │  INSERT events,    │                │
 │                        │                    │  rounds,           │                │
 │                        │                    │  player_rounds     │                │
 │                        │                    │───────────────────▶│                │
 │                        │                    │  COMMIT            │                │
 │                        │                    │                    │                │
 │                        │                    │  UPDATE demo       │                │
 │                        │                    │  (status=ready)    │                │
 │                        │                    │───────────────────▶│                │
 │                        │                    │                    │                │
 │                        │  Demo (ready)      │                    │                │
 │                        │◀───────────────────│                    │                │
 │  Show in library       │                    │                    │                │
 │◀───────────────────────│                    │                    │                │
```

### 5.2 Viewer Playback

```
User                    React SPA          Zustand Store       PixiJS App         Go Backend         SQLite
 │                        │                    │                  │                  │                │
 │  Open demo in viewer   │                    │                  │                  │                │
 │───────────────────────▶│                    │                  │                  │                │
 │                        │  GetRounds(demoId) │                  │                  │                │
 │                        │──────────────────────────────────────────────────────────▶│                │
 │                        │                    │                  │                  │  SELECT         │
 │                        │                    │                  │                  │────────────────▶│
 │                        │  rounds            │                  │                  │◀────────────────│
 │                        │◀──────────────────────────────────────────────────────────│                │
 │                        │                    │                  │                  │                │
 │                        │  setState(rounds)  │                  │                  │                │
 │                        │───────────────────▶│                  │                  │                │
 │                        │                    │  subscribe()     │                  │                │
 │                        │                    │─────────────────▶│                  │                │
 │                        │                    │                  │                  │                │
 │  Press Play            │                    │                  │                  │                │
 │───────────────────────▶│                    │                  │                  │                │
 │                        │  setPlaying(true)  │                  │                  │                │
 │                        │───────────────────▶│                  │                  │                │
 │                        │                    │  tick advance    │                  │                │
 │                        │                    │─────────────────▶│                  │                │
 │                        │                    │                  │                  │                │
 │                        │                    │  Need more ticks │                  │                │
 │                        │                    │  GetTicks(range) │                  │                │
 │                        │                    │──────────────────────────────────────▶│                │
 │                        │                    │                  │                  │  SELECT         │
 │                        │                    │                  │                  │────────────────▶│
 │                        │                    │  tick data       │                  │◀────────────────│
 │                        │                    │◀──────────────────────────────────────│                │
 │                        │                    │  buffer ticks    │                  │                │
 │                        │                    │─────────────────▶│                  │                │
 │                        │                    │                  │  Render frame    │                │
 │  60 FPS playback       │                    │                  │  (60 FPS)        │                │
 │◀───────────────────────────────────────────────────────────────│                  │                │
```

### 5.3 Faceit OAuth Loopback Flow

```
User                    React SPA          Go Backend            System Browser    Faceit API        OS Keychain
 │                        │                    │                    │                │                │
 │  Click "Login"         │                    │                    │                │                │
 │───────────────────────▶│                    │                    │                │                │
 │                        │  StartLogin()      │                    │                │                │
 │                        │───────────────────▶│                    │                │                │
 │                        │                    │  Start temp HTTP   │                │                │
 │                        │                    │  listener on       │                │                │
 │                        │                    │  127.0.0.1:{port}  │                │                │
 │                        │                    │                    │                │                │
 │                        │                    │  Open auth URL     │                │                │
 │                        │                    │───────────────────▶│                │                │
 │                        │                    │                    │                │                │
 │  Authenticate in       │                    │                    │  GET /authorize │                │
 │  browser               │                    │                    │───────────────▶│                │
 │───────────────────────▶│                    │                    │                │                │
 │                        │                    │                    │  302 → localhost│                │
 │                        │                    │                    │◀───────────────│                │
 │                        │                    │                    │                │                │
 │                        │                    │  Receive callback  │                │                │
 │                        │                    │◀───────────────────│                │                │
 │                        │                    │  (auth code)       │                │                │
 │                        │                    │                    │                │                │
 │                        │                    │  Exchange code     │                │                │
 │                        │                    │  for tokens        │                │                │
 │                        │                    │──────────────────────────────────────▶│                │
 │                        │                    │  access + refresh  │                │                │
 │                        │                    │◀──────────────────────────────────────│                │
 │                        │                    │                    │                │                │
 │                        │                    │  Store refresh     │                │                │
 │                        │                    │  token in keychain │                │                │
 │                        │                    │──────────────────────────────────────────────────────▶│
 │                        │                    │                    │                │                │
 │                        │                    │  Fetch profile     │                │                │
 │                        │                    │──────────────────────────────────────▶│                │
 │                        │                    │  profile data      │                │                │
 │                        │                    │◀──────────────────────────────────────│                │
 │                        │                    │                    │                │                │
 │                        │                    │  INSERT/UPDATE user│                │                │
 │                        │                    │  in SQLite         │                │                │
 │                        │                    │                    │                │                │
 │                        │  LoginResult       │                    │                │                │
 │                        │◀───────────────────│                    │                │                │
 │  Show dashboard        │                    │                    │                │                │
 │◀───────────────────────│                    │                    │                │                │
```

### 5.4 Faceit Match Sync

```
User                    React SPA          Go Backend            Faceit API        SQLite
 │                        │                    │                    │                │
 │  Dashboard loads       │                    │                    │                │
 │───────────────────────▶│                    │                    │                │
 │                        │  SyncMatches()     │                    │                │
 │                        │───────────────────▶│                    │                │
 │                        │                    │  GET /matches      │                │
 │                        │                    │───────────────────▶│                │
 │                        │                    │  match list        │                │
 │                        │                    │◀───────────────────│                │
 │                        │                    │                    │                │
 │                        │                    │  For each new match│                │
 │                        │                    │  GET /match/{id}   │                │
 │                        │                    │───────────────────▶│                │
 │                        │                    │  match detail      │                │
 │                        │                    │◀───────────────────│                │
 │                        │                    │                    │                │
 │                        │                    │  UPSERT matches    │                │
 │                        │                    │  into SQLite       │                │
 │                        │                    │──────────────────────────────────────▶│
 │                        │                    │                    │                │
 │                        │  SyncResult        │                    │                │
 │                        │◀───────────────────│                    │                │
 │  Show updated matches  │                    │                    │                │
 │◀───────────────────────│                    │                    │                │
```

### 5.5 Demo Download from Match Row

Triggered when a user clicks **Import demo** on a dashboard match row that has a Faceit-hosted `demo_url` but no local import. The download runs in-process via `DownloadService`, and parsing is auto-triggered on success (reusing the flow from 5.1).

```
User        React SPA           Go Backend (App)      DownloadService    Faceit CDN    SQLite
 │             │                      │                      │               │           │
 │ Click       │                      │                      │               │           │
 │ Import ───▶ │ ImportMatchDemo()    │                      │               │           │
 │             │────────────────────▶ │ DownloadAndImport()  │               │           │
 │             │                      │────────────────────▶ │ GET demo.dem  │           │
 │             │                      │                      │─────────────▶ │           │
 │             │                      │                      │  bytes        │           │
 │             │                      │ Emit                 │◀───────────── │           │
 │             │                      │ faceit:demo:         │               │           │
 │             │◀─ download progress  │ download:progress    │               │           │
 │  Show pill  │                      │                      │ INSERT demo   │           │
 │             │                      │                      │────────────────────────── ▶│
 │             │                      │                      │               │           │
 │             │                      │  (auto-trigger parse — see 5.1)     │           │
 │             │                      │                      │               │           │
 │             │ Invalidate           │                      │               │           │
 │             │ ['faceit-matches']   │                      │               │           │
 │             │ ['demos'] queries    │                      │               │           │
 │◀ Row flips  │                      │                      │               │           │
 │  to "Demo   │                      │                      │               │           │
 │   ready"    │                      │                      │               │           │
```

### 5.6 Dashboard / Demos → Match Details → Viewer

Clicks from the dashboard match list and the Demo Library converge on Match Details (`/matches/:demoId`). The 2D Viewer is reached from Match Details via the **Play demo** button.

```
User        React SPA (list)       Match Details          2D Viewer        demoStore
 │             │                         │                    │                │
 │  Click row  │                         │                    │                │
 │  (has_demo) │                         │                    │                │
 │───────────▶ │                         │                    │                │
 │             │  If importProgress      │                    │                │
 │             │  targets this demo &    │                    │                │
 │             │  stage ∈ {importing,    │                    │                │
 │             │           parsing}:     │                    │                │
 │             │    show "Parsing…"      │                    │                │
 │             │    indicator, wait on   │                    │                │
 │             │    demo:parse:progress  │                    │                │
 │             │    stage === complete   │                    │                │
 │             │                         │                    │                │
 │             │  navigate(/matches/:id) │                    │                │
 │             │────────────────────────▶│                    │                │
 │             │                         │ GetDemoByID +      │                │
 │             │                         │ rounds + scoreboard│                │
 │             │                         │                    │                │
 │◀ scoreboard │                         │                    │                │
 │  + timeline │                         │                    │                │
 │             │                         │                    │                │
 │  Click      │                         │                    │                │
 │  Play demo  │                         │                    │                │
 │───────────────────────────────────────▶│                    │                │
 │             │                         │ navigate(/demos/:id)                │
 │             │                         │────────────────────▶│                │
 │             │                         │                    │  Load ticks,   │
 │             │                         │                    │  render PixiJS │
 │◀ Viewer     │                         │                    │                │
```

---

## 6. Wails Bindings Specification

Wails bindings replace the REST API from the web version. Go struct methods decorated with Wails annotations are automatically available as TypeScript functions in the frontend.

### 6.1 Binding Architecture

```
Go struct method                    Auto-generated TS function
─────────────────                   ──────────────────────────
func (a *App) ImportDemo(           import { ImportDemo } from
  path string,                        '../../wailsjs/go/main/App';
) (*Demo, error)
                                    const demo = await ImportDemo(path);
```

Wails generates the TypeScript bindings at build time from Go method signatures. The generated files live in `frontend/wailsjs/`.

### 6.2 Error Handling Convention

All binding methods return `(result, error)` in Go. In TypeScript, errors become rejected promises:

```typescript
try {
  const demo = await ImportDemo(path);
} catch (err) {
  // err contains the Go error message
  toast.error(`Import failed: ${err}`);
}
```

### 6.3 Event System

For long-running operations (demo parsing), Wails runtime events provide progress updates:

```go
// Go: emit progress events
runtime.EventsEmit(a.ctx, "demo:parse:progress", DemoProgress{
    DemoID:  id,
    Percent: 42,
    Phase:   "parsing_ticks",
})
```

```typescript
// TypeScript: subscribe to progress
import { EventsOn } from '../../wailsjs/runtime/runtime';

EventsOn('demo:parse:progress', (progress: DemoProgress) => {
  setParseProgress(progress.Percent);
});
```

### 6.4 Full Binding Reference

See [PRD.md Section 10](PRD.md#10-wails-bindings-overview) for the complete binding method table.

---

## 7. Database Schema

SQLite database using `modernc.org/sqlite` (pure Go). WAL mode enabled for concurrent reads during writes. This section contains the canonical DDL. For business-level field descriptions, see [PRD.md Section 9](PRD.md#9-data-models).

### 7.1 Schema DDL

```sql
-- Enable WAL mode (run once on DB creation)
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

-- ─────────────────────────────────────────────
-- Users
-- ─────────────────────────────────────────────
CREATE TABLE users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    faceit_id       TEXT    NOT NULL UNIQUE,
    nickname        TEXT    NOT NULL,
    avatar_url      TEXT    NOT NULL DEFAULT '',
    faceit_elo      INTEGER NOT NULL DEFAULT 0,
    faceit_level    INTEGER NOT NULL DEFAULT 0,
    country         TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ─────────────────────────────────────────────
-- Demos
-- ─────────────────────────────────────────────
CREATE TABLE demos (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    faceit_match_id TEXT,
    map_name        TEXT    NOT NULL,
    file_path       TEXT    NOT NULL,
    file_size       INTEGER NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'imported',
    total_ticks     INTEGER NOT NULL DEFAULT 0,
    tick_rate       REAL    NOT NULL DEFAULT 0,
    duration_secs   INTEGER NOT NULL DEFAULT 0,
    match_date      TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_demos_user_id ON demos(user_id);
CREATE INDEX idx_demos_status ON demos(status);

-- ─────────────────────────────────────────────
-- Rounds
-- ─────────────────────────────────────────────
CREATE TABLE rounds (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id         INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_number    INTEGER NOT NULL,
    start_tick      INTEGER NOT NULL,
    end_tick        INTEGER NOT NULL,
    winner_side     TEXT    NOT NULL,
    win_reason      TEXT    NOT NULL,
    ct_score        INTEGER NOT NULL DEFAULT 0,
    t_score         INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_rounds_demo_id ON rounds(demo_id);

-- ─────────────────────────────────────────────
-- Player Rounds
-- ─────────────────────────────────────────────
CREATE TABLE player_rounds (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    round_id        INTEGER NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    steam_id        TEXT    NOT NULL,
    player_name     TEXT    NOT NULL,
    team_side       TEXT    NOT NULL,
    kills           INTEGER NOT NULL DEFAULT 0,
    deaths          INTEGER NOT NULL DEFAULT 0,
    assists         INTEGER NOT NULL DEFAULT 0,
    damage          INTEGER NOT NULL DEFAULT 0,
    headshot_kills  INTEGER NOT NULL DEFAULT 0,
    first_kill      INTEGER NOT NULL DEFAULT 0,
    first_death     INTEGER NOT NULL DEFAULT 0,
    clutch_kills    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_player_rounds_round_id ON player_rounds(round_id);
CREATE INDEX idx_player_rounds_steam_id ON player_rounds(steam_id);

-- ─────────────────────────────────────────────
-- Tick Data (largest table; ~1.28M rows per demo)
-- ─────────────────────────────────────────────
CREATE TABLE tick_data (
    demo_id         INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    tick            INTEGER NOT NULL,
    steam_id        TEXT    NOT NULL,
    x               REAL    NOT NULL,
    y               REAL    NOT NULL,
    z               REAL    NOT NULL,
    yaw             REAL    NOT NULL,
    health          INTEGER NOT NULL,
    armor           INTEGER NOT NULL,
    is_alive        INTEGER NOT NULL DEFAULT 1,
    weapon          TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (demo_id, tick, steam_id)
);

-- Primary composite index handles range scans: WHERE demo_id = ? AND tick BETWEEN ? AND ?
-- No additional index needed; the PK serves as the clustered index.

-- ─────────────────────────────────────────────
-- Game Events
-- ─────────────────────────────────────────────
CREATE TABLE game_events (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id             INTEGER NOT NULL REFERENCES demos(id) ON DELETE CASCADE,
    round_id            INTEGER NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    tick                INTEGER NOT NULL,
    event_type          TEXT    NOT NULL,
    attacker_steam_id   TEXT,
    victim_steam_id     TEXT,
    weapon              TEXT,
    x                   REAL    NOT NULL,
    y                   REAL    NOT NULL,
    z                   REAL    NOT NULL,
    extra_data          TEXT    NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_game_events_demo_id ON game_events(demo_id);
CREATE INDEX idx_game_events_round_id ON game_events(round_id);
CREATE INDEX idx_game_events_type ON game_events(demo_id, event_type);

-- ─────────────────────────────────────────────
-- Strategy Boards
-- ─────────────────────────────────────────────
CREATE TABLE strategy_boards (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    title           TEXT    NOT NULL,
    map_name        TEXT    NOT NULL,
    board_state     TEXT    NOT NULL DEFAULT '{}',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- ─────────────────────────────────────────────
-- Grenade Lineups
-- ─────────────────────────────────────────────
CREATE TABLE grenade_lineups (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    demo_id         INTEGER REFERENCES demos(id) ON DELETE SET NULL,
    tick            INTEGER NOT NULL DEFAULT 0,
    map_name        TEXT    NOT NULL,
    grenade_type    TEXT    NOT NULL,
    throw_x         REAL    NOT NULL,
    throw_y         REAL    NOT NULL,
    throw_z         REAL    NOT NULL,
    throw_yaw       REAL    NOT NULL,
    throw_pitch     REAL    NOT NULL,
    land_x          REAL    NOT NULL,
    land_y          REAL    NOT NULL,
    land_z          REAL    NOT NULL,
    title           TEXT    NOT NULL DEFAULT '',
    description     TEXT    NOT NULL DEFAULT '',
    tags            TEXT    NOT NULL DEFAULT '[]',
    is_favorite     INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_grenade_lineups_map ON grenade_lineups(map_name);
CREATE INDEX idx_grenade_lineups_type ON grenade_lineups(map_name, grenade_type);

-- ─────────────────────────────────────────────
-- Faceit Matches
-- ─────────────────────────────────────────────
CREATE TABLE faceit_matches (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    faceit_match_id TEXT    NOT NULL,
    map_name        TEXT    NOT NULL,
    score_team      INTEGER NOT NULL,
    score_opponent  INTEGER NOT NULL,
    result          TEXT    NOT NULL,
    elo_before      INTEGER NOT NULL DEFAULT 0,
    elo_after       INTEGER NOT NULL DEFAULT 0,
    kills           INTEGER NOT NULL DEFAULT 0,
    deaths          INTEGER NOT NULL DEFAULT 0,
    assists         INTEGER NOT NULL DEFAULT 0,
    demo_url        TEXT    NOT NULL DEFAULT '',
    demo_id         INTEGER REFERENCES demos(id) ON DELETE SET NULL,
    played_at       TEXT    NOT NULL,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, faceit_match_id)
);

CREATE INDEX idx_faceit_matches_user_id ON faceit_matches(user_id);
CREATE INDEX idx_faceit_matches_played_at ON faceit_matches(user_id, played_at);
```

### 7.2 Key Schema Differences from Web Version

| Aspect | Web (PostgreSQL + TimescaleDB) | Desktop (SQLite) |
|--------|-------------------------------|------------------|
| Primary keys | UUID (gen_random_uuid()) | INTEGER AUTOINCREMENT |
| Timestamps | TIMESTAMPTZ | TEXT (ISO 8601) |
| JSON fields | JSONB | TEXT (JSON string) |
| Arrays | TEXT[] | TEXT (JSON array string) |
| Binary data | BYTEA | BLOB |
| Booleans | BOOLEAN | INTEGER (0/1) |
| Tick data | Hypertable with chunk compression | Regular table with composite PK |
| Sessions | Redis | Not needed (desktop, single-user) |
| Object storage | MinIO (S3) | Local filesystem |

---

## 8. Local Storage Layout

### 8.1 OS-Specific Paths

| Platform | App Data Directory |
|----------|--------------------|
| macOS | `~/Library/Application Support/oversite/` |
| Windows | `%APPDATA%\oversite\` |
| Linux | `~/.local/share/oversite/` |

### 8.2 Directory Structure

```
{app_data_dir}/
├── oversite.db              # SQLite database (WAL mode)
├── oversite.db-wal          # WAL file (auto-managed)
├── oversite.db-shm          # Shared memory file (auto-managed)
├── logs/
│   └── oversite.log         # Application log (rotated)
└── config.json              # User preferences (theme, watch folder, etc.)
```

### 8.3 Demo File Storage

Demo `.dem` files are **not** copied into the app data directory. The `demos.file_path` column stores the absolute path to the original file on the user's filesystem. This avoids doubling disk usage for large demo files.

If a user deletes the original `.dem` file, the parsed data in SQLite remains available. The demo's status can be updated to `source_missing` if the file is no longer found.

### 8.4 Credential Storage

OAuth tokens are stored in the OS keychain, **not** in the app data directory:

| Platform | Keychain API | Service Name |
|----------|-------------|-------------|
| macOS | Keychain Services | `oversite-faceit-auth` |
| Windows | Credential Manager | `oversite-faceit-auth` |
| Linux | Secret Service (GNOME Keyring / KWallet) | `oversite-faceit-auth` |

---

## 9. Cross-Cutting Concerns

### 9.1 Error Handling

| Layer | Strategy |
|-------|----------|
| Go bindings | Return `error` as second return value; Wails converts to rejected Promise |
| Frontend | `try/catch` on binding calls; toast notifications for user-facing errors |
| Demo parser | Structured errors with context (e.g., `ParseError{Phase: "ticks", Tick: 42000, Err: ...}`) |
| SQLite | Wrap in transaction; rollback on error; return descriptive error to caller |

### 9.2 Logging

Implemented in `internal/logging/` (see [ADR-0013](adr/0013-logging.md)). Files live under `{AppDataDir}/logs/`:

| File | When | Contents |
|------|------|----------|
| `errors.txt` | Always | `slog` WARN+ records from all Go packages, plus a bridge that captures any remaining `log.Printf` calls |
| `network.txt` | Dev builds only (`runtime.Environment(ctx).BuildType == "dev"`) | Full HTTP request/response dumps from the Faceit client, demo download client, and OAuth token exchange |

- Both files rotate at 5MB with 3 backups via `lumberjack.v2` (plain `.txt`, no gzip).
- `logging.Init(dir)` runs once from `main.go` before `wails.Run`; `logging.Close()` runs from `App.Shutdown`.
- The dev network transport is wired into HTTP clients in `App.Startup`; the `OVERSITE_DEBUG_HTTP` env flag used by the old `debug_transport` is retired.
- Frontend: `console.error` for binding failures; no separate log file (developers use the Wails DevTools).

### 9.3 Configuration

User preferences stored in `config.json` in the app data directory:

```json
{
  "theme": "dark",
  "watchFolder": "/path/to/cs2/demos",
  "autoFetchFaceit": true,
  "viewerDefaults": {
    "playbackSpeed": 1.0,
    "showHealthBars": true,
    "showMinimap": true
  }
}
```

Loaded at startup by Go backend; exposed to frontend via `GetConfig`/`SetConfig` bindings.

### 9.4 SQLite Data Integrity & Recovery

The SQLite database is the sole store for all parsed demo data. Since re-parsing demos is possible but expensive (< 10s per demo), data integrity matters.

**WAL checkpoint**: SQLite WAL mode is used for concurrent reads. The WAL file is checkpointed automatically by SQLite. On graceful shutdown (`app.Shutdown()`), force a WAL checkpoint via `PRAGMA wal_checkpoint(TRUNCATE)` to ensure the database file is self-contained.

**Pre-migration backup**: Before running `golang-migrate` up migrations, copy the database file to `oversite.db.bak`. If migration fails, the backup allows manual recovery. The backup is overwritten on each migration run (only the most recent is kept).

**Corruption detection**: On startup, run `PRAGMA integrity_check` (fast on small databases). If corruption is detected:
1. Log the error with full details
2. Show a modal to the user: "Database may be corrupted. You can re-import your demos to rebuild."
3. Offer to reset the database (delete and recreate with fresh migrations)
4. Demo `.dem` source files are stored by reference and are not affected

**Transaction discipline**: All multi-row inserts (tick data, events, rounds) are wrapped in explicit transactions. A parse failure mid-way rolls back the entire transaction — no partial demo data in the database.

**Recovery path**: Since demos are stored as external `.dem` files (not copied into the database), the worst-case recovery is: delete `oversite.db`, restart the app (migrations recreate schema), re-import demos. Faceit match data can be re-synced from the API.

### 9.5 Coordinate Calibration

Each CS2 map has calibration data mapping game world-space to radar image pixel-space. Stored in the frontend as TypeScript constants:

```typescript
// src/lib/maps/calibration.ts
export const MAP_CALIBRATION = {
  de_dust2:  { originX: -2476, originY: 3239, scale: 4.4 },
  de_mirage: { originX: -3230, originY: 1713, scale: 5.0 },
  // ...
};

// Formula: pixelX = (worldX - originX) / scale
//          pixelY = (originY - worldY) / scale
```

### 9.6 Auto-Update

- On startup, check for new version via HTTPS to a releases endpoint (GitHub Releases API or custom)
- Show non-intrusive notification if update available
- User-initiated download and install (no silent auto-update)
- Version check skippable in settings

---

## 10. Project Directory Structure

```
oversite/
├── main.go                         # Wails entry point
├── app.go                          # App struct (Startup/Shutdown)
├── go.mod                          # Root Go module (github.com/ok2ju/oversite)
├── wails.json                      # Wails project config
├── internal/
│   ├── auth/                       # OAuth loopback, keyring
│   ├── config/                     # Env/file-based config
│   ├── database/                   # SQLite connection, migrations
│   ├── demo/                       # Parser, import service
│   ├── faceit/                     # API client, sync
│   ├── heatmap/                    # KDE generation
│   ├── lineup/                     # Grenade lineup service
│   ├── model/                      # Domain types
│   ├── store/                      # sqlc generated code (SQLite)
│   ├── strat/                      # Strategy board service
│   └── testutil/                   # Shared test helpers
├── migrations/                     # SQL migration files (SQLite, embedded)
├── queries/                        # sqlc SQL files
├── testdata/                       # Golden files for parser tests
├── frontend/
│   ├── src/
│   │   ├── routes/                 # react-router-dom pages
│   │   ├── components/             # UI, viewer, strat, layout
│   │   ├── hooks/                  # Custom React hooks
│   │   ├── lib/                    # PixiJS, maps, utils
│   │   ├── stores/                 # Zustand stores
│   │   ├── test/                   # Test setup and helpers
│   │   ├── types/                  # TypeScript types
│   │   └── utils/
│   ├── wailsjs/                    # Auto-generated Wails bindings
│   ├── public/maps/                # Radar images
│   ├── index.html                  # Vite entry point
│   └── vite.config.ts
├── backend/                        # Web version (legacy, not used for desktop)
├── e2e/                            # Playwright E2E tests
├── Makefile                        # Root dev commands
└── docs/                           # PRD, Architecture, Plans, ADRs
```

> **Note:** All desktop Go code lives at the root module level. The `backend/` directory contains the web version's codebase and is **not** modified for desktop development.

---

## 11. Testing Architecture

### 11.1 Test Strategy by Layer

| Layer | Tool | Strategy |
|-------|------|----------|
| Go services | `go test` | Classical TDD; interfaces enable mocking |
| Go bindings | `go test` + httptest-style | Test service methods directly |
| Go demo parser | Golden-file TDD + spike | Spike validates library, then golden tests |
| sqlc queries | `go test` + temp SQLite | `:memory:` or temp file SQLite for each test |
| Zustand stores | Vitest | Classical TDD; pure state + actions |
| React components | Vitest + RTL | Render + interaction tests |
| Wails binding hooks | Vitest + mock bindings | Mock auto-generated binding functions |
| PixiJS rendering | Test-alongside | TDD the logic; screenshot-test the visuals |
| E2E flows | Playwright | Test-alongside; written after features work |

### 11.2 Test Infrastructure

| Component | Purpose |
|-----------|---------|
| Temp SQLite databases | In-memory (`:memory:`) or temp file per test; run migrations; clean between tests |
| MSW (Mock Service Worker) | Mock Faceit API responses for frontend tests |
| Vitest + React Testing Library | Frontend component and hook testing |
| Playwright | E2E tests against a running Wails dev instance |
| Golden files | Known-good parser output for regression testing |

### 11.3 Key Differences from Web Test Architecture

| Aspect | Web Version | Desktop Version |
|--------|-------------|-----------------|
| Database tests | testcontainers (PostgreSQL) | Temp SQLite (`:memory:` or temp file) |
| API tests | httptest against chi router | Direct service method calls |
| WebSocket tests | WebSocket test client | Not needed (no WebSocket) |
| Yjs tests | In-memory Yjs docs | Not needed (no Yjs) |
| Auth tests | Mock Redis session store | Mock keyring interface |

---

*Cross-references: [PRD.md](PRD.md) for feature requirements, [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for delivery phases, [TASK_BREAKDOWN.md](TASK_BREAKDOWN.md) for granular tasks.*
