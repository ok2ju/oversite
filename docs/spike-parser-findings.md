# Demo Parser Spike Findings

**Date:** 2026-04-15
**Branch:** main (spike code in `internal/demo/` + `cmd/spike-parser/`)
**Library:** `github.com/markus-wa/demoinfocs-golang/v5@v5.1.2`

---

## 1. Performance

| Demo | Size (decompressed) | Map | Rounds | OT Rounds | Parse Time | Heap Delta | Sys Delta | Ticks | Lineups | Result |
|------|---------------------|-----|--------|-----------|------------|------------|-----------|-------|---------|--------|
| 1-591a... | 862.3 MB | de_ancient | 54 | 30 | 6.5s | +118 MB | +495 MB | 872,495 | 859 | PASS |
| 1-9e28... | 394.2 MB | de_ancient | 25 | 1 | 3.1s | -68 MB* | +0 MB | 397,405 | 369 | PASS |
| 1-a66b... | 454.3 MB | de_dust2 | 30 | 6 | 3.8s | +12 MB | +0 MB | 484,284 | 383 | PASS |

*Negative heap delta means GC reclaimed more memory than the parse allocated (previous demo's allocations freed).

**Targets met:** All demos parse in <10s and use <500 MB heap. Even the 862 MB / 54-round overtime marathon completes in 6.5s with only +118 MB heap. The streaming parser confirmed -- memory is proportional to in-flight state, not file size.

---

## 2. CS2 Edge Cases Encountered

| Edge Case | Observed? | Details |
|-----------|-----------|---------|
| Warmup rounds | Yes | Skipped correctly (default `skipWarmup=true`). Round numbering starts at 1 post-warmup. |
| Bot presence | No | No SteamID=0 snapshots in these Faceit demos (bots not present in competitive). |
| Overtime | Yes | All 3 demos had overtime. 1 OT round, 6 OT rounds, and 30 OT rounds respectively. `isOvertime(roundNum > 24)` works correctly. |
| Score consistency | OK | No score regressions detected. Monotonic progression confirmed. |
| Truncated demo | No | All 3 demos parsed completely without `ErrUnexpectedEndOfDemo`. |
| Zero-position players | No | No alive players at (0, 0, 0) detected. |
| Orphaned grenade throws | **Yes** | 25% orphan rate (124/493, 100/483, 282/1141). See Section 3. |
| World kills (nil-killer) | Yes | 1 instance in demo 2 (fall damage or world kill). |
| Round numbering gaps | No | Sequential round numbers, no gaps. |

---

## 3. Incendiary/Molotov Gap (Primary Finding)

**Problem:** ~25% of grenade throws have no matching detonation event, producing orphaned lineups.

**Root cause:** The parser (`parser.go`) does not register a handler for incendiary grenade / molotov events. The `registerHandlers` function handles:
- `events.HeExplode` -> `grenade_detonate`
- `events.FlashExplode` -> `grenade_detonate`
- `events.SmokeStart` -> `smoke_start`
- `events.SmokeExpired` -> `smoke_expired`
- `events.DecoyStart` -> `decoy_start`

**Missing:** No handler for `events.FireGrenadeStart` or equivalent incendiary/molotov detonation event from `demoinfocs-golang`.

The grenade extractor (`grenade_extractor.go`) only correlates throws with these `detonationTypes`:
```go
var detonationTypes = map[string]bool{
    "grenade_detonate": true,
    "smoke_start":      true,
    "decoy_start":      true,
}
```

**Impact:** Incendiary and molotov throws appear as `grenade_throw` events but never get a matching detonation, so `ExtractGrenadeLineups()` cannot produce lineups for them. This affects heatmap accuracy and strategy features.

**Fix for P2-T06:**
1. Add `events.FireGrenadeStart` handler in `parser.go` emitting `"fire_start"` event type
2. Add `"fire_start": true` to `detonationTypes` in `grenade_extractor.go`
3. Minor contributor: `DecoyExpired` events (decoy destruction) are also unmatched but this is a small number

---

## 4. API Compatibility

**demoinfocs-golang v5.1.2 is fully compatible.** Zero code changes were needed to compile and run the web-era parser (`backend/internal/demo/parser.go`, 642 lines) at the root module level. All event types, handler signatures, and `common.Player` / `GameState` APIs work as documented.

Transitive dependencies resolved cleanly: `golang/geo`, `golang/snappy`, `markus-wa/gobitread`, `markus-wa/godispatch`, `markus-wa/quickhull-go`, `oklog/ulid`, `markus-wa/go-unassert`.

---

## 5. MaxUploadSize

**Problem:** `validate.go` defines `MaxUploadSize = 500 << 20` (500 MB). This is a web-upload safety limit that doesn't apply to local desktop file parsing. Faceit `.dem.zst` files decompress to 400-860+ MB routinely.

**Fix for P2-T05:** Raise `MaxUploadSize` to 1 GB (`1 << 30`), or remove size validation entirely for the desktop import path (keep it only if we add a web upload endpoint later).

---

## 6. Recommendations for P2-T06

### Keep from backend parser (no changes needed)
- All event handler registrations (kills, hurt, grenades, bombs, rounds, warmup, overtime)
- `DemoParser` struct with `Option` functional pattern
- `ParseResult` / `MatchHeader` / `RoundData` / `TickSnapshot` / `GameEvent` types
- `shouldSampleTick`, `isOvertime`, `shouldSkipPlayer`, `teamSideString` helpers
- `CalculatePlayerRoundStats()` from `stats.go`
- `resolveCallout()` and `mapCallouts` from `callouts.go`
- Panic recovery and truncated demo handling

### Change during implementation
1. **Add incendiary/molotov handler** -- register `events.FireGrenadeStart` (or check demoinfocs docs for the exact event name), emit `"fire_start"` type
2. **Add `"fire_start"` to `detonationTypes`** in grenade extractor
3. **Raise or remove `MaxUploadSize`** in `validate.go`
4. **Add Wails progress events** -- `ProgressFunc` callback for UI feedback during parse
5. **Consider `.dem.zst` native support** -- Faceit distributes compressed demos; `klauspost/compress/zstd` can decompress in-process without requiring the `zstd` CLI

### Risk areas
- None identified. Parser is stable, performant, and the library API is unchanged.
