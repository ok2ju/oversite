# Architecture — Data Flow Diagrams

> **Siblings:** [overview](overview.md) · [structure](structure.md) · [components](components.md) · [wails-bindings](wails-bindings.md) · [database](database.md) · [crosscutting](crosscutting.md) · [testing](testing.md)

---

## Data Flow Diagrams

### Demo Import & Parse

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

### Viewer Playback

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

### Faceit OAuth Loopback Flow

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

### Faceit Match Sync

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

### Demo Download from Match Row

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
 │             │                      │  (auto-trigger parse — see Demo Import & Parse) │
 │             │                      │                      │               │           │
 │             │ Invalidate           │                      │               │           │
 │             │ ['faceit-matches']   │                      │               │           │
 │             │ ['demos'] queries    │                      │               │           │
 │◀ Row flips  │                      │                      │               │           │
 │  to "Demo   │                      │                      │               │           │
 │   ready"    │                      │                      │               │           │
```

### Dashboard / Demos → Match Details → Viewer

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
