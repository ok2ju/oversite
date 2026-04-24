# Architecture — Component Diagrams

> **Siblings:** [overview](overview.md) · [structure](structure.md) · [data-flows](data-flows.md) · [wails-bindings](wails-bindings.md) · [database](database.md) · [crosscutting](crosscutting.md) · [testing](testing.md)

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
| **PixiJS outside React** | PixiJS Application instantiated in a `useEffect`; React renders a container `<div>`, PixiJS manages its own render loop. Zustand store bridges React UI controls to PixiJS state. (See [ADR-0001](../decisions/0001-pixijs-outside-react.md)) |
| **Zustand stores** | Separate stores per domain: `viewerStore` (playback state, current tick), `stratStore` (board state), `uiStore` (sidebar, modals), `faceitStore` (profile, matches), `demoStore` (library state). |
| **TanStack Query** | Wraps Wails binding calls. Stale-while-revalidate for demo lists, Faceit data. Invalidation on import/delete. |
| **react-router-dom** | Client-side routing; replaces Next.js App Router. Outlet-based layout with sidebar navigation. |

#### Dashboard composition

The `/dashboard` route is intentionally lean: it renders only the Faceit **ProfileHero** (avatar, level, ELO, progress to next tier) and **RecentMatches** (the match history list) in a single column. Deeper stats (per-map, per-weapon, rolling form) live on per-demo analytics surfaces — primarily **Match Details** (`/matches/:demoId`, reached by clicking a match row) and the 2D Viewer. Earlier PerformanceGrid / RecentForm / MapPerformance / Weapons widgets were removed because they duplicated data shown elsewhere or were placeholder-only.
