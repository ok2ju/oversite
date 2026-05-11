# Project Log

Append-only chronological record of notable project activity. New entries at the top. Maintained via the `/ingest-session` slash command.

Format: `YYYY-MM-DD — <summary>` with links to affected pages.

---

## 2026-05-11 — Duel-scoped mistakes (slice 13)

New `Duel` analyzer entity reconstructs directed attacker→victim engagements from the merged `weapon_fire + player_hurt + kill` stream — hit-anchored target via `WeaponFireExtra.HitVictimSteamID`, falling back to cone enumeration (15° / 2200u / 2s activity). Engagement-class mistakes attach via the new `analysis_mistakes.duel_id` FK; `eco_misbuy` / `he_damage` stay duel-less. `analysis_duels` table (migration 018) carries directional outcome (`won` / `inconclusive` / `won_then_traded`) and a self-referential `mutual_duel_id` for crossfire pairs.

`Run` signature changed: `([]Mistake, []Duel, error)`. Persistence inserts duels first, captures rowids in a local→DB map, then writes mistakes with resolved `duel_id` — mutual links backfilled in a second-pass UPDATE since both peers need rowids before they can reference each other. `AnalysisVersion` 1→2 triggers the existing "missing" status path so old demos surface the Recompute CTA.

Frontend: new `DuelsLane` on round-timeline renders bands tinted by perspective (sky-blue when the selected player is attacker, rose-red when victim) with outcome glyph + severity dots for inline mistakes; `useDuelTimeline` hook mirrors `useMistakeTimeline`; `mistake-list.tsx` gains a `Duel` chip per attributed row plus the new `PatternsSection` for cross-duel signals.

Refs: [[knowledge/wails-bindings]], [[knowledge/sqlc-workflow]], [[knowledge/migrations]], [[knowledge/testing]] (all updated), [[decisions/0019-duel-as-first-class-entity]] (new ADR).

## 2026-05-08 — File-close-on-retry fix + demoinfocs v5.1.2 → v5.2.0

The corrupt-entity auto-retry added earlier today never actually retried on Windows: `gobitread.BitReader.Close` type-asserts the reader to `io.ReadCloser` and closes our `*os.File`, so the second attempt's `f.Seek(0, ...)` returned "file already closed". Wrap the reader in `struct{ io.Reader }{r}` inside `Parse` to hide `Close` from demoinfocs; new `TestParse_DoesNotCloseReader` pins the contract.

Bumped `demoinfocs-golang/v5` to `v5.2.0` — pulls in v5.1.4 AnimGraph 2 demo support (likely the actual cause of the recent Windows import failures, per a user report after upgrade), v5.1.3 `bindBomb` nil-deref fix, plus v5.2.0 `CGlobalSymbol` crash fix, `getThrownGrenade` infinite-recursion fix, and ~30-35% parse speedup. No public API breaks.

Refs: [[knowledge/demo-parser]] (updated).

## 2026-05-08 — Auto-retry on ErrCorruptEntityTable

Follow-up to the entity-panic opt-in earlier today: a Windows v0.1.8 user hit `ErrCorruptEntityTable` on a corrupt-entity demo (`unable to find existing entity 1647731180`) with no way forward — the `SetTolerateEntityErrors` binding exists in `frontend/wailsjs/go/main/App.d.ts` but no UI calls it. `app.go parseDemo` now auto-retries once with `WithIgnoreEntityPanics(true)` after a first attempt fails with `ErrCorruptEntityTable`; the heap watchdog and tick/event caps still backstop the runaway-memory case the flag was kept off to avoid.

The streaming parse+ingest pipeline (was inline in `parseDemo`) is extracted into `runParsePipeline(demoID, f, tolerateEntityErrors, emitProgress)`. Retry is safe because `IngestStream` wraps everything in a single tx with `defer tx.Rollback()` (`ingest.go:100`) and the next call begins with `DeleteTickDataByDemoID` (`ingest.go:103`) — partial rows from the failed first attempt do not survive. The file is rewound with `f.Seek(0, io.SeekStart)`. A `slog.Warn("parseDemo: retrying with entity-panic tolerance", ...)` line records when the retry fires so it's traceable in `errors.txt`.

Refs: [[knowledge/demo-parser]] (updated), [[decisions/0018-corrupt-entity-auto-retry]] (new ADR documenting the retry layer; builds on ADR-0017).

## 2026-05-08 — Independent heap watchdog + entity-panic opt-in

Follow-up to the Windows 16 GB OOM safeguards: the static watchdog still missed the failure mode on a pathological 325 MB demo that drove the working set to 13 GB before the FrameDone heartbeat could fire. Two structural changes:

- **`internal/demo/heap_watchdog.go`** (new): independent goroutine polls `runtime.ReadMemStats` every 500 ms outside the demoinfocs FrameDone handler, so the kill-switch is enforced even while the parser is stuck in pre-frame work (string tables, entity baselines, DataTable decode). On trip, dumps a pprof heap profile to `{AppData}/oversite/profiles/heap-{demoID}-{ts}.pprof`, sets `state.limitExceeded`, and `Cancel()`s the parser. Soft-warning at 50% of the limit fires once. Windows also trips when `WorkingSet > 1.5× limit` (the Go runtime is slow to madvise/decommit pages back to the OS there). In-handler heartbeat retained as belt-and-braces for healthy parses.
- **`internal/demo/parser.go`**: `IgnorePacketEntitiesPanic` is now opt-in via new `WithIgnoreEntityPanics(bool)` option (default **false**). The previous unconditional `true` traded a visible crash for an unbounded internal accumulation loop on corrupt demos. "unable to find existing entity" panics now surface as new `ErrCorruptEntityTable` with a user-facing message. The `maxGameEvent` cap drops 500K → 100K — still 2–3× the worst legitimate case, but fails earlier on corrupt demos.
- **`internal/sysinfo/procmem*.go`** (new): `ProcessMemory()` returns OS-reported `WorkingSetSize`/`PrivateUsage` via `psapi!GetProcessMemoryInfo` (Windows only; same `NewLazySystemDLL` pattern as `GlobalMemoryStatusEx`). Non-Windows returns zeros; watchdog falls back to `MemStats` only.
- **`internal/database/sqlite.go`**: new `ProfilesDir()` parallel to `DemosDir()`; prunes oldest *.pprof files keeping the last 5.
- **`app.go`**: `debug.FreeOSMemory()` after `result.Events = nil` so Windows actually drops the working set after a memory-heavy parse (~50–200 ms cost, mostly no-op on macOS/Linux). New Wails bindings `ProfilesDir()`, `OpenProfilesFolder()`, `GetTolerateEntityErrors()`, `SetTolerateEntityErrors()`.

Refs: [[knowledge/demo-parser]] (updated), [[knowledge/wails-bindings]] (updated). ADR candidate: defense-in-depth watchdog + entity-panic default flip.

## 2026-05-08 — Windows 16 GB OOM safeguards

User report: demo import on a 16 GB Windows box drove the system to 99% RAM and froze it. The static 4 GiB parser watchdog only fired after the OS was already paging, and no `GOMEMLIMIT` meant the runtime let heap grow to ~2× live set during pre-fault working-set spikes.

- **`internal/sysinfo/`** (new): cross-platform total-RAM detection — `sysctl hw.memsize` (darwin), `Sysinfo` (linux), `kernel32!GlobalMemoryStatusEx` via `golang.org/x/sys/windows` LazySystemDLL. `RecommendedHeapLimits(totalRAM)` returns `GOMEMLIMIT=12.5%` / `KillSwitch=18.75%` of host RAM, clamped to `[1 GiB, 4 GiB]`. Tested table-driven for 0 / 8 / 16 / 32 / 64 GB hosts.
- **`main.go`**: `configureMemoryLimits()` runs after logging init; calls `debug.SetMemoryLimit()` unless `GOMEMLIMIT` env is already set; logs the chosen budgets.
- **`internal/demo/parser.go`**: `const maxHeapBytes` → per-parser field via new `WithHeapLimit(bytes)` option (default 4 GiB for direct callers / tests). Watchdog error now includes `(limit X MiB, observed Y MiB)` so a stale `errors.txt` records which budget was in effect.
- **`app.go`**: new `fileImportMu` serializes `ImportService.ImportFile` so a 10-zst bulk drop runs one decompression at a time (was N parallel zstd windows). Wails caller still gets sync error returns. `runtime.GC()` between `parser.Parse()` and `IngestRounds`/`IngestGameEvents`. After events ingest commits, `result.Events`/`Lineups`/`Rounds` are nil-ed so the ~125 MB max event slice can be reclaimed before "complete" emits.
- **`go.mod`**: `golang.org/x/sys` promoted from indirect to direct dep.

Refs: [[knowledge/demo-parser]] (updated), [[knowledge/wails-bindings]] (updated). No ADR — tuning change, not structural.

## 2026-05-08 — Cross-layer performance overhaul

5-agent perf audit → ~50 items applied across parser, DB, Wails IPC, React, and PixiJS.

- **Parser/ingest streaming:** ticks now flow through a bounded channel (`WithTickSink`); `app.go` runs parse + ingest concurrently via `errgroup.WithContext`. Peak tick-path heap dropped ~30× (~120 MB → ~4 MB). Events stay in `state.events` — `dropKnifeRounds`/`pairShotsWithImpacts`/`ExtractGrenadeLineups` need the full list. Watchdog kept (protobuf/entity-table state still grows on corrupt demos).
- **Ingest speedups:** typed `EventExtra` structs replace `map[string]interface{}`; multi-row VALUES batching for `InsertTickData` (~500 rows/INSERT); prepared statements reused within ingest tx; `RETURNING *` removed from `CreateGameEvent`; `parseState.steamID(p)` caches `strconv.FormatUint`. `app.go parseDemo` calls now serialized (no more N parsers × hundreds of MB).
- **SQLite pragmas:** `synchronous=NORMAL`, `busy_timeout=5000`, `cache_size=-64000`, `mmap_size=268435456`, `temp_store=MEMORY`, `journal_size_limit`. `MaxOpenConns` raised 1→4 — write serialization moves to `busy_timeout`, reads go concurrent. Migration 010 promotes hot `extra_data` fields (headshot, *_steam_id, *_name, *_team, health_damage, is_self_kill) to real columns; migration 011 moves `tick_data.inventory` to `round_loadouts(round_id, steam_id, inventory)`.
- **Wails IPC:** `DemoSummary` list-vs-detail split (saves ~10–20 KB/100 rows). New bindings: `GetAllRosters`, `GetEventsByTypes`, `GetRoundLoadouts`, `CountDemos`. `GameEvent.ExtraData` ships as `json.RawMessage` (one-time decode on JS side). `emitProgress` 100 ms coalescer (terminal stages bypass). Cancellable context plumbed through Startup → parseDemo → Shutdown.
- **React:** `React.lazy` per route. `useKillFeed` (typed query for kills only). Memoized `PlayerRow`/`WeaponLabelRow`/`HealthRow`/`LoadoutIcons`/`KillRow`/`RoundPill`. `gcTime: 60_000` global; `useDeleteDemo` removes per-demo cache keys. Game-events cache dropped after `EventLayer` consumes.
- **PixiJS:** `_resetWeaponTextureCache()` on viewer destroy (fixes stale destroyed-Texture bug). Single shared `TickBuffer` via `frontend/src/stores/tick-buffer.ts` (was duplicated in `useLoadoutSnapshot`). Scratch-buffer reuse: `worldToPixelInto`, reused `nextById`/`activeSteamIds` Maps in `PlayerLayer`, reused `FramePair`/`SampleFrame` in `TickBuffer.getFramePair`. Map-texture unload via `Assets.unload(prevUrl)` tracking `_loadedUrl`. Timeline drag listeners + `use-parse-progress` timers + `heatmap` `setMap` race all cleaned up on unmount.
- **Cleanup:** dropped `recharts`, dead `chunkTickParams`/`tickToParams`, redundant `idx_game_events_demo_id`. `TickData.X/Y/Z/Yaw` switched to `int16` over the wire (~150 KB/chunk savings). `CalculatePlayerRoundStats` rewritten as a single sorted-events pass (no per-round map).

Refs: [[knowledge/demo-parser]] (updated), [[knowledge/sqlite-wal]] (updated), [[knowledge/wails-bindings]] (updated), [[knowledge/migrations]] (updated), [[knowledge/sqlc-workflow]] (updated), [[knowledge/pixijs-viewer]] (updated). ADR candidates: streaming parse→ingest contract, `MaxOpenConns 1→4` revision to ADR-0008.

## 2026-05-07 — Grenade sprites + kill-log ordering

In-flight grenade icons swapped from colored dots to CS2 weapon SVG sprites; kill log now renders oldest→newest so the latest kill lands at the bottom of the feed.

- **Frontend (`event-layer.ts`):** added `SpritePool` alongside `GraphicsPool`. Each active `grenade_traj` effect acquires *both* a `Graphics` (trail polyline) and a `Sprite` (weapon icon at the lerped head); `drawEffect` / `drawGrenadeTrajectory` thread the optional `Sprite` through, and both pools are released in lock-step on expiry/clear/destroy.
- **Async-texture fallback:** `getWeaponTexture(weapon)` is sync-from-cache and returns `null` on first call (kicks off background `Assets.load`). The drawer falls back to the legacy colored dot when sprite or texture is missing, so the grenade is never invisible during the load delay or for unmapped weapons.
- **Pool-acquire reset:** `sprite.texture = null` on acquire so the per-tick `if (sprite.texture !== texture)` branch fires for the new effect's weapon and re-applies `GRENADE_ICON_HEIGHT / texture.height` scale — otherwise a pooled sprite carries over the previous grenade's texture/scale.
- **Kill log (`lib/viewer/kill-log.ts`):** `selectVisibleKills` switched from `sort desc → slice(0, N)` to `sort asc → slice(-N)`. Same N most-recent entries, but emitted oldest-first so React renders the latest kill last in the flex column.

Refs: [[knowledge/pixijs-viewer]] (updated). No ADR — UX/rendering tweaks on established patterns.

## 2026-05-07 — Active-weapon ammo in viewer overlay

Active weapon name + clip/reserve now render as a small subtitle under each player on the 2D map and as a row in the team bars (no sprites; text only). `tick_data` migration 008 adds `ammo_clip` / `ammo_reserve` columns (default 0); demos imported before this read 0/0 until re-imported.

- **Parser (`internal/demo/parser.go`):** `TickSnapshot.AmmoClip` / `AmmoReserve` populated from `player.ActiveWeapon().AmmoInMagazine()` / `AmmoReserve()`. Both 0 when there's no active weapon (knife, bomb, etc.).
- **Frontend formatter (`frontend/src/lib/viewer/weapon-label.ts`):** single source of truth for the displayed string — `null` when no weapon, `"WEAPON  clip / reserve"` when either ammo > 0, plain `"WEAPON"` otherwise. Consumed by both `PlayerSprite` and `TeamBars`.
- **Loadout snapshot:** `sameLoadout` in `use-loadout-snapshot.ts` extended to include the new fields; without that, ammo changes between 250 ms polls would be silently dropped from the team bars.

Refs: [[knowledge/demo-parser]] (updated), [[knowledge/pixijs-viewer]] (updated). No ADR — additive feature on established patterns.

## 2026-05-07 — Grenade trajectory rendering

In-flight grenades now render as a colored, lerped icon with a faint trail through bounce points. Implementation spans the parser, `effects.ts`, and `EventLayer`, plus a round-end duration cap.

- **Backend (`internal/demo/parser.go`):** new `GrenadeProjectileBounce` handler emits `grenade_bounce` events with `entity_id` + position. Without bounce capture the icon would teleport between throw and detonation.
- **Frontend (`frontend/src/lib/pixi/sprites/effects.ts`):** added `progress()` lerp (from the Healey article), `interpolateTrajectory(waypoints, currentTick)`, `computeGrenadeTrajectoryState`, `computeFireState`, and `grenadeColor()` per weapon type.
- **Frontend (`frontend/src/lib/pixi/layers/event-layer.ts`):** new `grenade_traj` + `fire` effect types. Two-pass `buildScheduled` indexes throws + bounces + terminations by `entity_id` (across `grenade_detonate` / `smoke_start` / `fire_start` / `decoy_start`), materializes a waypoint list per throw. Drawer paints a polyline through completed waypoints + a colored dot at the lerped head.
- **Round-end cap:** `setEvents(events, rounds?)` now caps each effect's `durationTicks` to the `end_tick` of its containing round. Smokes (~18 s), molotov fires (~7 s), and late-round trajectories used to persist into the next round's freeze; the cap mirrors CS2's natural round cleanup.
- **Side fix:** `entity_id` flows through Wails as a JSON `number`, not a `string` (Go's `Entity.ID()` is `int`). The pre-existing smoke-pairing's `typeof === "string"` check silently never matched, so smokes always used `SMOKE_DURATION_TICKS` instead of the actual `smoke_expired` tick. New `entityKey()` helper accepts both shapes.

Refs: [[knowledge/demo-parser]] (updated), [[knowledge/pixijs-viewer]] (updated). Existing demos must be re-imported to populate `grenade_bounce` events. No ADR — additive feature with a localized rendering-layer decision.

## 2026-05-07 — Confirmation dialog before removing a demo

Trash icon in `LibraryTable` now opens a shadcn `AlertDialog` showing the filename and a destructive Remove action; `onDelete(id)` only fires on confirm. Tests assert open / confirm / cancel paths. Refs: [[knowledge/testing]].

## 2026-05-07 — Shot tracers in 2D viewer

Every firearm shot now renders as a tracer line in `EventLayer`. Implementation lives end-to-end across the parser and PixiJS layer:

- **Backend (`internal/demo/parser.go`):** new `WeaponFire` handler emits `weapon_fire` events with shooter X/Y/Z and yaw/pitch. Filtered by `Equipment.Class()` to firearm classes (`EqClassPistols/SMG/Heavy/Rifle`) — `WeaponFire` also fires for grenade throws and knife slashes, which would otherwise pollute the tracer layer.
- **Backend (`internal/demo/shot_impacts.go`, new):** `pairShotsWithImpacts` walks events in tick order and pairs each `weapon_fire` with the most recent prior unpaired shot from the same attacker when a `player_hurt` arrives within 16 ticks (~250ms). On pair, writes `hit_x`/`hit_y` into the shot's `extra_data`. One shot pairs with at most one hurt event (wallbangs through multiple players record only the first impact).
- **Backend (`player_hurt` handler):** previously left `X/Y/Z` as zero; now populated with `e.Player.Position()` so the pairing pass has a usable endpoint.
- **Frontend (`frontend/src/lib/pixi/layers/event-layer.ts`):** `drawShot` branches on `hit_x`/`hit_y`. When set: solid line shooter→impact + small filled circle at the impact, full alpha. When absent: 16-segment gradient ray of fixed length (`SHOT_TRACER_LENGTH = 2000` world units) fading toward the unknown endpoint — PixiJS Graphics has no native gradient strokes, so segment stacking approximates it.

Refs: [[knowledge/demo-parser]] (updated), [[knowledge/pixijs-viewer]] (updated). Existing demos must be re-imported to populate `weapon_fire` events. No ADR — feature addition with established patterns.

## 2026-05-07 — Demo parser quality fixes

Implemented six independent fixes from `.claude/temp/parser-quality-plan.md`:

- **Numeric-nickname bug:** parser captures a per-round roster at freeze-end (`RoundData.Roster`) and `stats.calculateRound` seeds the player map from it, so passive players (no kills/damage/deaths) now get a `player_rounds` row instead of falling back to the frontend's `steam_id.slice(0, 10)`.
- **Hardcoded MR12 overtime:** `IsOvertime` is now sourced from `p.GameState().OvertimeCount() > 0` at `RoundEnd`. Added `parseState.ensureFormat()` reading `mp_maxrounds` for a warn-only invariant check. The `isOvertime(roundNum)` helper is gone; `dropKnifeRounds` no longer recomputes the flag.
- **Knife inventory hardening:** `captureFreezeEnd` filters `IsAlive`; `isKnifeRoundByInventory` requires pure `{EqKnife}` and `>= 8` samples (was: any subset, 1+ samples, C4-allowed).
- **Warmup gate unification:** removed `state.inWarmup` + the `IsWarmupPeriodChanged` handler — the cached value lagged by one dispatch and let `RoundNumber=0` events leak through. Every gate now reads `p.GameState().IsWarmupPeriod()`.
- **Score read at `RoundEnd`:** dropped `e.WinnerState.Score()` reads. `state.ctScore/tScore` (kept current by `ScoreUpdated`, which fires before `RoundEnd` in v5) is the source of truth; increment-from-winner fallback if missing.
- **Team clan names:** migration `006_rounds_team_names` adds `ct_team_name`/`t_team_name` (default `''`), captured from `gs.TeamCounterTerrorists().ClanName()` at `RoundEnd` and threaded through sqlc → Wails binding → frontend `Round` type. `MatchHeader` prefers per-round clan name, falls back to `team_<player>` then `"CT"`/`"T"`.

Refs: [[knowledge/demo-parser]] (updated). No schema/binding changes outside #6; no new ADR (bug-fix sweep, no architectural decision).

## 2026-05-06 — Faceit integration removed

The app pivots away from the Faceit-account-tied desktop client to a single-tenant local tool. Removed:

- Backend: `internal/auth/`, `internal/faceit/`, `vars.go`, all auth/sync/download bindings in `app.go`
- Database: migration 005 drops `users`, `faceit_matches`, and the `demos.user_id` / `demos.faceit_match_id` columns; `queries/users.sql` and `queries/faceit_matches.sql` deleted; `demos.sql` no longer filters by user
- Frontend: dashboard, auth provider, login/callback/match-detail routes, all faceit hooks/store/types, Faceit MSW handlers and binding mocks; index now redirects to `/demos`
- CI: Faceit secrets and `LDFLAGS` injection removed from `.github/workflows/release.yml` (delete the `FACEIT_*` repo secrets manually)
- Docs: added [[decisions/0014-remove-faceit-integration|ADR-0014]] superseding [[decisions/0005-faceit-oauth-pkce|ADR-0005]] and [[decisions/0009-loopback-oauth-desktop|ADR-0009]] (both retained as historical superseded records); deleted `knowledge/faceit-oauth.md` and `plans/p4-faceit-heatmaps.md`; de-Faceit-ified the rest
- Known straggler: `internal/demo/compress.go:67` still uses `"faceit-demo-*.dem"` as the zstd-decompression temp-file prefix — cosmetic, unrelated to the integration; rename in a follow-up

The 2D demo player and per-demo analytics remain. Comprehensive per-player statistics built on `player_rounds` / `game_events` are a follow-up plan. Plan: `~/.claude/plans/i-ve-changed-the-original-frolicking-bengio.md`.

## 2026-04-24 — Docs vault reorganization

Restructured `docs/` into an Obsidian vault with split monoliths, a knowledge wiki, and an entry-point MOC.

- Split [[product/vision|PRD]] into 7 topical pages under `product/`
- Split [[architecture/overview|ARCHITECTURE]] into 8 Arc42-aligned pages under `architecture/`
- Renamed `adr/` → `decisions/`, `TASK_BREAKDOWN.md` → `tasks.md`, `IMPLEMENTATION_PLAN.md` → `roadmap.md`
- Lowercased phase plan filenames
- Seeded [[knowledge/README|knowledge wiki]] with 9 entity/pattern pages
- Absorbed `spike-parser-findings.md` into [[knowledge/demo-parser]]
- Added [[index]] as MOC and this log
- Updated root `README.md` to reference the new vault layout (MOC + subfolder links)

Rationale: the flat `docs/` folder with four 800–1250 line monoliths was hard to navigate and edit. Splits give cleaner URLs, smaller LLM edit surfaces, and surface-area for a curated wiki layer. See `/Users/sundayfunday/.claude/plans/im-thinking-about-having-binary-clover.md` for the plan that drove this change.
