# Demo Parser

**Library:** `github.com/markus-wa/demoinfocs-golang/v5`
**Related:** [[sqlite-wal]] · [[wails-bindings]] · [plans/p2-auth-demo-pipeline](../plans/p2-auth-demo-pipeline.md)

> Note: this page describes the parser as it runs against MR12 competitive CS2 demos (24 regulation rounds + optional overtime, no bots). Casual / bot-laden demos are out of scope.

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
| Bot presence | N/A | No bots in MR12 competitive demos |
| Overtime | Yes | Sourced from `gs.OvertimeCount() > 0` at `RoundEnd`; survives `dropKnifeRounds` renumbering. See "Parser quality fixes (2026-05-07)". |
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

- **Raise or remove `MaxUploadSize`** in `validate.go` (currently 500 MB, meant for web uploads). Real `.dem.zst` files decompress to 400–860+ MB.
- **Consider native `.dem.zst` support** via `klauspost/compress/zstd` (in-process, no CLI dependency).
- **Add progress events** for Wails UI feedback — `ProgressFunc` callback exposed via `runtime.EventsEmit`.

## What NOT to change from the web-era parser

API and helpers are solid — the library itself didn't change between web and desktop:
- Event handler registrations (kills, hurt, grenades, bombs, rounds)
- `DemoParser` struct with `Option` functional pattern
- `ParseResult` / `MatchHeader` / `RoundData` / `TickSnapshot` / `GameEvent` types
- `shouldSampleTick`, `shouldSkipPlayer`, `teamSideString` helpers (`isOvertime` was removed; see "Parser quality fixes (2026-05-07)")
- `CalculatePlayerRoundStats()` from `stats.go` — signature unchanged, but it now seeds players from `RoundData.Roster` before layering events. Roster comes from the parser, not from the stats layer.
- Callouts resolution from `callouts.go`
- Panic recovery and truncated-demo handling

## Parser quality fixes (2026-05-07)

Six gotchas surfaced by a deep review of the v5 parser; all fixed in `internal/demo/parser.go` + `stats.go` (no public API change).

### Don't trust `WinnerState.Score()` at `RoundEnd`

The library docs say it's not up-to-date and recommend a `+1` workaround. v5's actual behavior contradicts the docs — `ScoreUpdated` fires *before* `RoundEnd` and updates the team state in-place, so reading `WinnerState.Score()` here double-counts. **Don't read it.** Source the per-team score from `ScoreUpdated` into `state.ctScore` / `state.tScore` and consume those at `RoundEnd`. If `ScoreUpdated` is missing for a round (rare; malformed demos), increment from `e.Winner` and `slog.Warn` so the demo surfaces.

### Use `IsWarmupPeriod()` everywhere — never cache the warmup flag

The cached `IsWarmupPeriodChanged` value lags the live state by one event dispatch. If `IsWarmupPeriodChanged(false)` arrives a frame after `RoundStart`, `state.currentRound` stays at 0 while subsequent kill/hurt events fire — those events ship with `RoundNumber=0` and break the FK on `game_events.round_id`. Gate every handler (`RoundStart`, `RoundEnd`, `FrameDone`, kill/hurt/grenade/bomb) on `p.GameState().IsWarmupPeriod()`.

### Filter `IsAlive` before reading `Weapons()`

`Participants().Playing()` returns players regardless of liveness, but a dead player has an empty `Weapons()` slice. The freeze-end inventory snapshot must filter `IsAlive` upstream — otherwise `isKnifeRoundByInventory` produces false negatives for real knife rounds whenever even one player is dead at freeze-end.

### Knife-round inventory needs a minimum-sample guard

`isKnifeRoundByInventory` now requires `len(inventories) >= 8`. Without it, a 1–2 player frame during a reconnect can flag a real eco round as a knife round. The C4 exception was also dropped — Faceit knife configs zero out `mp_t_default_secondary`, so T-side normally won't even have a pistol on a knife round. If a future demo format violates this, re-introduce a typed exception with the demo evidence.

### Detect overtime from `OvertimeCount()`, not the round number

The previous `isOvertime(roundNum > 24)` hardcoded MR12 and broke under MR15, Wingman, custom `mp_maxrounds`, and after `dropKnifeRounds` renumbered rounds. Capture `gs.OvertimeCount() > 0` at `RoundEnd`; the flag survives `dropKnifeRounds` renumbering unchanged. `parseState.ensureFormat(p)` reads `mp_maxrounds` lazily from convars for a warn-only cross-check; convars are streamed during the demo, so reading too early returns empty — call `ensureFormat` from `RoundEnd`.

### Seed `player_rounds` from a per-round roster

`stats.CalculatePlayerRoundStats` previously registered players only when it saw a kill/hurt event for them — passive players (no kills, no damage, no deaths) got no `player_rounds` row, and the viewer's roster lookup fell back to `steam_id.slice(0, 10)`, surfacing as numeric nicknames. The parser now snapshots `(SteamID, Name, TeamSide)` for every alive non-bot player at freeze-end into `RoundData.Roster`, and `calculateRound` seeds the player map from it before layering kill/hurt events. Late joiners not in the roster still register via the existing `getPlayer` fallback.

### Pin the demoinfocs minor version

The score-read fragility above depends on v5's `ScoreUpdated` → `RoundEnd` ordering. Pin the minor version in `go.mod` and document the assumed firing order near the `ScoreUpdated` handler in `parser.go`. A patch release bumping that order would silently break score capture.

## Shot tracers (2026-05-07)

Adds `weapon_fire` events plus a post-processing pass that pairs each shot with its `player_hurt` to give the 2D viewer exact impact endpoints. Both live in `parser.go` and the new `internal/demo/shot_impacts.go`.

### `WeaponFire` fires for grenades and knives — filter by `Equipment.Class()`

`p.RegisterEventHandler(func(e events.WeaponFire) {...})` fires for **every** weapon use, including grenade throws and knife slashes. The handler must filter to firearm classes only:

```go
switch e.Weapon.Class() {
case common.EqClassPistols, common.EqClassSMG, common.EqClassHeavy, common.EqClassRifle:
default:
    return
}
```

Without this filter, smoke throws and decoy tosses would emit `weapon_fire` events and show up as tracers in the viewer.

### `PlayerHurt.X/Y/Z` were historically zero — now populated

Until this change, the `player_hurt` handler emitted `GameEvent` with `X = Y = Z = 0`. Nothing read those fields, so the bug was invisible. The shot-impact pairing pass needs the victim's position, so the handler now writes `e.Player.Position()` into the event. Anything new that consumes `player_hurt` should expect populated coordinates.

### Bullet impact data: what demoinfocs v5.1.2 exposes

| Event | Endpoint data | Limitations |
|-------|---------------|-------------|
| `PlayerHurt` | victim `Player.Position()` | player hits only |
| `BulletDamage` | `Distance` + `DamageDirX/Y/Z` | CS2 demos post-2024-07-22 only; not always present |
| `Kill` | victim `Position()` (already used) | terminal hit only |
| `bullet_impact` user message | wall/world impacts | **not surfaced as a Go event** |

CS2's `bullet_impact` user message (which carries actual wall/object impact coordinates) is in the demo format but not parsed by demoinfocs v5. Surfacing it would require a raw user-message handler at the protobuf level — a non-trivial lift, deferred. Until then, wall hits and pure misses have no endpoint data; the viewer falls back to a fixed-length directional ray.

### Pairing strategy: most-recent-prior, consume on pair

`pairShotsWithImpacts` (in `shot_impacts.go`) walks events in tick order and maintains `lastShotIdx[attackerSteamID]`. Each `weapon_fire` overwrites the entry; each matching `player_hurt` consumes it. Trade-offs:

- ✅ Spray with mixed hits/misses works — each new shot replaces the lastShot record, so a `player_hurt` always pairs with the closest prior shot.
- ✅ Cross-attacker isolation is automatic.
- ❌ Wallbangs (one shot, multiple `player_hurt` events for different victims) only pair the first hurt; subsequent ones are dropped.
- ❌ Window is 16 ticks (~250ms @ 64Hz) — long enough for any in-map bullet flight, short enough that a stale record from an old shot can't pair with an unrelated `player_hurt` after a reload.
