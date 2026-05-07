# Project Log

Append-only chronological record of notable project activity. New entries at the top. Maintained via the `/ingest-session` slash command.

Format: `YYYY-MM-DD — <summary>` with links to affected pages.

---

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
