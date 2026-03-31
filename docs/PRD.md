# Oversite -- Product Requirements Document

> **Version:** 1.0
> **Last Updated:** 2026-03-31
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
9. [API Design Overview](#9-api-design-overview)

---

## 1. Product Vision

**Oversite** is a web-based 2D demo viewer and analytics platform for Counter-Strike 2 (CS2) Faceit players. It transforms raw `.dem` files into interactive playback, heatmaps, strategy boards, and stat dashboards -- giving competitive players the tools to study their game without leaving the browser.

### Problem Statement

CS2 players on Faceit lack a unified, web-accessible tool to:

- Review demo playback in 2D (top-down) without launching CS2
- Aggregate statistics across multiple demos and Faceit matches
- Collaborate on strategies in real time with teammates
- Catalog and share grenade lineups extracted from actual gameplay

### Product Goals

| # | Goal | Success Metric |
|---|------|---------------|
| G1 | Instant demo playback | < 30s from upload to first frame rendered |
| G2 | Cross-demo analytics | Heatmaps aggregating 10+ demos render in < 5s |
| G3 | Real-time collaboration | Strat board syncs across 5 concurrent users with < 200ms latency |
| G4 | Grenade knowledge base | Users can save, tag, and share lineups |
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
| Technical Level | Comfortable with web apps, not a developer |

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
| Pain Point | Existing tools require desktop apps or CS2 running |
| Key Features | 2D Viewer (speed controls), Heatmaps, shareable links |
| Technical Level | High; comfortable with web tools and data |

---

## 3. Technology Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| **Frontend** | Next.js 14+ (App Router) | Server components, file-based routing |
| **UI Components** | shadcn/ui + Tailwind CSS | Accessible, themeable component library |
| **2D Rendering** | PixiJS v8 | WebGL canvas for performant 2D playback |
| **State Management** | Zustand | Lightweight stores, selector-based subscriptions |
| **Data Fetching** | TanStack Query v5 | Server state caching, background refetch |
| **Backend** | Go 1.22+ + chi router | High-performance HTTP + middleware |
| **Demo Parser** | markus-wa/demoinfocs-golang v5 | Only mature Go-based CS2 demo parser |
| **Database** | PostgreSQL 16 + TimescaleDB | Hypertables for tick-level time-series data |
| **SQL Generation** | sqlc | Type-safe Go code from SQL queries |
| **Cache / Queue** | Redis 7 | Sessions, caching, job queue via Redis Streams |
| **Object Storage** | MinIO | S3-compatible; stores `.dem` files and assets |
| **Real-time** | WebSocket (gorilla/websocket) | Bidirectional for playback sync + strat collab |
| **CRDT** | Yjs | Conflict-free collaboration for strategy board |
| **Auth** | Faceit OAuth 2.0 | Single sign-on for target audience |
| **Containerization** | Docker Compose | Local dev + production-ready |

---

## 4. Information Architecture

### Next.js Route Structure

```
app/
├── (auth)/
│   ├── login/page.tsx              # Faceit OAuth login
│   └── callback/page.tsx           # OAuth callback handler
├── (app)/
│   ├── layout.tsx                  # Authenticated shell (sidebar + header)
│   ├── dashboard/page.tsx          # Faceit stats overview
│   ├── demos/
│   │   ├── page.tsx                # Demo library (list/grid)
│   │   └── [demoId]/
│   │       ├── page.tsx            # 2D Viewer (PixiJS canvas)
│   │       └── heatmap/page.tsx    # Heatmap view for this demo
│   ├── heatmaps/page.tsx           # Cross-demo aggregated heatmaps
│   ├── strats/
│   │   ├── page.tsx                # Strategy board list
│   │   └── [stratId]/page.tsx      # Collaborative strat board (Yjs)
│   ├── lineups/
│   │   ├── page.tsx                # Grenade lineup library
│   │   └── [lineupId]/page.tsx     # Lineup detail + 2D preview
│   └── settings/page.tsx           # User preferences
└── api/                            # Next.js API routes (BFF proxy)
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

#### F3.3 Real-Time Collaboration

- **Protocol**: WebSocket with Yjs CRDT for state synchronization
- Multiple users can draw simultaneously; changes merge automatically
- User cursors visible with name labels (Yjs Awareness protocol)
- Presence indicators showing who is viewing the board

#### F3.4 Persistence & Sharing

- Save strategy boards to user's library
- Shareable link (read-only or edit access)
- Export as PNG image
- Duplicate/fork existing boards

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
- Click to open demo (if available) in 2D Viewer
- Filter by map, result (W/L), date range

#### F4.4 Performance Trends

- Rolling averages for K/D ratio, win rate, ADR
- Map-specific stats breakdown
- Best/worst maps identification

#### F4.5 Auto-Fetch

- On login, automatically fetch the user's recent Faceit match history
- Background job to periodically sync new matches
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

#### F5.4 Community Sharing (Future)

- Submit lineups to a public catalog
- Upvote/downvote system
- Filter by popularity and recency

---

## 6. User Stories

### Authentication & Onboarding

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-01 | As a Faceit player, I want to log in with my Faceit account so I don't need a separate password | OAuth 2.0 flow completes; user profile created/updated from Faceit data |
| US-02 | As a new user, I want to see a quick onboarding tour so I understand the main features | First-login modal with 3-4 slides; dismissible; doesn't show again |
| US-03 | As a user, I want to log out and have my session cleared | Session cookie/token invalidated; redirected to login page |

### Demo Management

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-04 | As a player, I want to upload a `.dem` file so I can review it in 2D | File upload accepts `.dem` (max 500 MB); progress indicator; parsing starts automatically |
| US-05 | As a player, I want to see a list of my uploaded demos | Library page shows demos sorted by date; shows map, date, players, status |
| US-06 | As a player, I want to delete a demo I no longer need | Confirm dialog; removes demo file from storage and parsed data from DB |
| US-07 | As a player, I want demos from my Faceit matches auto-imported | Background sync fetches demos from recent Faceit matches; status shown in library |

### 2D Viewer

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-08 | As a player, I want to watch a demo in 2D top-down view | PixiJS canvas renders map + players + events; plays at real-time speed |
| US-09 | As a player, I want to control playback speed | Speed selector works (0.25x-4x); playback visually matches selected speed |
| US-10 | As a player, I want to scrub to any point in the demo | Timeline slider seeks to correct tick; canvas updates immediately |
| US-11 | As a player, I want to jump to a specific round | Round selector lists all rounds; clicking jumps to round start tick |
| US-12 | As a player, I want to see kill events on the map | Kill lines drawn from killer to victim; kill-feed updates; death X appears |
| US-13 | As a player, I want to see grenade effects on the map | Smokes, flashes, HEs, molotovs render with appropriate visual effects and timing |
| US-14 | As a player, I want to zoom and pan the map | Scroll-to-zoom works; click-drag pans; mini-map shows viewport position |
| US-15 | As a player, I want to see the scoreboard for the current round | Toggle-able overlay shows accurate per-player stats for the viewed round |

### Heatmaps & Analytics

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-16 | As a player, I want to see a kill heatmap for a demo | KDE heatmap overlays on map image; color gradient indicates density |
| US-17 | As a player, I want to filter heatmaps by side, weapon, or player | Filters update heatmap in real-time; UI shows active filters |
| US-18 | As a player, I want to see aggregated heatmaps across multiple demos | User can select demos to aggregate; combined heatmap renders correctly |
| US-19 | As a player, I want to see my per-demo statistics | Stats page shows K/D/A, ADR, HS%, KAST, Rating for each demo |
| US-20 | As a player, I want to see stat trends over time | Line charts render with correct data points; time range selectable |

### Strategy Board

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-21 | As an IGL, I want to draw strategies on a map | Drawing tools (freehand, line, arrow, shapes) work on map canvas |
| US-22 | As an IGL, I want to place player tokens on the map | CT/T tokens draggable; labeled; snap to reasonable positions |
| US-23 | As an IGL, I want to collaborate with teammates in real time | Multiple users see each other's changes within 200ms; no conflicts |
| US-24 | As an IGL, I want to share a strategy board with a link | Generated link opens board in read-only or edit mode as configured |
| US-25 | As a user, I want to export a strategy board as an image | PNG export captures the full board state at current zoom |

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
| US-33 | As a player, I want to jump to the lineup's source moment in the demo | "View in Demo" link opens viewer at the exact tick of the throw |

---

## 7. Non-Functional Requirements

### 7.1 Performance

| Metric | Target |
|--------|--------|
| Demo upload + parse (avg 100 MB file) | < 30 seconds end-to-end |
| 2D Viewer frame rate | Stable 60 FPS at 1080p |
| Heatmap render (single demo) | < 2 seconds |
| Heatmap render (10-demo aggregate) | < 5 seconds |
| API response time (p95) | < 200ms |
| Time to Interactive (TTI) | < 3 seconds on broadband |
| Strategy board sync latency | < 200ms between participants |

### 7.2 Security

- All traffic over HTTPS (TLS 1.3)
- OAuth 2.0 with PKCE for Faceit authentication
- Server-side session management via Redis
- CSRF protection on all state-changing endpoints
- Input validation and sanitization on all API endpoints
- Rate limiting on upload and API endpoints
- `.dem` file validation (magic bytes, size limits) before parsing
- No direct database exposure; all access through Go API
- MinIO bucket policies: private by default, signed URLs for access

### 7.3 Accessibility

- WCAG 2.1 AA compliance for all non-canvas UI
- Keyboard navigation for all controls (playback, menus, forms)
- Screen reader support for UI chrome (aria labels, roles)
- Color-blind-friendly palette option for team colors
- Canvas elements: provide text alternatives where feasible (scoreboard, stats)

### 7.4 Browser Support

| Browser | Minimum Version |
|---------|----------------|
| Chrome | 110+ |
| Firefox | 110+ |
| Safari | 16+ |
| Edge | 110+ |

- WebGL 2.0 required for PixiJS rendering
- WebSocket support required for real-time features
- JavaScript must be enabled

### 7.5 Scalability Targets (Initial)

| Dimension | Target |
|-----------|--------|
| Concurrent users | 100 |
| Stored demos | 10,000 |
| Max demo file size | 500 MB |
| Max strategy boards per user | 50 |
| Max concurrent strat board collaborators | 10 |

---

## 8. Data Models

### 8.1 Core Entities

#### User

| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| faceit_id | VARCHAR(64) | Unique; from Faceit OAuth |
| nickname | VARCHAR(64) | Faceit display name |
| avatar_url | TEXT | Faceit avatar |
| faceit_elo | INTEGER | Last known ELO |
| faceit_level | SMALLINT | 1-10 |
| country | VARCHAR(2) | ISO country code |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

#### Demo

| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| user_id | UUID | FK → User |
| faceit_match_id | VARCHAR(64) | Nullable; for auto-imported demos |
| map_name | VARCHAR(32) | e.g., "de_dust2" |
| file_path | TEXT | MinIO object key |
| file_size | BIGINT | Bytes |
| status | VARCHAR(16) | uploaded / parsing / ready / error |
| total_ticks | INTEGER | Set after parsing |
| tick_rate | REAL | Ticks per second |
| duration_secs | INTEGER | Match duration |
| match_date | TIMESTAMPTZ | When the match was played |
| created_at | TIMESTAMPTZ | |

#### Round

| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| demo_id | UUID | FK → Demo |
| round_number | SMALLINT | 1-based |
| start_tick | INTEGER | |
| end_tick | INTEGER | |
| winner_side | VARCHAR(2) | CT / T |
| win_reason | VARCHAR(32) | elimination / bomb_exploded / defused / time |
| ct_score | SMALLINT | Score after this round |
| t_score | SMALLINT | Score after this round |

#### PlayerRound

| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| round_id | UUID | FK → Round |
| steam_id | VARCHAR(20) | Steam64 ID |
| player_name | VARCHAR(64) | |
| team_side | VARCHAR(2) | CT / T |
| kills | SMALLINT | |
| deaths | SMALLINT | |
| assists | SMALLINT | |
| damage | INTEGER | |
| headshot_kills | SMALLINT | |
| first_kill | BOOLEAN | Got the opening kill |
| first_death | BOOLEAN | Died first |
| clutch_kills | SMALLINT | |

#### TickData (TimescaleDB Hypertable)

| Field | Type | Notes |
|-------|------|-------|
| time | TIMESTAMPTZ | Hypertable partition key (synthetic: demo timestamp + tick offset) |
| demo_id | UUID | FK → Demo |
| tick | INTEGER | |
| steam_id | VARCHAR(20) | |
| x | REAL | World-space X |
| y | REAL | World-space Y |
| z | REAL | World-space Z |
| yaw | REAL | View angle (horizontal) |
| health | SMALLINT | |
| armor | SMALLINT | |
| is_alive | BOOLEAN | |
| weapon | VARCHAR(32) | Active weapon |

#### GameEvent

| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| demo_id | UUID | FK → Demo |
| round_id | UUID | FK → Round |
| tick | INTEGER | |
| event_type | VARCHAR(32) | kill / grenade_throw / grenade_detonate / bomb_plant / bomb_defuse |
| attacker_steam_id | VARCHAR(20) | Nullable |
| victim_steam_id | VARCHAR(20) | Nullable |
| weapon | VARCHAR(32) | Nullable |
| x | REAL | Event position |
| y | REAL | |
| z | REAL | |
| extra_data | JSONB | Event-specific data (headshot, penetration, flash assist, etc.) |

#### StrategyBoard

| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| user_id | UUID | FK → User (owner) |
| title | VARCHAR(128) | |
| map_name | VARCHAR(32) | |
| yjs_state | BYTEA | Serialized Yjs document |
| share_mode | VARCHAR(16) | private / read_only / editable |
| share_token | VARCHAR(64) | Unique; for share links |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

#### GrenadeLineup

| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| user_id | UUID | FK → User (who saved it) |
| demo_id | UUID | FK → Demo (source, nullable) |
| tick | INTEGER | Source tick in demo |
| map_name | VARCHAR(32) | |
| grenade_type | VARCHAR(16) | smoke / flash / he / molotov |
| throw_x | REAL | Thrower position |
| throw_y | REAL | |
| throw_z | REAL | |
| throw_yaw | REAL | Aim angle |
| throw_pitch | REAL | Aim angle |
| land_x | REAL | Landing/detonation position |
| land_y | REAL | |
| land_z | REAL | |
| title | VARCHAR(128) | User-provided or auto-generated |
| description | TEXT | |
| tags | TEXT[] | Array of tags |
| is_favorite | BOOLEAN | Default false |
| created_at | TIMESTAMPTZ | |

#### FaceitMatch

| Field | Type | Notes |
|-------|------|-------|
| id | UUID | Primary key |
| user_id | UUID | FK → User |
| faceit_match_id | VARCHAR(64) | Unique per user |
| map_name | VARCHAR(32) | |
| score_team | SMALLINT | User's team score |
| score_opponent | SMALLINT | Opponent team score |
| result | VARCHAR(4) | win / loss / draw |
| elo_before | INTEGER | |
| elo_after | INTEGER | |
| kills | SMALLINT | |
| deaths | SMALLINT | |
| assists | SMALLINT | |
| demo_url | TEXT | Faceit demo download URL |
| demo_id | UUID | FK → Demo (nullable, if imported) |
| played_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | |

---

## 9. API Design Overview

### 9.1 REST API Base Path

All API endpoints are served under `/api/v1/`.

### 9.2 Endpoint Groups

#### Authentication

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/auth/faceit` | Initiate Faceit OAuth flow |
| GET | `/api/v1/auth/faceit/callback` | Handle OAuth callback |
| POST | `/api/v1/auth/logout` | Invalidate session |
| GET | `/api/v1/auth/me` | Get current user profile |

#### Demos

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/demos` | List user's demos (paginated) |
| POST | `/api/v1/demos` | Upload a new demo file |
| GET | `/api/v1/demos/:id` | Get demo metadata |
| DELETE | `/api/v1/demos/:id` | Delete a demo |
| GET | `/api/v1/demos/:id/rounds` | Get round summaries |
| GET | `/api/v1/demos/:id/rounds/:num` | Get round detail + player stats |
| GET | `/api/v1/demos/:id/ticks` | Get tick data (paginated by tick range) |
| GET | `/api/v1/demos/:id/events` | Get game events (filterable) |
| GET | `/api/v1/demos/:id/heatmap` | Get heatmap data points |

#### Heatmaps (Aggregated)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/heatmaps/aggregate` | Generate aggregated heatmap from multiple demos |

#### Strategy Boards

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/strats` | List user's strategy boards |
| POST | `/api/v1/strats` | Create a new board |
| GET | `/api/v1/strats/:id` | Get board metadata |
| PUT | `/api/v1/strats/:id` | Update board settings |
| DELETE | `/api/v1/strats/:id` | Delete a board |
| GET | `/api/v1/strats/shared/:token` | Access shared board |
| POST | `/api/v1/strats/:id/export` | Export board as PNG |

#### Grenade Lineups

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/lineups` | List/search lineups |
| POST | `/api/v1/lineups` | Save a lineup |
| GET | `/api/v1/lineups/:id` | Get lineup detail |
| PUT | `/api/v1/lineups/:id` | Update lineup |
| DELETE | `/api/v1/lineups/:id` | Delete a lineup |
| POST | `/api/v1/lineups/:id/favorite` | Toggle favorite |

#### Faceit

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/faceit/profile` | Get user's Faceit profile (cached) |
| GET | `/api/v1/faceit/elo-history` | Get ELO history |
| GET | `/api/v1/faceit/matches` | Get match history (paginated) |
| POST | `/api/v1/faceit/sync` | Trigger manual match sync |
| POST | `/api/v1/faceit/matches/:id/import` | Import demo from a Faceit match |

### 9.3 WebSocket Endpoints

| Path | Purpose |
|------|---------|
| `/ws/viewer/:demoId` | Real-time tick data streaming for 2D viewer |
| `/ws/strat/:stratId` | Yjs sync protocol for strategy board collaboration |

### 9.4 Common Response Format

```json
{
  "data": { ... },
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 142
  }
}
```

Error responses:

```json
{
  "error": {
    "code": "DEMO_PARSE_FAILED",
    "message": "Failed to parse demo file: unsupported format"
  }
}
```

---

*Cross-references: [ARCHITECTURE.md](ARCHITECTURE.md) for detailed system design, [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) for delivery phases, [TASK_BREAKDOWN.md](TASK_BREAKDOWN.md) for granular tasks.*
