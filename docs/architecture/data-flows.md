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
 │                        │  ImportDemoByPath  │                    │                │
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
 │                        │                    │  (status=imported) │                │
 │                        │                    │───────────────────▶│                │
 │                        │  Demo (imported)   │                    │                │
 │                        │◀───────────────────│                    │                │
 │  Show in library       │                    │                    │                │
 │◀───────────────────────│                    │                    │                │
 │                        │                    │  parseDemo (go)    │                │
 │                        │                    │  status=parsing    │                │
 │                        │                    │───────────────────▶│                │
 │                        │                    │                    │                │
 │                        │                    │  Parse demo        │                │
 │                        │                    │  (demoinfocs)      │                │
 │                        │                    │                    │                │
 │                        │                    │  Emit              │                │
 │                        │                    │  demo:parse:       │                │
 │                        │  parse progress    │  progress events   │                │
 │                        │◀───────────────────│                    │                │
 │  Progress UI           │                    │                    │                │
 │◀───────────────────────│                    │                    │                │
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
 │                        │  parse complete    │                    │                │
 │                        │◀───────────────────│                    │                │
 │  Library row flips to  │                    │                    │                │
 │  "ready"; clickable    │                    │                    │                │
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

### Heatmap Aggregation

```
User                    React SPA          Go Backend            SQLite
 │                        │                    │                    │
 │  Open Heatmaps         │                    │                    │
 │───────────────────────▶│                    │                    │
 │                        │  GetUniqueWeapons  │                    │
 │                        │  GetUniquePlayers  │                    │
 │                        │───────────────────▶│                    │
 │                        │                    │  SELECT DISTINCT   │
 │                        │                    │───────────────────▶│
 │                        │                    │◀───────────────────│
 │                        │  filter options    │                    │
 │                        │◀───────────────────│                    │
 │                        │                    │                    │
 │  Adjust filters        │                    │                    │
 │  (weapon, side, demo)  │                    │                    │
 │───────────────────────▶│                    │                    │
 │                        │  GetHeatmapData    │                    │
 │                        │  (json demoIDs +   │                    │
 │                        │   weapons + opts)  │                    │
 │                        │───────────────────▶│                    │
 │                        │                    │  json_each() join  │
 │                        │                    │  on game_events    │
 │                        │                    │  WHERE event=kill  │
 │                        │                    │  GROUP BY x, y     │
 │                        │                    │───────────────────▶│
 │                        │                    │  aggregated points │
 │                        │                    │◀───────────────────│
 │                        │  HeatmapPoint[]    │                    │
 │                        │◀───────────────────│                    │
 │  KDE renders on        │                    │                    │
 │  PixiJS canvas         │                    │                    │
 │◀───────────────────────│                    │                    │
```
