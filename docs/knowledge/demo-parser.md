# Demo Parser

**Library:** `github.com/markus-wa/demoinfocs-golang/v5`
**Related:** [[sqlite-wal]] · [[wails-bindings]] · [plans/p2-auth-demo-pipeline](../plans/p2-auth-demo-pipeline.md)

## Original spike (2026-04-15, v5.1.2)

### Performance

| Demo size | Map | Rounds | Parse time | Heap Δ | Ticks | Lineups |
|-----------|-----|--------|------------|--------|-------|---------|
| 862 MB | de_ancient | 54 (30 OT) | 6.5s | +118 MB | 872k | 859 |
| 394 MB | de_ancient | 25 (1 OT) | 3.1s | −68 MB* | 397k | 369 |
| 454 MB | de_dust2 | 30 (6 OT) | 3.8s | +12 MB | 484k | 383 |

*Negative delta: GC reclaimed more memory than the parse allocated. The parser streams — memory is proportional to in-flight state, not file size.

All three demos met the targets: < 10s parse, < 500 MB heap. Even an 862 MB 54-round marathon finishes in 6.5s.

### Edge cases observed

| Case | Handled? | Notes |
|------|----------|-------|
| Warmup rounds | Yes | Default `skipWarmup=true`; round numbering starts at 1 post-warmup |
| Bot presence | N/A | No bots in Faceit competitive demos |
| Overtime | Yes | `isOvertime(roundNum > 24)` works correctly |
| Truncated demo | N/A | All three parsed cleanly; no `ErrUnexpectedEndOfDemo` |
| Orphaned grenade throws | **No — bug** | ~25% orphan rate. See below. |
| World kills (nil-killer) | Yes | Fall damage / world kills handled |

## Known bug: incendiary/molotov gap

**25% of grenade throws have no matching detonation.** Root cause: `parser.go` registers handlers for `HeExplode`, `FlashExplode`, `SmokeStart/Expired`, `DecoyStart` — but **not** `FireGrenadeStart` (molotov/incendiary). Without that, the grenade extractor's `detonationTypes` map can't match throws to detonations for fires.

**Fix (P2-T06):**
1. Register `events.FireGrenadeStart` in `parser.go`, emit `"fire_start"` event type.
2. Add `"fire_start": true` to `detonationTypes` in `grenade_extractor.go`.
3. `DecoyExpired` is also unmatched, but the volume is small — lower priority.

## Other spike recommendations

- **Raise or remove `MaxUploadSize`** in `validate.go` (currently 500 MB, meant for web uploads). Faceit `.dem.zst` decompresses to 400–860+ MB.
- **Consider native `.dem.zst` support** via `klauspost/compress/zstd` (in-process, no CLI dependency).
- **Add progress events** for Wails UI feedback — `ProgressFunc` callback exposed via `runtime.EventsEmit`.

## What NOT to change from the web-era parser

API and helpers are solid — the library itself didn't change between web and desktop:
- Event handler registrations (kills, hurt, grenades, bombs, rounds)
- `DemoParser` struct with `Option` functional pattern
- `ParseResult` / `MatchHeader` / `RoundData` / `TickSnapshot` / `GameEvent` types
- `shouldSampleTick`, `isOvertime`, `shouldSkipPlayer`, `teamSideString` helpers
- `CalculatePlayerRoundStats()` from `stats.go`
- Callouts resolution from `callouts.go`
- Panic recovery and truncated-demo handling
