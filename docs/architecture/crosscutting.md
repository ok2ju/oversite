# Architecture — Cross-Cutting Concerns

> **Siblings:** [overview](overview.md) · [structure](structure.md) · [components](components.md) · [data-flows](data-flows.md) · [wails-bindings](wails-bindings.md) · [database](database.md) · [testing](testing.md)

---

## Cross-Cutting Concerns

### Error Handling

| Layer | Strategy |
|-------|----------|
| Go bindings | Return `error` as second return value; Wails converts to rejected Promise |
| Frontend | `try/catch` on binding calls; toast notifications for user-facing errors |
| Demo parser | Structured errors with context (e.g., `ParseError{Phase: "ticks", Tick: 42000, Err: ...}`) |
| SQLite | Wrap in transaction; rollback on error; return descriptive error to caller |

### Logging

Implemented in `internal/logging/` (see [ADR-0013](../decisions/0013-logging.md)). Files live under `{AppDataDir}/logs/`:

| File | When | Contents |
|------|------|----------|
| `errors.txt` | Always | `slog` WARN+ records from all Go packages, plus a bridge that captures any remaining `log.Printf` calls |
| `network.txt` | Dev builds only (`runtime.Environment(ctx).BuildType == "dev"`) | Full HTTP request/response dumps from the Faceit client, demo download client, and OAuth token exchange |

- Both files rotate at 5MB with 3 backups via `lumberjack.v2` (plain `.txt`, no gzip).
- `logging.Init(dir)` runs once from `main.go` before `wails.Run`; `logging.Close()` runs from `App.Shutdown`.
- The dev network transport is wired into HTTP clients in `App.Startup`; the `OVERSITE_DEBUG_HTTP` env flag used by the old `debug_transport` is retired.
- Frontend: `console.error` for binding failures; no separate log file (developers use the Wails DevTools).

### Configuration

User preferences stored in `config.json` in the app data directory:

```json
{
  "theme": "dark",
  "watchFolder": "/path/to/cs2/demos",
  "autoFetchFaceit": true,
  "viewerDefaults": {
    "playbackSpeed": 1.0,
    "showHealthBars": true,
    "showMinimap": true
  }
}
```

Loaded at startup by Go backend; exposed to frontend via `GetConfig`/`SetConfig` bindings.

### SQLite Data Integrity & Recovery

The SQLite database is the sole store for all parsed demo data. Since re-parsing demos is possible but expensive (< 10s per demo), data integrity matters.

**WAL checkpoint**: SQLite WAL mode is used for concurrent reads. The WAL file is checkpointed automatically by SQLite. On graceful shutdown (`app.Shutdown()`), force a WAL checkpoint via `PRAGMA wal_checkpoint(TRUNCATE)` to ensure the database file is self-contained.

**Pre-migration backup**: Before running `golang-migrate` up migrations, copy the database file to `oversite.db.bak`. If migration fails, the backup allows manual recovery. The backup is overwritten on each migration run (only the most recent is kept).

**Corruption detection**: On startup, run `PRAGMA integrity_check` (fast on small databases). If corruption is detected:
1. Log the error with full details
2. Show a modal to the user: "Database may be corrupted. You can re-import your demos to rebuild."
3. Offer to reset the database (delete and recreate with fresh migrations)
4. Demo `.dem` source files are stored by reference and are not affected

**Transaction discipline**: All multi-row inserts (tick data, events, rounds) are wrapped in explicit transactions. A parse failure mid-way rolls back the entire transaction — no partial demo data in the database.

**Recovery path**: Since demos are stored as external `.dem` files (not copied into the database), the worst-case recovery is: delete `oversite.db`, restart the app (migrations recreate schema), re-import demos. Faceit match data can be re-synced from the API.

### Coordinate Calibration

Each CS2 map has calibration data mapping game world-space to radar image pixel-space. Stored in the frontend as TypeScript constants:

```typescript
// src/lib/maps/calibration.ts
export const MAP_CALIBRATION = {
  de_dust2:  { originX: -2476, originY: 3239, scale: 4.4 },
  de_mirage: { originX: -3230, originY: 1713, scale: 5.0 },
  // ...
};

// Formula: pixelX = (worldX - originX) / scale
//          pixelY = (originY - worldY) / scale
```

See the knowledge wiki entry on [coordinate calibration](../knowledge/coordinate-calibration.md) for implementation notes.

### Auto-Update

- On startup, check for new version via HTTPS to a releases endpoint (GitHub Releases API or custom)
- Show non-intrusive notification if update available
- User-initiated download and install (no silent auto-update)
- Version check skippable in settings
