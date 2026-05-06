# Architecture — Component Diagrams

> **Siblings:** [overview](overview.md) · [structure](structure.md) · [data-flows](data-flows.md) · [wails-bindings](wails-bindings.md) · [database](database.md) · [crosscutting](crosscutting.md) · [testing](testing.md)

---

## Component Diagrams (C4 Level 3)

### Go Backend Components

```
┌─────────────────────────────────────────────────────┐
│                   App (Wails Bindings)                │
│                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │ Demo bindings │  │ Viewer       │  │ Heatmap   │  │
│  │               │  │ bindings     │  │ bindings  │  │
│  │ - ImportFile  │  │ - GetDemo    │  │ - GetData │  │
│  │ - ImportDir   │  │ - GetRounds  │  │ - Weapons │  │
│  │ - ListDemos   │  │ - GetTicks   │  │ - Players │  │
│  │ - DeleteDemo  │  │ - GetEvents  │  │ - Stats   │  │
│  │ - parseDemo   │  │ - Roster     │  └─────┬─────┘  │
│  └──────┬───────┘  │ - Scoreboard │        │        │
│         │          └──────┬───────┘        │        │
│  ┌──────▼─────────────────▼────────────────▼─────┐  │
│  │              Store (sqlc / SQLite)              │  │
│  │                                                │  │
│  │  - DemoQueries    - RoundQueries               │  │
│  │  - TickQueries    - EventQueries               │  │
│  │  - LineupQueries  - BoardQueries               │  │
│  │  - Heatmap custom queries (json_each based)    │  │
│  └────────────────────────────────────────────────┘  │
│                                                     │
│  ┌──────────────┐  ┌──────────────┐                 │
│  │ Demo Parser   │  │ Import       │                 │
│  │ (demoinfocs)  │  │ Service      │                 │
│  │               │  │              │                 │
│  │ - ticks       │  │ - validate   │                 │
│  │ - events      │  │ - decompress │                 │
│  │ - rounds      │  │ - persist    │                 │
│  └──────────────┘  └──────────────┘                 │
└─────────────────────────────────────────────────────┘
```

### React Frontend Components

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
│  │  router)   │  │            │  │ - QueryProvider       │ │
│  │            │  │ - viewer   │  │ - ThemeProvider       │ │
│  │ - Demos    │  │ - strat    │  │                       │ │
│  │ - Viewer   │  │ - ui       │  └───────────────────────┘ │
│  │ - Heatmaps │  │ - demo     │                           │
│  │ - Strats   │  │            │                           │
│  │ - Lineups  │  └─────┬─────┘                           │
│  │ - Settings │        │                                   │
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
| **Zustand stores** | Separate stores per domain: `viewerStore` (playback state, current tick), `stratStore` (board state), `uiStore` (sidebar, modals), `demoStore` (library state). |
| **TanStack Query** | Wraps Wails binding calls. Stale-while-revalidate for demo lists. Invalidation on import/delete. |
| **react-router-dom** | Client-side routing; replaces Next.js App Router. Outlet-based layout with sidebar navigation. |
