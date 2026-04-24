# Product — User Stories & State Handling

> **Siblings:** [vision](vision.md) · [personas](personas.md) · [features](features.md) · [non-functional](non-functional.md) · [data-models](data-models.md) · [wails-bindings](wails-bindings.md)

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
| US-28 | As a player, I want to browse my recent Faceit matches | Paginated list of matches from the last 30 days; each entry shows map, score, K/D |
| US-29 | As a player, I want to open a Faceit match demo in the viewer | Click-through from match history to 2D viewer works seamlessly |

### Grenade Lineups

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| US-30 | As a player, I want lineups auto-extracted from my demos | After parsing, grenade throws appear in lineup catalog with correct data |
| US-31 | As a player, I want to browse lineups by map and type | Filter UI works; results update; 2D preview shows throw + landing |
| US-32 | As a player, I want to save lineups to my personal collection | Save button adds to collection; appears in "My Lineups" view |
| US-33 | As a player, I want to jump to the lineup's source tick in the demo | "View in Demo" link opens viewer at the exact tick |

---

## 7. Empty States, Error States & Onboarding

### 7.1 First-Run Onboarding

On first launch (no `config.json` exists):

1. **Welcome screen** with app logo and one-line description
2. **Faceit login prompt** — "Connect your Faceit account" button (skippable; app works without auth for local demo analysis)
3. **Import prompt** — "Import your first demo" with drag-drop zone and folder picker
4. After dismissal, user lands on the Dashboard

The onboarding flow does not reappear after first completion (flag in `config.json`).

### 7.2 Empty States

| Page | Empty State |
|------|-------------|
| Dashboard | Illustration + "Import your first demo to get started" CTA |
| Demo Library | Drag-drop zone prominently displayed + "No demos imported yet" |
| Viewer | Not reachable without a demo (route guard redirects to library) |
| Heatmaps | "Import and parse demos to generate heatmaps" message |
| Strategy Board | "Create your first strategy" CTA button |
| Faceit Dashboard | "Connect your Faceit account" login CTA (if not authed) or "Sync your matches" button (if authed but no data) |
| Grenade Lineups | "Parse demos to discover grenade lineups" message |

### 7.3 Error States

| Scenario | Behavior |
|----------|----------|
| Faceit OAuth failure (user cancels, timeout, network error) | Toast notification with error message; user stays on login page; retry button available |
| Demo parse failure (corrupt file, unsupported format) | Demo record marked as `error` status in library; error message shown in toast; other demos in batch continue |
| Demo parse partial failure (crash mid-parse) | Partial data rolled back (SQLite transaction); demo marked `error`; user can retry |
| Network offline (Faceit sync) | Toast "No internet connection — Faceit features unavailable"; local features remain fully functional |
| SQLite error (disk full, permissions) | Modal error with explanation; suggest checking disk space; app remains open for read-only browsing |
| Invalid .dem file (wrong magic bytes, too small) | Rejected at import with "Not a valid CS2 demo file" message; file skipped in batch import |
