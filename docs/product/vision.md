# Product — Vision & Goals

> **Version:** 2.1 · **Siblings:** [personas](personas.md) · [features](features.md) · [user-stories](user-stories.md) · [non-functional](non-functional.md) · [data-models](data-models.md) · [wails-bindings](wails-bindings.md)

---

## Product Vision

**Oversite** is a desktop 2D demo viewer and analytics platform for Counter-Strike 2 (CS2). It transforms local `.dem` files into interactive playback, heatmaps, strategy boards, and stat dashboards — giving competitive players the tools to study their game on their own machine, with zero cloud infrastructure and no account required.

### Problem Statement

CS2 players lack a fast, unified, local-only tool to:

- Review demo playback in 2D (top-down) without launching CS2
- Aggregate statistics across multiple demos
- Plan strategies on a map canvas with drawing tools
- Catalog and browse grenade lineups extracted from actual gameplay

### Why Desktop

- **No upload latency**: Demos are already on disk; parsing starts instantly
- **No infrastructure cost**: No servers to host or maintain
- **Full hardware utilization**: Gamers have capable machines; leverage local CPU/GPU
- **Simpler architecture**: Single binary, single process, local database, no auth

### Product Goals

| # | Goal | Success Metric |
|---|------|---------------|
| G1 | Instant demo playback | < 10s from selecting a local `.dem` to first frame rendered |
| G2 | Cross-demo analytics | Heatmaps aggregating 10+ demos render in < 5s |
| G3 | Local strategy planning | Drawing tools responsive at 60 FPS on the map canvas |
| G4 | Grenade knowledge base | Users can browse, save, tag, and filter lineups |
| G5 | Per-player insights | Comprehensive per-player statistics across imported demos to help users improve |

## Technology Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Desktop Framework** | Wails v2 | Go backend + system WebView; single binary output |
| **Frontend** | Vite + React 19 | SPA with react-router-dom; embedded via `embed.FS` |
| **UI Components** | shadcn/ui + Tailwind CSS | Accessible, themeable component library |
| **2D Rendering** | PixiJS v8 | WebGL canvas for performant 2D playback |
| **State Management** | Zustand | Lightweight stores, selector-based subscriptions |
| **Data Fetching** | TanStack Query v5 | Caches Wails binding responses |
| **Backend** | Go 1.26+ | Business logic exposed as Wails bindings |
| **Demo Parser** | markus-wa/demoinfocs-golang v5 | Only mature Go-based CS2 demo parser |
| **Database** | SQLite (modernc.org/sqlite) | Pure Go, CGo-free; WAL mode; local file |
| **SQL Generation** | sqlc (SQLite dialect) | Type-safe Go code from SQL queries |
| **Auth** | None | Single-tenant local app |
| **Routing** | react-router-dom v6 | Client-side SPA routing |
