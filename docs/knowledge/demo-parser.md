# Demo Parser

**Library:** `github.com/markus-wa/demoinfocs-golang/v5`
**Related:** [[sqlite-wal]] · [[wails-bindings]] · [ADR-0015](../decisions/0015-streaming-parse-ingest-pipeline.md) · [ADR-0017](../decisions/0017-parser-defense-in-depth.md) · [ADR-0018](../decisions/0018-corrupt-entity-auto-retry.md) · [plans/p2-auth-demo-pipeline](../plans/p2-auth-demo-pipeline.md)

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

## Grenade trajectories (2026-05-07)

Adds a `GrenadeProjectileBounce` handler so the 2D viewer can curve in-flight grenade icons through their actual path instead of teleporting from throw to detonation.

### `GrenadeProjectileBounce` is the practical way to get bounce points

`Projectile.Trajectory` is populated over the projectile's lifetime but only fully readable on `GrenadeProjectileDestroy`, which would require holding live projectile references and snapshotting at destroy. The shape that fits the existing event-driven storage is to register `events.GrenadeProjectileBounce` directly and emit a per-bounce `GameEvent` carrying `entity_id` + `bounce_nr`. Each event lands in `game_events`; the frontend reassembles trajectories by `entity_id`.

### Four event types terminate a grenade trajectory

Pairing a throw with its endpoint requires checking all four: `grenade_detonate` (HE / Flashbang), `smoke_start`, `fire_start` (Molotov / Incendiary), and `decoy_start`. The viewer's `buildScheduled` indexes terminations by `entity_id` and skips orphaned throws (no termination ⇒ truncated demo).

### `entity_id` is a JSON number across the Wails boundary

`e.Projectile.Entity.ID()` returns Go `int`. After `json.Marshal` → DB TEXT → `json.Unmarshal` → Wails struct → JSON, it lands in TypeScript as a `number`, not a `string`. Frontend pairing code that does `typeof id === "string"` will silently never match — `entity_id`-keyed maps stay empty, and any duration that depends on the pairing falls back to its default. Use the `entityKey()` helper in `event-layer.ts` to normalize.

## Active-weapon ammo (2026-05-07)

`TickSnapshot.AmmoClip` / `AmmoReserve` populated from `player.ActiveWeapon().AmmoInMagazine()` / `AmmoReserve()`. Both `0` when the active item has no ammo concept (knife, C4) or when there is no active weapon.

### `AmmoReserve()` for grenades returns `held - 1`

Per the demoinfocs docs: "Returns CWeaponCSBase.m_iPrimaryReserveAmmoCount for most weapons and 'Owner.AmmoLeft[AmmoType] - 1' for grenades." A player whose active item is a single grenade therefore reads `clip=0, reserve=0`, indistinguishable from a knife at the parser level. The viewer's formatter (`frontend/src/lib/viewer/weapon-label.ts`) hides the ammo suffix when both values are zero, so the displayed result is just the weapon name — but if you ever need true grenade counts, prefer the `Inventory` slice over inferring from ammo.

## Player visibility / spotted mask (captured) (2026-05-12)

Captured via `events.PlayerSpottersChanged` in [`internal/demo/parser.go`](../../internal/demo/parser.go) (handler `handleSpottersChanged`). The demoinfocs event fires whenever `m_bSpottedByMask.0000` or `m_bSpottedByMask.0001` changes; for each `Spotted` player we re-derive the full spotter set by iterating `p.GameState().Participants().Playing()` and calling `spotted.IsSpottedBy(other)` (no `Spotters()` method exists on `common.Player`). Transitions are persisted to the `player_visibility` table (migration 019) as one row per `(spotted_steam, spotter_steam, state, tick)` — `state=1` is spotted_on, `state=0` is spotted_off.

### Filters (drop the event when any are true)

- warmup period (when `skipWarmup` is on);
- pre-match (`state.currentRound == 0`);
- freezetime (`tick < round.freeze_end_tick`);
- either side is a bot, nil, or has `SteamID64 == 0` (`shouldSkipPlayer`);
- either side is dead;
- either side is not on `TERRORIST` / `COUNTER_TERRORIST`.

### Debounce — defer-then-commit, 4 ticks

Each candidate transition is held as `pending`. If a flip-back arrives on the same pair within `visibilityDebounceTicks = 4` ticks, both are dropped (flicker rejection). Otherwise the row commits at its original tick, and the per-pair last-emitted state is updated. `RoundEnd` and parser teardown flush any still-pending rows unconditionally so a fight ending exactly at a round boundary is not lost. The per-round flush also clears all per-pair maps so visibility never crosses a round boundary.

### Volume

Measured on the reference fixture `testdata/demos/1.dem` (24 rounds, de_ancient, Faceit MR12): **534 visibility rows** for 7,476 events. Budget is 50 000 rows per demo; the parser hard-fails (sets `state.limitExceeded`) above 200 000 — at that point the fallback is run-length-window storage (see analysis §9.1).

### Downstream consumer

Phase 2 contact builder (`.claude/plans/timeline-contact-moments/phase-2-contact-builder.md`) reads `player_visibility` via `ListVisibilityForRound`. The table is *not* exposed via Wails bindings; the frontend never reads it directly. A pointer comment in [`types.go`](../../types.go) directs future contributors to `internal/demo.VisibilityChange` instead of re-introducing a frontend type.

### Pre-merge spike

`cmd/spike-spotted` (build with `-tags spike`) prints `tick / spotted / spotters` triples for any demo. Confirmed on the reference Faceit demo: entry-fight bursts of spotter additions at round start, no freezetime fires (the parser guard skips them), no per-frame spam for a single pair.

Cross-link: implementation plan at [`.claude/plans/timeline-contact-moments/phase-1/`](../../.claude/plans/timeline-contact-moments/phase-1/README.md).

## Streaming parse → ingest pipeline (2026-05-08)

`Parse(ctx, r)` now accepts a `WithTickSink(chan<- TickSnapshot)` option. Ticks flow into a bounded channel; `app.go parseDemo` runs parse + ingest concurrently under `errgroup.WithContext` with `DefaultTickSinkBuffer = 5000`. Peak tick-path heap dropped from ~120 MB to ~4 MB.

### Don't try to stream events too

Events remain accumulated in `state.events`. `dropKnifeRounds`, `pairShotsWithImpacts`, and `ExtractGrenadeLineups` all need the full forward-correlated list (e.g. shot pairing walks ticks while owning a `lastShotIdx[attackerSteamID]` map across the entire match). A previous attempt to stream events broke knife-round detection silently — only ticks stream.

### `IngestStream` is the canonical path; `Ingest` is a thin wrapper

`IngestStream(ctx, demoID, ticksIn)` owns the single ingest tx + ctx-cancel + recover. The slice-based `Ingest(demoID, ticks)` exists only as a fan-into-channel adapter so the batch logic lives in one place.

### Don't drop the heap watchdog

The watchdog stayed in. Streaming ticks bounds the **tick** path, but `demoinfocs`'s internal protobuf/entity-table state still grows on corrupt demos and that's where the watchdog earns its keep. Its ceiling is no longer a static 4 GiB — see "Windows OOM safeguards (2026-05-08)" for the per-host sizing.

### Cache `strconv.FormatUint(SteamID64)` per player

`parseState.steamID(p *common.Player)` lazily fills a `map[uint64]string` on first reference. Replaces 13 call sites that each allocated ~24 B per call × 10 players × ticks.

### Typed event extras, not `map[string]interface{}`

Per-kind structs in `internal/demo/extras.go` (`KillExtra`, `WeaponFireExtra`, `PlayerHurtExtra`, `GrenadeThrowExtra`, etc.) implement an `EventExtra` marker. Parser allocates one pointer-to-struct per event; `marshalExtraData` in `events.go` JSON-encodes at the ingest boundary. The wire format to the frontend is unchanged — this is purely an allocation reduction in the hot path.

## Windows OOM safeguards (2026-05-08)

The static `maxHeapBytes = 4 GiB` watchdog plus an unset `GOMEMLIMIT` was OOMing 16 GB Windows hosts. WebView2 (~1–2 GB) + OS + drivers + AV left no room for a Go heap allowed to grow to 2× live set, and the 4 GiB ceiling only fired *after* the OS was paging.

### Heap budgets are sized from host RAM at startup

`internal/sysinfo.RecommendedHeapLimits(totalRAM)` returns `(GOMEMLIMIT=12.5%, KillSwitch=18.75%)` of total RAM, clamped to `[1 GiB, 4 GiB]`. On a 16 GB Windows host that's **2 GiB soft / 3 GiB hard**, leaving ~80% of RAM for WebView2 and the OS. `main.go configureMemoryLimits()` calls `debug.SetMemoryLimit()` (skipped if `GOMEMLIMIT` env is already set) and stashes the kill-switch in `heapLimits.KillSwitch` for `NewApp` to plumb into the parser.

### `WithHeapLimit(bytes)` overrides the default ceiling

`NewDemoParser(demo.WithHeapLimit(a.parserHeapLimit))`. `defaultMaxHeapBytes = 4 GiB` is kept as the fallback for direct callers (tests, future CLI tools) so they don't have to wire sysinfo themselves. The watchdog error message includes the observed and limit MiB — when a corrupt demo trips the cap, `errors.txt` records *which* budget was in effect, so a future re-tune doesn't make stale logs ambiguous.

### `runtime.GC()` between parse and ingest

After `parser.Parse()` returns, the demoinfocs parser is closed but its internal entity tables / packet buffers are unreferenced and not yet collected. Running `runtime.GC()` before `IngestRounds`/`IngestGameEvents` prevents that transient state from coexisting with the ingest tx for the seconds it runs — particularly important on Windows where the runtime is slower to scavenge. After events commit, `result.Events`/`Lineups`/`Rounds` are nil-ed so the ~125 MB worst-case event slice can be reclaimed before "complete" emits.

### Cross-platform RAM detection without a heavy dep

`golang.org/x/sys/windows` v0.42.0 doesn't expose `GlobalMemoryStatusEx`, so the Windows path resolves it through `windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")` with a hand-declared `MEMORYSTATUSEX` struct. Darwin uses `unix.SysctlUint64("hw.memsize")`; Linux uses `unix.Sysinfo`. Detection failure returns `(0, err)`; callers fall back to the conservative floor (1 GiB soft / 1.5 GiB hard) — never fatal.

## Heap watchdog v2: independent goroutine (2026-05-08)

> Decision rationale and rejected alternatives: [ADR-0017](../decisions/0017-parser-defense-in-depth.md).

The static heap ceiling from the previous section still didn't fire on a pathological 325 MB demo that drove the working set to 13 GB. Two structural holes the in-handler heartbeat couldn't close:

### The FrameDone heartbeat is blind to pre-frame work

`p.RegisterEventHandler(func(_ events.FrameDone) {...})` only runs after the demoinfocs library has decoded its first frame. String tables, entity baselines, DataTable decoding, and the SendTable bootstrap all happen *before* the first FrameDone, and that's exactly the path where corrupt demos blow up — the heartbeat never fires, the watchdog can't trip, and the heap is gone before the user sees anything. Fix: `internal/demo/heap_watchdog.go` spawns an independent goroutine in `Parse()` that polls `runtime.ReadMemStats` every 500 ms regardless of dispatcher state. On trip it dumps a pprof heap profile to `{AppData}/oversite/profiles/`, calls a stop callback (sets `state.limitExceeded`, `Cancel()`s the parser), and stops itself. The FrameDone heartbeat stays as belt-and-braces for healthy parses — it logs every 10K frames and doubles up on the limit check.

### `IgnorePacketEntitiesPanic = true` is a footgun

Set unconditionally in commit `2329a1d` to stop crashes on certain Windows POV demos. With it on, demoinfocs swallows "unable to find existing entity" panics and continues — which on a pathological demo means an unbounded internal accumulation loop the watchdog couldn't catch fast enough on Windows. The fix in `2329a1d` traded a visible crash for a silent 13 GB blow-up. Now opt-in via `WithIgnoreEntityPanics(bool)` (default **false**); the panic recovery in `Parse` checks the message and returns `ErrCorruptEntityTable` so the import fails fast with a clear user-facing error. Users who *need* partial-parse tolerance can flip the bool via the Settings binding (`SetTolerateEntityErrors`).

### Auto-retry on `ErrCorruptEntityTable` (2026-05-08)

> Decision rationale and rejected alternatives: [ADR-0018](../decisions/0018-corrupt-entity-auto-retry.md).

Default-off-with-Settings-opt-in turned out to dead-end users — `SetTolerateEntityErrors` is bound but no UI surfaces it, so a v0.1.8 Windows user with a corrupt-entity demo had no path forward. `app.go parseDemo` now reruns the parse once with `WithIgnoreEntityPanics(true)` after a first-attempt failure with `ErrCorruptEntityTable`. The streaming pipeline was extracted into `runParsePipeline(demoID, f, tolerateEntityErrors, emitProgress)` so both attempts share the errgroup logic.

Retry is safe because `IngestStream` wraps everything in a single tx with `defer tx.Rollback()` (`ingest.go:100`) and the next call's `DeleteTickDataByDemoID` (`ingest.go:103`) wipes any rows that did commit. The file is rewound between attempts with `f.Seek(0, io.SeekStart)`. The retry path emits `slog.Warn("parseDemo: retrying with entity-panic tolerance", first_err=…)` so `errors.txt` records when tolerance kicked in.

The `SetTolerateEntityErrors` binding remains — flipping it on skips the first (likely-failing) attempt, which is the right shape for re-importing a known-corrupt batch.

### Windows holds onto pages even after Go GC

`runtime.GC()` between phases collects unreachable heap, but on Windows the runtime is slow to madvise/decommit those pages back to the OS — Task Manager's "Memory (active private working set)" stays high even after the Go heap shrinks. `debug.FreeOSMemory()` after `result.Events = nil` forces a full GC + scavenge and is the only reliable way to drop the working set post-parse. Cost is ~50–200 ms once per import; mostly a no-op on macOS/Linux where the runtime is already aggressive. Watchdog also trips when `WorkingSet > 1.5× heap limit` to catch the case where Go thinks the heap is fine but the OS-visible memory has run away.

### `golang.org/x/sys/windows` doesn't expose `GetProcessMemoryInfo`

Same gap as `GlobalMemoryStatusEx`. Wire it via `windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")` with a hand-declared `PROCESS_MEMORY_COUNTERS_EX` struct (see `internal/sysinfo/procmem_windows.go`). `SIZE_T` is `uintptr`; set `counters.CB = sizeof(struct)` before the call. Non-Windows (`procmem_other.go`) returns zeros so the watchdog falls back to `MemStats`.

### `maxGameEvent` cap dropped 500K → 100K

The 500K cap was set when entity-panic recovery let runaway demos accumulate millions of events. With the goroutine watchdog catching pathological cases earlier, 100K is still 2–3× the worst legitimate match and fails sooner on corrupt demos.

## File-close-on-retry + demoinfocs v5.2.0 (2026-05-08)

### `gobitread.BitReader.Close` closes the underlying reader

`gobitread/bitread.go:99` type-asserts `BitReader.underlying` to `io.ReadCloser` and calls `Close()` on it if the assertion succeeds. `*os.File` satisfies that interface, so our deferred `p.Close()` in `internal/demo/parser.go` was closing the caller's file. The auto-retry path in `app.go parseDemo` then failed `f.Seek(0, io.SeekStart)` with `"file already closed"` — every Windows retry on `ErrCorruptEntityTable` was a silent no-op. Fix: wrap the reader before handing it to demoinfocs:

```go
p := demoinfocs.NewParserWithConfig(struct{ io.Reader }{r}, config)
```

`TestParse_DoesNotCloseReader` (in `parser_test.go`) calls `Parse` with a close-tracking `io.ReadCloser` and asserts `Close` is never invoked. If a future change strips the wrapper, the test fails immediately.

### Upgraded `demoinfocs-golang/v5` to v5.2.0

Bumped from `v5.1.2`. Three releases of relevant fixes:

| Version | Key changes |
|---------|-------------|
| v5.1.3 | `bindBomb` nil-pointer fix; broadcast parsing fix; protobuf refresh |
| v5.1.4 | **AnimGraph 2 demo support** — newer CS2 demos started failing on v5.1.2 with "unable to find existing entity"; this was the most likely actual root cause of the recent Windows import reports |
| v5.2.0 | `CGlobalSymbol` decode crash fix; infinite-recursion fix in `getThrownGrenade` (ControlledBot circular reference — possibly explains some heap-watchdog trips); **~30-35% parse speedup**; `PlayerHurt` world-vs-bomb damage classification fix; new `Player.PositionEyes()` helper |

Public API unchanged for our usage. The v5.2.0 fixes don't replace the watchdog/auto-retry/heap-budget defenses — they reduce how often those defenses are triggered.
