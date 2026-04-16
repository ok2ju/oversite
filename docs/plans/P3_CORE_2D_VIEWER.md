# Phase 3: Core 2D Viewer — Implementation Plan

## Context

Phase 3 delivers the core 2D demo viewer: users can open a parsed demo, see the map with player positions, watch playback at variable speeds, view events (kills, grenades, bombs), navigate rounds, and check the scoreboard. The TASK_BREAKDOWN.md lists 12 tasks (P3-T01 through P3-T12).

**Critical finding**: After thorough codebase exploration, ~90% of the Phase 3 frontend is already implemented from prior waves. All PixiJS rendering layers, playback engine, tick buffer, playback controls UI, camera/zoom/pan, minimap, Zustand store bridge, and data-fetching hooks are built and tested. The remaining work is:

1. **4 Go backend binding stubs** — the viewer frontend calls these but they return `errNotImplemented`
2. **Demo viewer route page** — `demo-viewer.tsx` is a placeholder `<div>`
3. **Round selector** (P3-T08) — component doesn't exist
4. **Scoreboard overlay** (P3-T10) — component doesn't exist
5. **Keyboard shortcuts** (P3-T11) — hook doesn't exist

---

## Wave 1: Go Backend Bindings (Critical Path)

All frontend components are blocked on real data. This wave implements the 4 stub bindings + 1 new binding + 1 new SQL query.

### W1-A: `GetDemoByID` binding
- **File**: `app.go`
- **Why**: Viewer route needs demo metadata (mapName, totalTicks, tickRate) by ID
- **How**: Parse string→int64, call `a.queries.GetDemoByID()` (already exists at `internal/store/demos.sql.go:81`), convert via existing `storeDemoToBinding()` helper
- **Signature**: `func (a *App) GetDemoByID(id string) (*Demo, error)`

### W1-B: `GetDemoRounds` binding
- **File**: `app.go:270` (replace stub)
- **How**: Parse ID, call `a.queries.GetRoundsByDemoID()`, convert via new `storeRoundToBinding()` helper
- **Type mapping** (`store.Round` → `Round`):
  - `ID`: `strconv.FormatInt(r.ID, 10)` (int64→string)
  - `RoundNumber/StartTick/EndTick/CTScore/TScore`: `int(field)` (int64→int)
  - `IsOvertime`: derive from `roundNumber > 24` (MR12 regulation)

### W1-C: `GetDemoEvents` binding
- **File**: `app.go:275` (replace stub)
- **How**: Parse ID, call `a.queries.GetGameEventsByDemoID()`, convert via new `storeGameEventToBinding()`
- **Type mapping** (`store.GameEvent` → `GameEvent`):
  - `AttackerSteamID/VictimSteamID/Weapon`: `sql.NullString` → `*string`
  - `X/Y/Z`: `float64` → `*float64` (take address)
  - `ExtraData`: `string` (JSON) → `map[string]any` via `json.Unmarshal`, fallback to nil on empty/error

### W1-D: `GetDemoTicks` binding
- **File**: `app.go:280` (replace stub)
- **How**: Parse ID, call `a.queries.GetTickDataByRange()`, convert via new `storeTickDatumToBinding()`
- **Type mapping** (`store.TickDatum` → `TickData`):
  - `IsAlive`: `d.IsAlive != 0` (int64→bool)
  - `Weapon`: `string` → `*string`
- **Performance note**: Hottest path (~64k rows per chunk). Already indexed by `(demo_id, tick)`.

### W1-E: `GetRoundRoster` binding
- **File**: `app.go:285` (replace stub)
- **How**: Two-step query — `GetRoundByDemoAndNumber()` to get round ID, then `GetPlayerRoundsByRoundID()`. Convert `store.PlayerRound` → `PlayerRosterEntry` (SteamID, PlayerName, TeamSide)
- **Edge case**: Return empty slice (not error) if round not found

### W1-F: Scoreboard SQL query + binding
- **New SQL** in `queries/player_rounds.sql`:
  ```sql
  -- name: GetPlayerStatsByDemoID :many
  SELECT pr.steam_id, pr.player_name, pr.team_side,
         SUM(pr.kills) as total_kills, SUM(pr.deaths) as total_deaths,
         SUM(pr.assists) as total_assists, SUM(pr.damage) as total_damage,
         SUM(pr.headshot_kills) as total_headshot_kills, COUNT(*) as rounds_played
  FROM player_rounds pr
  JOIN rounds r ON pr.round_id = r.id
  WHERE r.demo_id = @demo_id
  GROUP BY pr.steam_id ORDER BY pr.team_side, total_kills DESC;
  ```
- **Run**: `make sqlc` to regenerate
- **New type** in `types.go`: `ScoreboardEntry` (SteamID, PlayerName, TeamSide, K/D/A, Damage, HSKills, RoundsPlayed, HSPercent, ADR — last two computed in Go)
- **New binding**: `GetScoreboard(demoID string) ([]ScoreboardEntry, error)`

### W1-G: Go tests
- **New file**: `app_test.go` — table-driven tests for all 6 bindings
- Uses `testutil.NewTestQueries(t)` with seeded demo/rounds/events/ticks/player_rounds
- Tests: valid ID, invalid string ID, not-found, empty results, null field handling, JSON parsing

### Verification
- `go test -race ./...` passes
- `go vet ./...` clean
- `wails dev` builds (confirms Wails binding generation)

---

## Wave 2: Demo Viewer Route (Unlocks E2E)

Wire the stub `demo-viewer.tsx` into the full viewer by composing existing components.

### W2-A: `useDemo` hook
- **New file**: `frontend/src/hooks/use-demo.ts`
- TanStack Query hook calling `GetDemoByID(id)`, enabled when `id` is defined

### W2-B: Implement `demo-viewer.tsx`
- **File**: `frontend/src/routes/demo-viewer.tsx` (replace stub)
- Extract `id` from `useParams`, fetch via `useDemo(id)`
- On load: set viewer store state (`setDemoId`, `setMapName`, `setTotalTicks`)
- Render: `<ViewerCanvas />`, `<PlaybackControls />`, `<MiniMap />`
- Handle: loading spinner, error state (demo not found / not ready), cleanup via `reset()` on unmount
- Layout: Full-height within root layout (`h-[calc(100vh-4rem)]`, `overflow-hidden`)

### W2-C: Update test mocks
- **File**: `frontend/src/test/mocks/bindings.ts` — add `GetDemoByID` mock

### W2-D: Route test
- **New file**: `frontend/src/routes/demo-viewer.test.tsx`
- Cases: loading state, renders viewer canvas on ready demo, error state, store cleanup on unmount

### Verification
- `pnpm test` + `pnpm typecheck` pass
- `wails dev` → navigate to `/demos/{id}` with a real imported demo, see map + players render

---

## Wave 3: Round Selector (P3-T08)

### W3-A: `RoundSelector` component
- **New file**: `frontend/src/components/viewer/round-selector.tsx`
- Uses existing `useRounds(demoId)` hook + viewer store
- shadcn `Select` dropdown: format `"Round {n}: {ct_score}-{t_score}"`, winner side color indicator
- Selecting a round: calls `setRound(n)` + `setTick(round.start_tick)`
- Current round highlighted (determined by comparing `currentTick` to round boundaries)

### W3-B: Test
- **New file**: `frontend/src/components/viewer/round-selector.test.tsx`
- Cases: renders all rounds, correct scores, selecting jumps to tick, highlights current round

### W3-C: Wire into viewer page
- Add `<RoundSelector />` to `demo-viewer.tsx` layout

### Verification
- `pnpm test` passes
- Visual: round selector jumps playback between rounds

---

## Wave 4: Scoreboard Overlay (P3-T10)

### W4-A: Frontend type + hook
- **New file**: `frontend/src/types/scoreboard.ts` — `ScoreboardEntry` interface
- **New file**: `frontend/src/hooks/use-scoreboard.ts` — TanStack Query hook calling `GetScoreboard(demoId)`

### W4-B: `Scoreboard` component
- **New file**: `frontend/src/components/viewer/scoreboard.tsx`
- Toggle-able overlay (`bg-black/80 backdrop-blur-sm`)
- Two team sections (CT/T), shadcn `Table` with columns: Player, K, D, A, ADR, HS%
- Clicking a player row selects them in viewer store
- Selected player highlighted

### W4-C: Test + mocks
- **New file**: `frontend/src/components/viewer/scoreboard.test.tsx`
- Update `bindings.ts` — add `GetScoreboard` mock
- Update `fixtures/demos.ts` — add `mockScoreboardEntries`
- Cases: hidden when not visible, renders teams, correct stats, player selection

### W4-D: Wire into viewer page
- Add scoreboard state + `<Scoreboard />` to `demo-viewer.tsx`

### Verification
- `pnpm test` + `go test -race ./...` pass
- Visual: scoreboard overlay shows correct data

---

## Wave 5: Keyboard Shortcuts (P3-T11)

### W5-A: `useViewerKeyboard` hook
- **New file**: `frontend/src/hooks/use-viewer-keyboard.ts`
- Bindings: Space (play/pause), Left/Right (seek 5s), Up/Down (speed cycle), Tab (scoreboard toggle), Escape (close overlays/deselect), R (reset viewport)
- Skips events when target is `<input>` or `<textarea>`
- Takes `onToggleScoreboard` callback (scoreboard visibility is local page state)

### W5-B: Test
- **New file**: `frontend/src/hooks/use-viewer-keyboard.test.ts`
- Cases: each keybinding, input field skip, Tab preventDefault

### W5-C: Wire into viewer page
- Add `useViewerKeyboard({ onToggleScoreboard })` to `demo-viewer.tsx`

### Verification
- `pnpm test` passes
- Visual: all keyboard shortcuts functional in `wails dev`

---

## Task-to-Wave Mapping

| TASK_BREAKDOWN ID | Status | Wave | Notes |
|---|---|---|---|
| P3-T01 PixiJS setup | Done | — | `lib/pixi/app.ts` |
| P3-T02 Map layer | Done | — | `lib/pixi/layers/map-layer.ts`, `calibration.ts` |
| P3-T03 Tick data fetching | Frontend done, **Go stub** | W1-D | Implement `GetDemoTicks` binding |
| P3-T04 Player layer | Done | — | `lib/pixi/layers/player-layer.ts` |
| P3-T05 Event layer | Done | — | `lib/pixi/layers/event-layer.ts` |
| P3-T06 Playback engine | Done | — | `lib/pixi/playback-engine.ts` |
| P3-T07 Playback controls | Done | — | `components/viewer/playback-controls.tsx` |
| P3-T08 Round selector | **Not started** | W3 | New component |
| P3-T09 Zoom & pan | Done | — | `lib/pixi/camera.ts`, `mini-map.tsx` |
| P3-T10 Scoreboard | **Not started** | W1-F + W4 | Needs new SQL query + Go binding + component |
| P3-T11 Keyboard shortcuts | **Not started** | W5 | New hook |
| P3-T12 Store bridge | Done | — | `viewer-canvas.tsx` subscriptions |

---

## Tasks Needing Additional Planning

### P3-T10 Scoreboard — Needs design review
The scoreboard needs a new SQL aggregation query (`GetPlayerStatsByDemoID`) that joins `player_rounds` with `rounds`. The `sqlc` codegen for aggregate queries with `SUM()` may produce `interface{}` or `int64` return types that need careful handling. Need to verify after `make sqlc` that the generated types are usable. If sqlc struggles with the aggregate, fallback: fetch raw `PlayerRound` rows in Go and aggregate in-memory.

### Viewer page layout — Minor design decision
The viewer must fill available height within the root layout (sidebar + header). Using `h-[calc(100vh-4rem)]` with `overflow-hidden` is the straightforward approach. If the header height is not exactly `4rem`, measure it during W2 implementation.

---

## Critical Path

```
W1 (Go bindings) → W2 (Viewer route) → W3 (Round selector)
                                      → W4 (Scoreboard) → W5 (Keyboard)
```

W3 and W4 can run in parallel after W2. W5 depends on W4 for scoreboard toggle integration.

## Estimated Effort

| Wave | Effort | Description |
|---|---|---|
| W1 | 2-3 hours | Go bindings, type conversion, SQL, tests |
| W2 | 1-2 hours | Route wiring, hook, test |
| W3 | 1 hour | Round selector component + test |
| W4 | 1.5-2 hours | Scoreboard type/hook/component + test |
| W5 | 1 hour | Keyboard hook + test |
| **Total** | **~7-9 hours** | |

## Key Files Reference

**Go (modify)**:
- `app.go` — implement bindings (lines 269-287), add conversion helpers
- `types.go` — add `ScoreboardEntry`
- `queries/player_rounds.sql` — add aggregate query

**Go (read-only, for reference)**:
- `internal/store/models.go` — sqlc-generated types for conversion
- `internal/store/demos.sql.go:81` — `GetDemoByID` query
- `internal/store/rounds.sql.go` — `GetRoundsByDemoID`, `GetRoundByDemoAndNumber`
- `internal/store/game_events.sql.go` — `GetGameEventsByDemoID`
- `internal/store/tick_data.sql.go` — `GetTickDataByRange`
- `internal/store/player_rounds.sql.go` — `GetPlayerRoundsByRoundID`

**Frontend (modify)**:
- `frontend/src/routes/demo-viewer.tsx` — full implementation
- `frontend/src/test/mocks/bindings.ts` — add new mocks

**Frontend (create)**:
- `frontend/src/hooks/use-demo.ts`
- `frontend/src/components/viewer/round-selector.tsx`
- `frontend/src/types/scoreboard.ts`
- `frontend/src/hooks/use-scoreboard.ts`
- `frontend/src/components/viewer/scoreboard.tsx`
- `frontend/src/hooks/use-viewer-keyboard.ts`
- Plus test files for each

**Frontend (read-only, reuse)**:
- `frontend/src/components/viewer/viewer-canvas.tsx` — existing orchestrator
- `frontend/src/components/viewer/playback-controls.tsx` — existing controls
- `frontend/src/components/viewer/mini-map.tsx` — existing minimap
- `frontend/src/stores/viewer.ts` — viewer Zustand store
- `frontend/src/hooks/use-rounds.ts` — existing rounds hook
- `frontend/src/hooks/use-game-events.ts` — existing events hook
- `frontend/src/hooks/use-roster.ts` — existing roster fetch
- `frontend/src/test/fixtures/demos.ts` — existing mock data

## End-to-End Verification

After all waves:
1. `go test -race ./...` — all Go tests pass
2. `pnpm test` — all frontend tests pass
3. `pnpm typecheck` — clean
4. `wails dev` → import a real `.dem` file → open viewer:
   - Map renders with correct radar image
   - Players appear at correct positions with team colors
   - Play/pause works, speed control works
   - Seek via timeline and arrow keys
   - Round selector jumps between rounds
   - Events (kills, grenades, bombs) render with correct timing
   - Scoreboard overlay via Tab key shows accurate stats
   - Zoom/pan with scroll wheel and drag
   - Minimap shows viewport position
   - 60 FPS maintained during playback
