# Product — Information Architecture & Features

> **Siblings:** [vision](vision.md) · [personas](personas.md) · [user-stories](user-stories.md) · [non-functional](non-functional.md) · [data-models](data-models.md) · [wails-bindings](wails-bindings.md)

---

## Information Architecture

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

## Feature Specifications

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

#### F4.2 Match History

- Paginated list of Faceit matches from the last 30 days
- Each entry: map, score, K/D/A, date
- Rows with a Faceit-hosted demo but no local import show an **Import demo** button that downloads and auto-parses the demo in-process
- Rows with an imported demo are clickable; click navigates to **Match Details** (F4.5). If the demo is still parsing when clicked, the row shows an inline "Parsing…" indicator and navigates automatically once parsing completes.
- Rows with neither a local demo nor a Faceit demo URL are inert
- Filter by map, result (W/L)

#### F4.4 Auto-Fetch

- On login, automatically fetch the user's recent Faceit match history
- In-process sync (no background worker -- runs in the Go backend directly)
- Optionally auto-download demos from Faceit match rooms

#### F4.5 Match Details

- Dedicated route (`/matches/:demoId`) reached by clicking a match row with an imported demo (from the dashboard or the Demo Library)
- Shows the match scoreboard, round-by-round timeline, map/mode/duration, and per-player stats
- Toolbar includes a **Play demo** button that opens the 2D Viewer (`/demos/:demoId`); enabled only when the demo status is `ready`

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
- Clicking a row navigates to **Match Details** (F4.5), not directly to the 2D Viewer. If the demo is still parsing, the click shows a "Parsing…" indicator and navigates automatically once parsing completes.
- Row-hover action buttons still include direct **Play** (to `/demos/:id`, enabled only when status = `ready`) and **Delete**

#### F6.3 Auto-Scan (Optional)

- Configure a "watch folder" that the app scans for new `.dem` files on startup
- Default to CS2's demo download directory if detectable

#### F6.4 Storage Management

- Show total database size and demo count
- Delete demos (removes parsed data from SQLite; optionally removes source `.dem` file)
- Re-parse a demo (useful after parser updates)

### F7: Settings

Application-wide preferences accessible via the `/settings` route.

#### F7.1 Appearance

- Theme toggle: light / dark / system (persisted in `config.json`)
- Theme applies globally via Tailwind CSS class strategy

#### F7.2 Demo Management Settings

- **Watch folder**: Optional directory path the app scans for new `.dem` files on startup
- Default to CS2's demo download directory if detectable
- Browse button opens native folder picker

#### F7.3 Faceit Integration Settings

- **Auto-fetch**: Toggle automatic Faceit match sync on login (default: on)
- **Sync interval**: How often to check for new matches (manual / on launch / periodic)

#### F7.4 About

- App version, build info, and link to check for updates
- Link to project repository / issue tracker
