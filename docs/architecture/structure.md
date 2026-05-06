# Architecture — Application Structure

> **Siblings:** [overview](overview.md) · [components](components.md) · [data-flows](data-flows.md) · [wails-bindings](wails-bindings.md) · [database](database.md) · [crosscutting](crosscutting.md) · [testing](testing.md)

---

## Application Structure (C4 Level 2)

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
│  │  │  - Demo bindings     │──┼──┼─▶│                  │  │ │
│  │  │  - Viewer bindings   │  │  │  │  - Demo Library  │  │ │
│  │  │  - Heatmap bindings  │  │  │  │  - 2D Viewer     │  │ │
│  │  └──────────┬───────────┘  │  │  │  - Heatmaps      │  │ │
│  │             │              │  │  │  - Strat Board   │  │ │
│  │  ┌──────────▼───────────┐  │  │  │  - Lineups       │  │ │
│  │  │  SQLite (WAL mode)    │  │  │  │  - Settings      │  │ │
│  │  │  modernc.org/sqlite   │  │  │  └──────────────────┘  │ │
│  │  └──────────────────────┘  │  │                        │ │
│  │                            │  └────────────────────────┘ │
│  │  ┌──────────────────────┐  │                             │
│  │  │  Demo Parser          │  │                             │
│  │  │  (demoinfocs-golang)  │  │                             │
│  │  └──────────────────────┘  │                             │
│  └────────────────────────────┘                             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                       ┌──────────────┐
                       │ Local        │
                       │ Filesystem   │
                       │ (.dem, .db)  │
                       └──────────────┘
```

### Component Communication

| From | To | Mechanism | Notes |
|------|----|-----------|-------|
| React SPA | Go Backend | Wails bindings | Auto-generated TS functions from Go methods |
| Go Backend | SQLite | `modernc.org/sqlite` | sqlc-generated queries; WAL mode |
| Go Backend | Filesystem | `os` package | Read `.dem` files, manage app data dir |
| Demo Parser | SQLite | Transaction batches | 10K-row batched inserts for tick data |

The app has no network dependencies at runtime — everything is local.

---

## Project Directory Structure

```
oversite/
├── main.go                         # Wails entry point
├── app.go                          # App struct (Startup/Shutdown, Wails bindings)
├── types.go                        # Domain types exposed to the frontend
├── go.mod                          # Root Go module (github.com/ok2ju/oversite)
├── wails.json                      # Wails project config
├── internal/
│   ├── database/                   # SQLite connection, migration runner
│   ├── demo/                       # Parser, importer, ingest, stats
│   ├── logging/                    # Structured logging + rotation
│   ├── store/                      # sqlc generated code (SQLite)
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
├── e2e/                            # Playwright E2E tests
├── Makefile                        # Root dev commands
└── docs/                           # Obsidian vault: product, architecture, decisions, plans, knowledge
```

> **Note:** All desktop Go code lives at the root module level. There are no separate `backend/` or web-app subprojects.
