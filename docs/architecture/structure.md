# Architecture — Application Structure

> **Siblings:** [overview](overview.md) · [components](components.md) · [data-flows](data-flows.md) · [wails-bindings](wails-bindings.md) · [database](database.md) · [crosscutting](crosscutting.md) · [testing](testing.md)

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
