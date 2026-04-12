# Oversite -- Product Requirements Document

> **Version:** 2.0
> **Last Updated:** 2026-04-12
> **Status:** Draft

---

## Table of Contents

1. [Product Vision](#1-product-vision)
2. [Target Audience & Personas](#2-target-audience--personas)
3. [Technology Stack](#3-technology-stack)
4. [Information Architecture](#4-information-architecture)
5. [Feature Specifications](#5-feature-specifications)
6. [User Stories](#6-user-stories)
7. [Non-Functional Requirements](#7-non-functional-requirements)
8. [Data Models](#8-data-models)
9. [Wails Bindings Overview](#9-wails-bindings-overview)

---

## 1. Product Vision

**Oversite** is a desktop 2D demo viewer and analytics platform for Counter-Strike 2 (CS2) Faceit players. It transforms local `.dem` files into interactive playback, heatmaps, strategy boards, and stat dashboards -- giving competitive players the tools to study their game on their own machine, with zero cloud infrastructure.

### Problem Statement

CS2 players on Faceit lack a fast, unified tool to:

- Review demo playback in 2D (top-down) without launching CS2
- Aggregate statistics across multiple demos and Faceit matches
- Plan strategies on a map canvas with drawing tools
- Catalog and browse grenade lineups extracted from actual gameplay

### Why Desktop

- **No upload latency**: Demos are already on disk; parsing starts instantly
- **No infrastructure cost**: No servers to host or maintain
- **Full hardware utilization**: Gamers have capable machines; leverage local CPU/GPU
- **Simpler architecture**: Single binary, single process, local database

### Product Goals

| # | Goal | Success Metric |
|---|------|---------------|
| G1 | Instant demo playback | < 10s from selecting a local `.dem` to first frame rendered |
| G2 | Cross-demo analytics | Heatmaps aggregating 10+ demos render in < 5s |
| G3 | Local strategy planning | Drawing tools responsive at 60 FPS on the map canvas |
| G4 | Grenade knowledge base | Users can browse, save, tag, and filter lineups |
| G5 | Faceit integration | Auto-fetch recent matches, display ELO history |

---

## 2. Target Audience & Personas

### Persona 1: Solo Grinder ("Kai")

| Attribute | Detail |
|-----------|--------|
| Role | Faceit Level 7-10 player, solo queue |
| Goal | Identify personal mistakes, track ELO trends |
| Pain Point | Rewatching full demos in-game is slow; no easy cross-demo stats |
| Key Features | 2D Viewer, Heatmaps, Faceit Stats Dashboard |
| Technical Level | Comfortable installing desktop apps; not a developer |

### Persona 2: Team IGL ("Sofia")

| Attribute | Detail |
|-----------|--------|
| Role | In-Game Leader of a 5-stack team |
| Goal | Prepare strats, review team demos, share grenade setups |
| Pain Point | Coordinating strat prep across Discord, screenshots, and in-game practice |
| Key Features | Strategy Board, Grenade Lineups, 2D Viewer |
| Technical Level | Moderate; uses Faceit, Discord, basic tools |

### Persona 3: Casual Analyst ("Marcus")

| Attribute | Detail |
|-----------|--------|
| Role | Content creator / coach who reviews others' demos |
| Goal | Quickly scrub through demos, generate visual insights for content |
| Pain Point | Existing tools require CS2 running or clunky web uploads |
| Key Features | 2D Viewer (speed controls), Heatmaps, PNG export |
| Technical Level | High; comfortable with desktop tools and data |

---

## 3. Technology Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Desktop Framework** | Wails v2 | Go backend + system WebView; single binary output |
| **Frontend** | Vite + React 18 | SPA with react-router-dom; embedded via `embed.FS` |
| **UI Components** | shadcn/ui + Tailwind CSS | Accessible, themeable component library |
| **2D Rendering** | PixiJS v8 | WebGL canvas for performant 2D playback |
| **State Management** | Zustand | Lightweight stores, selector-based subscriptions |
| **Data Fetching** | TanStack Query v5 | Caches Wails binding responses; background refetch for Faceit data |
| **Backend** | Go 1.22+ | Business logic exposed as Wails bindings |
| **Demo Parser** | markus-wa/demoinfocs-golang v5 | Only mature Go-based CS2 demo parser |
| **Database** | SQLite (modernc.org/sqlite) | Pure Go, CGo-free; WAL mode; local file |
| **SQL Generation** | sqlc (SQLite dialect) | Type-safe Go code from SQL queries |
| **Auth** | Faceit OAuth 2.0 + PKCE | Loopback redirect flow (RFC 8252) |
| **Token Storage** | zalando/go-keyring | OS keychain (Keychain, Credential Manager, Secret Service) |
| **Routing** | react-router-dom v6 | Client-side SPA routing |

---

## 4. Information Architecture

### React Router Structure

```
src/
├── routes/
│   ├── root.tsx                    # App shell (sidebar + header + Outlet)
│   ├── login.tsx                   # Faceit OAuth login trigger
│   ├── dashboard.tsx               # Faceit stats overview
│   ├── demos/
│   │   ├── index.tsx               # Demo library (list/grid, drag-drop import)
│   │   └── $demoId/
│   │       ├── viewer.tsx          # 2D Viewer (PixiJS canvas)
│   │       └── heatmap.tsx         # Heatmap view for this demo
│   ├── heatmaps.tsx                # Cross-demo aggregated heatmaps
│   ├── strats/
│   │   ├── index.tsx               # Strategy board list
│   │   └── $stratId.tsx            # Strategy board canvas
│   ├── lineups/
│   │   ├── index.tsx               # Grenade lineup library
│   │   └── $lineupId.tsx           # Lineup detail + 2D preview
│   └── settings.tsx                # User preferences
```

### Navigation Hierarchy

```
Sidebar:
  ├── Dashboard          → /dashboard
  ├── Demos              → /demos
  ├── Heatmaps           → /heatmaps
  ├── Strategy Board     → /strats
  ├── Grenade Lineups    → /lineups
  └── Settings           → /settings
```

### Window & Menu Structure

```
Oversite
├── File
│   ├── Open Demo...        (Ctrl/Cmd+O)
│   ├── Import Folder...
│   └── Quit                (Ctrl/Cmd+Q)
├── View
│   ├── Toggle Sidebar
│   └── Full Screen         (F11)
└── Help
    ├── Check for Updates
    └── About Oversite
```

---

## 5. Feature Specifications

### F1: 2D Demo Viewer (Core)

The flagship feature. Renders a top-down 2D view of CS2 gameplay from parsed `.dem` files using PixiJS on an HTML5 Canvas.

#### F1.1 Map Rendering

- Display the correct radar image for the map (de_dust2, de_mirage, de_inferno, etc.)
- Scale coordinates from demo world-space to canvas pixel-space using map-specific calibration data
- Support all Active Duty maps in the current CS2 map pool

#### F1.2 Player Rendering

- Render each player as a colored circle with:
  - Team color (CT = blue, T = orange/yellow)
  - Player name label
  - View-angle indicator (cone or line showing where they're looking)
  - Health bar (optional toggle)
- Highlight the currently selected player
- Dim/fade dead players; show death marker (X) at kill location

#### F1.3 Event Rendering

| Event | Visual |
|-------|--------|
| Kill | Kill-feed entry + death X on map + line from killer to victim |
| Grenade (HE) | Expanding red circle at detonation point |
| Smoke | Gray filled circle with fade-in/fade-out matching smoke duration |
| Flashbang | Yellow flash circle, briefly |
| Molotov/Incendiary | Orange fill area matching fire spread |
| Bomb plant | Flashing icon at plant position |
| Bomb defuse | Progress indicator at bomb position |

#### F1.4 Playback Controls

- **Play / Pause** toggle
- **Playback speed**: 0.25x, 0.5x, 1x, 2x, 4x
- **Timeline scrubber**: Seek to any tick; displays round boundaries
- **Round selector**: Jump directly to any round
- **Tick counter**: Show current tick / total ticks
- **Keyboard shortcuts**: Space (play/pause), Left/Right (skip 5s), Up/Down (speed)

#### F1.5 Scoreboard Overlay

- Toggle-able scoreboard showing per-round and match-total stats
- Columns: Player, K, D, A, ADR, HS%, KAST, Rating
- Highlight the round being viewed

#### F1.6 Mini-map & Zoom

- Click-and-drag pan on the map
- Scroll-to-zoom (min 0.5x, max 4x)
- Mini-map in corner showing full map with viewport indicator
- Reset-view button

---

### F2: Heatmaps & Analytics

#### F2.1 Kill Heatmaps

- Kernel Density Estimation (KDE) rendered as color gradient overlay on map image
- Filter by: map, side (CT/T), player, weapon category, round type (eco/force/full buy)
- Single-demo and cross-demo aggregation modes

#### F2.2 Movement Heatmaps

- Show player position frequency as heat overlay
- Useful for identifying common rotations and positioning tendencies
- Filter by: player, side, round half

#### F2.3 Per-Player Statistics

- **Per-demo stats**: K/D/A, ADR, HS%, KAST, Rating 2.0 approximation
- **Cross-demo trends**: Line charts of key stats over time
- **Weapon breakdown**: Kills by weapon, accuracy estimates

#### F2.4 Aggregated Analytics

- Compare stats across multiple demos (last N matches)
- Map-specific performance breakdown
- Side-specific (CT vs T) performance

---

### F3: Strategy Board

#### F3.1 Canvas & Drawing Tools

- Full-screen canvas with the selected map as background
- Drawing tools: freehand, line, arrow, rectangle, circle, text label
- Color picker (preset team colors + custom)
- Eraser and undo/redo (Ctrl+Z / Ctrl+Shift+Z)
- Layer management: background (map), strategy layer, annotation layer

#### F3.2 Strategy Primitives

- Player tokens (draggable, labeled CT1-CT5 / T1-T5)
- Grenade trajectory lines (with arc indicator)
- Smoke/molotov/flash markers that can be placed on map
- Timing markers (numbered waypoints for execute order)

#### F3.3 Local Persistence

- Strategy board state saved to SQLite as JSON
- Autosave on every change (debounced)
- Full undo/redo history within a session

#### F3.4 Export & Management

- Save strategy boards to user's local library
- Export as PNG image
- Duplicate/fork existing boards
- Import/export boards as JSON files (for sharing via Discord, etc.)

---

### F4: Faceit Stats Dashboard

#### F4.1 Profile Overview

- Display Faceit avatar, nickname, level, ELO, country
- Current win streak / loss streak
- Membership info (Faceit Premium/Free)

#### F4.2 ELO History

- Line chart of ELO over time (last 30, 90, 180 days, all time)
- Annotate significant gains/losses
- Compare with average ELO for current level

#### F4.3 Match History

- Paginated list of recent Faceit matches
- Each entry: map, score, K/D/A, ELO change, date
- Click to open demo (if available locally) in 2D Viewer
- Filter by map, result (W/L), date range

#### F4.4 Performance Trends

- Rolling averages for K/D ratio, win rate, ADR
- Map-specific stats breakdown
- Best/worst maps identification

#### F4.5 Auto-Fetch

- On login, automatically fetch the user's recent Faceit match history
- In-process sync (no background worker -- runs in the Go backend directly)
- Optionally auto-download demos from Faceit match rooms

---

### F5: Grenade Lineups Library

#### F5.1 Auto-Extraction from Demos

- During demo parsing, detect grenade throws (smoke, flash, HE, molotov)
- Extract: thrower position, aim angle, grenade type, landing position
- Link to the specific tick in the demo for playback context

#### F5.2 Lineup Catalog

- Browse lineups by: map, grenade type, site (A/B/Mid), side
- Each lineup entry: 2D preview (throw position + landing on map), description, tags
- Search and filter functionality

#### F5.3 Personal Collection

- Users can save lineups to their personal library
- Add custom notes and tags
- Mark favorites for quick access

---

### F6: Local Demo Management

#### F6.1 File Import

- Drag-and-drop `.dem` files onto the app window
- File picker dialog (Ctrl/Cmd+O)
- Import entire folders (recursive scan for `.dem` files)

#### F6.2 Demo Library

- List view of all imported demos with metadata (map, date, players, size, status)
- Grid view option with map thumbnails
- Sort by date, map, size
- Search by player name, map name

#### F6.3 Auto-Scan (Optional)

- Configure a "watch folder" that the app scans for new `.dem` files on startup
- Default to CS2's demo download directory if detectable

#### F6.4 Storage Management

- Show total database size and demo count
- Delete demos (removes parsed data from SQLite; optionally removes source `.dem` file)
- Re-parse a demo (useful after parser updates)

---

## 6. User Stories

### Installation & Auth

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-01 | As a player, I want to install Oversite by downloading a single file | Installer/binary available for macOS, Windows, Linux; installs in < 30 seconds |
| US-02 | As a player, I want to log in with my Faceit account | Loopback OAuth flow opens browser; tokens stored in OS keychain; profile displayed |
| US-03 | As a new user, I want to see a quick onboarding tour | First-launch modal with 3-4 slides; dismissible; doesn't show again |

### Demo Management

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-04 | As a player, I want to drag-and-drop `.dem` files to import them | Drop zone accepts `.dem` files; parsing starts immediately; progress shown |
| US-05 | As a player, I want to import an entire folder of demos | Folder picker scans recursively; skips non-`.dem` files; batch progress indicator |
| US-06 | As a player, I want to see a list of my imported demos | Library shows demos sorted by date; displays map, date, players, parse status |
| US-07 | As a player, I want to delete a demo I no longer need | Confirm dialog; removes parsed data from SQLite; optionally deletes `.dem` file |
| US-08 | As a player, I want demos from Faceit matches auto-downloaded | After Faceit sync, app offers to download demos; progress shown in library |

### 2D Viewer

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-09 | As a player, I want to watch a demo in 2D top-down view | PixiJS canvas renders map + players + events; plays at real-time speed |
| US-10 | As a player, I want to control playback speed | Speed selector works (0.25x-4x); playback visually matches selected speed |
| US-11 | As a player, I want to scrub to any point in the demo | Timeline slider seeks to correct tick; canvas updates immediately |
| US-12 | As a player, I want to jump to a specific round | Round selector lists all rounds; clicking jumps to round start tick |
| US-13 | As a player, I want to see kill events on the map | Kill lines drawn from killer to victim; kill-feed updates; death X appears |
| US-14 | As a player, I want to see grenade effects on the map | Smokes, flashes, HEs, molotovs render with appropriate visual effects |
| US-15 | As a player, I want to zoom and pan the map | Scroll-to-zoom works; click-drag pans; mini-map shows viewport position |
| US-16 | As a player, I want to see the scoreboard for the current round | Toggle-able overlay shows accurate per-player stats |

### Heatmaps & Analytics

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-17 | As a player, I want to see a kill heatmap for a demo | KDE heatmap overlays on map image; color gradient indicates density |
| US-18 | As a player, I want to filter heatmaps by side, weapon, or player | Filters update heatmap in real-time; UI shows active filters |
| US-19 | As a player, I want to see aggregated heatmaps across multiple demos | Select demos to aggregate; combined heatmap renders correctly |
| US-20 | As a player, I want to see my per-demo statistics | Stats page shows K/D/A, ADR, HS%, KAST, Rating |
| US-21 | As a player, I want to see stat trends over time | Line charts render with correct data points; time range selectable |

### Strategy Board

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-22 | As an IGL, I want to draw strategies on a map | Drawing tools (freehand, line, arrow, shapes) work on map canvas |
| US-23 | As an IGL, I want to place player tokens on the map | CT/T tokens draggable; labeled; snap to reasonable positions |
| US-24 | As a user, I want to export a strategy board as PNG | PNG export captures the full board state at current zoom |
| US-25 | As a user, I want to share a board via JSON export | Export produces a JSON file; import on another machine restores the board |

### Faceit Dashboard

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-26 | As a player, I want to see my Faceit profile and ELO | Dashboard shows current ELO, level, avatar; data matches Faceit |
| US-27 | As a player, I want to see my ELO history as a chart | Line chart with selectable time ranges; hover shows exact values |
| US-28 | As a player, I want to browse my recent Faceit matches | Paginated list; each entry shows map, score, K/D, ELO delta |
| US-29 | As a player, I want to open a Faceit match demo in the viewer | Click-through from match history to 2D viewer works seamlessly |

### Grenade Lineups

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-30 | As a player, I want lineups auto-extracted from my demos | After parsing, grenade throws appear in lineup catalog with correct data |
| US-31 | As a player, I want to browse lineups by map and type | Filter UI works; results update; 2D preview shows throw + landing |
| US-32 | As a player, I want to save lineups to my personal collection | Save button adds to collection; appears in "My Lineups" view |
| US-33 | As a player, I want to jump to the lineup's source tick in the demo | "View in Demo" link opens viewer at the exact tick |

---

## 7. Non-Functional Requirements

### 7.1 Performance

| Metric | Target |
|--------|--------|
| Demo parse time (avg 100 MB file) | < 10 seconds (local disk, no upload) |
| 2D Viewer frame rate | Stable 60 FPS at 1080p |
| Heatmap render (single demo) | < 2 seconds |
| Heatmap render (10-demo aggregate) | < 5 seconds |
| App startup time (cold) | < 3 seconds to interactive UI |
| Tick data query latency (SQLite) | < 50ms for a 1000-tick range |

### 7.2 Security

- Faceit OAuth 2.0 with PKCE (loopback redirect, RFC 8252)
- Refresh tokens stored in OS keychain (encrypted at rest)
- Access tokens held in memory only (never persisted to disk)
- `.dem` file validation (magic bytes, size limits) before parsing
- No network listeners except during OAuth callback (temporary, localhost only)
- SQLite database file permissions: owner read/write only

### 7.3 Accessibility

- WCAG 2.1 AA compliance for all non-canvas UI
- Keyboard navigation for all controls (playback, menus, forms)
- Screen reader support for UI chrome (aria labels, roles)
- Color-blind-friendly palette option for team colors
- Canvas elements: provide text alternatives where feasible (scoreboard, stats)

### 7.4 Platform Support

| Platform | Minimum Version | WebView Engine |
|----------|----------------|---------------|
| macOS | 12 (Monterey)+ | WebKit (WKWebView) |
| Windows | 10 (1903)+ | WebView2 (Chromium-based) |
| Linux | Ubuntu 22.04+ | WebKitGTK |

- WebGL 2.0 required for PixiJS rendering (all supported WebView versions support this)
- Requires internet access only for Faceit OAuth and API calls

### 7.5 Installation & Distribution

| Dimension | Target |
|-----------|--------|
| Install size | < 30 MB |
| Installer format (macOS) | `.dmg` or `.app` bundle |
| Installer format (Windows) | `.exe` (NSIS) or `.msi` |
| Installer format (Linux) | `.AppImage` or `.deb` |
| Auto-update | Built-in update checker + download |

### 7.6 Test Coverage & Quality

The project follows **Test-Driven Development (TDD)**. Every feature is developed using the Red-Green-Refactor cycle: write a failing test first, implement the minimum code to pass, then refactor.

| Metric | Target |
|--------|--------|
| Go backend line coverage | >= 80% |
| Go critical-path coverage (parser, auth, SQLite store) | >= 90% |
| Frontend component/hook test coverage | >= 75% |
| Frontend utility/store coverage | >= 90% |
| E2E critical path coverage | 100% of US-01 (install), US-04 (import), US-09 (viewer), US-22 (strat board) |
| CI gate | Zero merge to main without all tests passing |

**Test execution time budgets:**

| Test Tier | Budget |
|-----------|--------|
| Unit tests (Go + TS) | < 30 seconds total |
| Integration tests (temp SQLite, MSW) | < 2 minutes total |
| End-to-end tests (Playwright) | < 10 minutes total |

---

## 8. Data Models

### 8.1 Core Entities

#### User

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| faceit_id | TEXT | Unique; from Faceit OAuth |
| nickname | TEXT | Faceit display name |
| avatar_url | TEXT | Faceit avatar |
| faceit_elo | INTEGER | Last known ELO |
| faceit_level | INTEGER | 1-10 |
| country | TEXT | ISO country code |
| created_at | TEXT | ISO 8601 datetime |
| updated_at | TEXT | ISO 8601 datetime |

#### Demo

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| user_id | INTEGER | FK -> User |
| faceit_match_id | TEXT | Nullable; for auto-imported demos |
| map_name | TEXT | e.g., "de_dust2" |
| file_path | TEXT | Absolute path to local `.dem` file |
| file_size | INTEGER | Bytes |
| status | TEXT | imported / parsing / ready / error |
| total_ticks | INTEGER | Set after parsing |
| tick_rate | REAL | Ticks per second |
| duration_secs | INTEGER | Match duration |
| match_date | TEXT | ISO 8601 datetime |
| created_at | TEXT | ISO 8601 datetime |

#### Round

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| demo_id | INTEGER | FK -> Demo |
| round_number | INTEGER | 1-based |
| start_tick | INTEGER | |
| end_tick | INTEGER | |
| winner_side | TEXT | CT / T |
| win_reason | TEXT | elimination / bomb_exploded / defused / time |
| ct_score | INTEGER | Score after this round |
| t_score | INTEGER | Score after this round |

#### PlayerRound

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| round_id | INTEGER | FK -> Round |
| steam_id | TEXT | Steam64 ID |
| player_name | TEXT | |
| team_side | TEXT | CT / T |
| kills | INTEGER | |
| deaths | INTEGER | |
| assists | INTEGER | |
| damage | INTEGER | |
| headshot_kills | INTEGER | |
| first_kill | INTEGER | 0/1 boolean |
| first_death | INTEGER | 0/1 boolean |
| clutch_kills | INTEGER | |

#### TickData

| Field | Type | Notes |
|-------|------|-------|
| demo_id | INTEGER | FK -> Demo; part of composite PK |
| tick | INTEGER | Part of composite PK |
| steam_id | TEXT | Part of composite PK |
| x | REAL | World-space X |
| y | REAL | World-space Y |
| z | REAL | World-space Z |
| yaw | REAL | View angle (horizontal) |
| health | INTEGER | |
| armor | INTEGER | |
| is_alive | INTEGER | 0/1 boolean |
| weapon | TEXT | Active weapon |

*Index: `(demo_id, tick)` composite index for range scan queries.*

#### GameEvent

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| demo_id | INTEGER | FK -> Demo |
| round_id | INTEGER | FK -> Round |
| tick | INTEGER | |
| event_type | TEXT | kill / grenade_throw / grenade_detonate / bomb_plant / bomb_defuse |
| attacker_steam_id | TEXT | Nullable |
| victim_steam_id | TEXT | Nullable |
| weapon | TEXT | Nullable |
| x | REAL | Event position |
| y | REAL | |
| z | REAL | |
| extra_data | TEXT | JSON string for event-specific data (headshot, penetration, flash assist) |

#### StrategyBoard

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| title | TEXT | |
| map_name | TEXT | |
| board_state | TEXT | JSON serialized board state |
| created_at | TEXT | ISO 8601 datetime |
| updated_at | TEXT | ISO 8601 datetime |

#### GrenadeLineup

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| demo_id | INTEGER | FK -> Demo (source, nullable) |
| tick | INTEGER | Source tick in demo |
| map_name | TEXT | |
| grenade_type | TEXT | smoke / flash / he / molotov |
| throw_x | REAL | Thrower position |
| throw_y | REAL | |
| throw_z | REAL | |
| throw_yaw | REAL | Aim angle |
| throw_pitch | REAL | Aim angle |
| land_x | REAL | Landing/detonation position |
| land_y | REAL | |
| land_z | REAL | |
| title | TEXT | User-provided or auto-generated |
| description | TEXT | |
| tags | TEXT | JSON array of tags |
| is_favorite | INTEGER | 0/1 boolean; default 0 |
| created_at | TEXT | ISO 8601 datetime |

#### FaceitMatch

| Field | Type | Notes |
|-------|------|-------|
| id | INTEGER | Primary key (autoincrement) |
| user_id | INTEGER | FK -> User |
| faceit_match_id | TEXT | Unique per user |
| map_name | TEXT | |
| score_team | INTEGER | User's team score |
| score_opponent | INTEGER | Opponent team score |
| result | TEXT | win / loss / draw |
| elo_before | INTEGER | |
| elo_after | INTEGER | |
| kills | INTEGER | |
| deaths | INTEGER | |
| assists | INTEGER | |
| demo_url | TEXT | Faceit demo download URL |
| demo_id | INTEGER | FK -> Demo (nullable, if imported) |
| played_at | TEXT | ISO 8601 datetime |
| created_at | TEXT | ISO 8601 datetime |

---

## 9. Wails Bindings Overview

Instead of a REST API, the Go backend exposes methods to the frontend via Wails bindings. The frontend calls these as async TypeScript functions.

### 9.1 Binding Groups

#### Auth

| Method | Signature | Description |
|--------|----------|-------------|
| `StartLogin` | `() -> LoginResult` | Start loopback OAuth; opens system browser |
| `GetCurrentUser` | `() -> User \| null` | Get logged-in user profile |
| `Logout` | `() -> void` | Clear tokens from keychain |
| `RefreshProfile` | `() -> User` | Re-fetch Faceit profile data |

#### Demos

| Method | Signature | Description |
|--------|----------|-------------|
| `ImportDemo` | `(path: string) -> Demo` | Import and parse a single `.dem` file |
| `ImportFolder` | `(path: string) -> Demo[]` | Recursively import `.dem` files from folder |
| `ListDemos` | `(opts: ListOpts) -> Demo[]` | List user's demos (sortable, filterable) |
| `GetDemo` | `(id: number) -> Demo` | Get demo metadata |
| `DeleteDemo` | `(id: number, deleteFile: bool) -> void` | Delete demo data; optionally remove `.dem` |
| `GetRounds` | `(demoId: number) -> Round[]` | Get round summaries for a demo |
| `GetRoundDetail` | `(roundId: number) -> RoundDetail` | Get round detail + player stats |
| `GetTicks` | `(demoId: number, from: number, to: number) -> TickData[]` | Get tick data for a range |
| `GetEvents` | `(demoId: number, filters: EventFilter) -> GameEvent[]` | Get filtered game events |

#### Heatmaps

| Method | Signature | Description |
|--------|----------|-------------|
| `GetHeatmapData` | `(demoIds: number[], filters: HeatmapFilter) -> HeatmapPoint[]` | Aggregated heatmap data |

#### Strategy Boards

| Method | Signature | Description |
|--------|----------|-------------|
| `ListBoards` | `() -> StrategyBoard[]` | List all strategy boards |
| `CreateBoard` | `(title: string, mapName: string) -> StrategyBoard` | Create a new board |
| `GetBoard` | `(id: number) -> StrategyBoard` | Get board with state |
| `SaveBoard` | `(id: number, state: string) -> void` | Save board state (JSON) |
| `DeleteBoard` | `(id: number) -> void` | Delete a board |
| `ExportBoardJSON` | `(id: number) -> string` | Export board as JSON string |
| `ImportBoardJSON` | `(json: string) -> StrategyBoard` | Import board from JSON |

#### Grenade Lineups

| Method | Signature | Description |
|--------|----------|-------------|
| `ListLineups` | `(filters: LineupFilter) -> GrenadeLineup[]` | List/search lineups |
| `GetLineup` | `(id: number) -> GrenadeLineup` | Get lineup detail |
| `UpdateLineup` | `(id: number, data: LineupUpdate) -> GrenadeLineup` | Update title, description, tags |
| `DeleteLineup` | `(id: number) -> void` | Delete a lineup |
| `ToggleFavorite` | `(id: number) -> void` | Toggle favorite status |

#### Faceit

| Method | Signature | Description |
|--------|----------|-------------|
| `GetFaceitProfile` | `() -> FaceitProfile` | Get user's Faceit profile (cached) |
| `GetEloHistory` | `(days: number) -> EloPoint[]` | Get ELO history |
| `GetMatches` | `(opts: MatchListOpts) -> FaceitMatch[]` | Get match history (paginated) |
| `SyncMatches` | `() -> SyncResult` | Trigger manual match sync |
| `ImportMatchDemo` | `(matchId: string) -> Demo` | Download and import demo from Faceit match |

#### System

| Method | Signature | Description |
|--------|----------|-------------|
| `OpenFileDialog` | `() -> string` | Native file picker for `.dem` files |
| `OpenFolderDialog` | `() -> string` | Native folder picker |
| `GetAppInfo` | `() -> AppInfo` | App version, data dir, DB size |
| `CheckForUpdates` | `() -> UpdateInfo \| null` | Check if a newer version is available |

### 9.2 Frontend Call Pattern

```typescript
import { ImportDemo, ListDemos } from '../../wailsjs/go/main/App';

// Wails bindings are called as regular async functions
const demo = await ImportDemo('/path/to/demo.dem');
const demos = await ListDemos({ sortBy: 'date', order: 'desc' });
```

TanStack Query wraps these bindings for caching and background refetch:

```typescript
const { data: demos } = useQuery({
  queryKey: ['demos', filters],
  queryFn: () => ListDemos(filters),
});
```

---

*Cross-references: [ARCHITECTURE.md](ARCHITECTURE.md) for detailed system design, [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for delivery phases, [TASK_BREAKDOWN.md](TASK_BREAKDOWN.md) for granular tasks.*
