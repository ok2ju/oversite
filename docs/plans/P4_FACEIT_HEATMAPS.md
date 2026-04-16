# Phase 4: Faceit & Heatmaps — Implementation Plan

## Context

Phase 4 delivers two independent feature tracks: (A) the Faceit integration pipeline — sync matches from the Faceit API to SQLite, display a stats dashboard with ELO history, and download demos from match rooms; (B) interactive KDE heatmaps — aggregate kill positions across one or many demos and render them as a density overlay on the map.

**Critical finding**: ~70% of the infrastructure is already built. The Faceit HTTP client (`internal/auth/faceit.go`) has `GetPlayer`, `GetPlayerHistory`, `GetMatchDetails`. The DB schema, SQL queries (UpsertFaceitMatch, GetFaceitMatchesFiltered, GetEloHistory, GetCurrentStreak), domain types (`types.go`), TypeScript interfaces (`types/faceit.ts`), TanStack Query hooks (`use-faceit.ts`, `use-faceit-matches.ts`), and dashboard UI components (`profile-card.tsx`, `elo-chart.tsx`, `match-list.tsx`) are implemented. The heatmap aggregation query (`GetHeatmapAggregation` in `internal/store/heatmaps_custom.go`) is also written. Three binding stubs in `app.go:431-444` return `errNotImplemented`.

**Remaining work**:
1. Wire the 3 stub bindings to real queries
2. Create Faceit sync service (`internal/faceit/sync.go`)
3. Add `SyncFaceitMatches` binding + frontend sync trigger
4. Demo download from Faceit match URLs
5. Heatmap Wails binding (wire `GetHeatmapAggregation` + supporting queries)
6. KDE algorithm + PixiJS HeatmapLayer
7. Heatmap filter UI + page
8. Per-demo stats view

---

## Parallel Tracks

The two tracks share no code dependencies and can be developed in parallel:

```
TRACK A: Faceit (W1 → W2 → W3 → W4)
  W1: Backend bindings (wire stubs to real queries)
  W2: Faceit sync service (fetch → upsert → progress events)
  W3: Dashboard page wiring + sync button
  W4: Demo download from Faceit

TRACK B: Heatmaps (W5 → W6 → W7)
  W5: Heatmap data binding (Wails + supporting queries)
  W6: KDE algorithm + PixiJS heatmap layer
  W7: Heatmap filter UI + per-demo stats
```

---

## Wave 1: Backend Bindings — Wire Stubs to Real Queries

### W1-A: `GetFaceitProfile` binding
- **File**: `app.go` (replace stub at line 432)
- **Logic**: Get current user from `a.authService.GetCurrentUser(ctx)` → query `CountFaceitMatchesByUserID` for matches_played → query `GetCurrentStreak` for streak → build `FaceitProfile` from user table fields (nickname, avatar_url, faceit_elo, faceit_level, country)
- **Edge cases**: Not logged in → return error. Zero matches → streak `{Type: "none", Count: 0}`. Empty avatar_url → nil pointer.

### W1-B: `GetEloHistory` binding
- **File**: `app.go` (replace stub at line 437)
- **Logic**: Get current user → compute `since` from `days` param (`time.Now().AddDate(0, 0, -days)`, 0 = epoch) → call `a.queries.GetEloHistory(ctx, ...)` → convert `[]store.GetEloHistoryRow` → `[]EloHistoryPoint`

### W1-C: `GetFaceitMatches` binding
- **File**: `app.go` (replace stub at line 442)
- **Logic**: Get current user → call `GetFaceitMatchesFiltered` + `CountFaceitMatchesFiltered` → convert `[]store.FaceitMatch` → `[]FaceitMatch`
- **Type mapping**: `ID` int64→string, `EloBefore/EloAfter` int64→`*int` (nil if 0), `EloChange` computed, `DemoUrl` string→`*string` (nil if empty), `DemoID` NullInt64→`*string`, `HasDemo` = DemoID.Valid

### W1-D: Helpers
- `computeStreak(results []string) CurrentStreak` — iterate from index 0, count consecutive same-result
- `storeFaceitMatchToBinding(m store.FaceitMatch) FaceitMatch` — follows existing `storeDemoToBinding` pattern at `app.go:450`

### W1-E: Go tests
- **File**: `app_test.go` (extend)
- **New helper**: `seedFaceitMatches(t, q, userID)` — inserts 3-5 matches with varied results/elos/maps
- **Challenge**: `newTestApp` doesn't set `authService`. Create `newTestAppWithUser(t)` returning `(*App, *store.Queries, store.User)` that creates a user, seeds a MockKeyring, and initializes AuthService.
- **Tests**: `TestGetFaceitProfile` (not logged in, no matches, with streak), `TestGetEloHistory` (30d, all, empty), `TestGetFaceitMatches` (unfiltered, map filter, result filter, pagination), `TestComputeStreak` (win/loss/mixed/empty)

### Verification
- `go test -race ./...` passes, `go vet ./...` clean, `wails dev` builds

---

## Wave 2: Faceit Sync Service (P4-T02)

### W2-A: Sync service
- **New file**: `internal/faceit/sync.go` (new package)
- **Struct**: `SyncService{faceit testutil.FaceitClient, queries *store.Queries}`
- **Method**: `SyncMatches(ctx, userID int64, faceitID string, onProgress func(current, total int)) (int, error)`
- **Algorithm**:
  1. `GetExistingFaceitMatchIDs(ctx, userID)` → build set for O(1) lookup
  2. Paginate `GetPlayerHistory(ctx, faceitID, offset, 20)` (max 200 matches)
  3. For each new match: optionally `GetMatchDetails` for demo URL → `UpsertFaceitMatch`
  4. Emit progress via callback
  5. Return count of newly inserted matches
- **Rate limiting**: Fixed 100ms delay between API calls. Retry with exponential backoff on 429.

### W2-B: `SyncFaceitMatches` Wails binding
- **File**: `app.go` (new method)
- **Logic**: Get current user → get access token → `auth.WithAccessToken(ctx, token)` → call `syncService.SyncMatches(...)` → progress callback emits `wailsRuntime.EventsEmit(a.ctx, "faceit:sync:progress", ...)`
- **App struct**: Add `syncService *faceit.SyncService`, init in `Startup`

### W2-C: Frontend hook
- **New file**: `frontend/src/hooks/use-faceit-sync.ts`
- `useFaceitSync()` — `useMutation` calling `SyncFaceitMatches()`, on success invalidates `["faceit"]` query keys

### W2-D: Tests
- **New file**: `internal/faceit/sync_test.go` — table-driven using `MockFaceitClient` + `NewTestQueries(t)`
- **Cases**: empty history, new matches, skip existing, progress callback, API error
- **Frontend**: Add `SyncFaceitMatches` mock to `bindings.ts`

### Verification
- `go test -race ./internal/faceit/...` passes

---

## Wave 3: Dashboard + Sync Button (P4-T03 + P4-T04)

### W3-A: Sync button
- **File**: `frontend/src/routes/dashboard.tsx`
- Add "Sync Matches" button in page header using `useFaceitSync()`, loading spinner while syncing, TanStack invalidation auto-refetches profile/elo/matches

### W3-B: Sync progress events
- **New file**: `frontend/src/hooks/use-faceit-sync-progress.ts`
- Listen for `"faceit:sync:progress"` Wails event (follow `use-parse-progress` pattern from `demos.tsx`)

### W3-C: Match list "Import Demo" prep
- **File**: `frontend/src/components/dashboard/match-list.tsx`
- Wire "Import Demo" button to `ImportMatchDemo` (disabled until W4)

### W3-D: Tests
- `dashboard.test.tsx` — sync button renders, loading state, queries invalidated after sync

### Verification
- `pnpm test` + `pnpm typecheck` pass
- `wails dev` → sync button populates matches

---

## Wave 4: Demo Download from Faceit (P4-T05)

### W4-A: Download service
- **New file**: `internal/faceit/download.go`
- **Struct**: `DownloadService{httpClient, importService *demo.ImportService, queries, downloadDir}`
- **Method**: `DownloadAndImport(ctx, faceitMatchID int64, userID int64, onProgress func(bytesDownloaded, totalBytes int64)) (*store.Demo, error)`
- **Algorithm**: Get match → extract DemoUrl → HTTP GET + stream to temp file → decompress `.gz` if needed → `importService.ImportFile(ctx, filePath, userID)` → `LinkFaceitMatchToDemo` → return demo

### W4-B: `ImportMatchDemo` Wails binding
- **File**: `app.go`
- Parse ID → get user → call download service → emit `"faceit:demo:download:progress"` events → trigger parse pipeline

### W4-C: Frontend wiring
- **File**: `frontend/src/components/dashboard/match-list.tsx` — enable "Import Demo" button
- **New file**: `frontend/src/hooks/use-demo-download.ts` — `useMutation` calling `ImportMatchDemo`, invalidates demo + faceit-matches queries

### W4-D: Tests
- **New file**: `internal/faceit/download_test.go` — `httptest.NewServer` serving fake `.dem`, test success/error/progress/gz-decompression
- Frontend: add `ImportMatchDemo` mock, match-list import button test

### Verification
- `go test -race ./internal/faceit/...` passes
- `wails dev` → import demo from match, appears in library

---

## Wave 5: Heatmap Data Binding (P4-T06)

### W5-A: `GetHeatmapData` binding
- **File**: `app.go` (new method)
- **Signature**: `GetHeatmapData(demoIDs []int64, weapons []string, playerSteamID string, side string) ([]HeatmapPoint, error)`
- **New types** in `types.go`: `HeatmapPoint{X, Y float64; KillCount int}`
- **Logic**: Validate user owns demos via `GetDemosByIDs` → convert arrays to JSON strings for `json_each()` → call `GetHeatmapAggregation` → convert result

### W5-B: Supporting queries
- **File**: `internal/store/heatmaps_custom.go` (extend)
- `GetDistinctWeapons(ctx, demoIDs string) ([]string, error)` — distinct weapons from game_events for given demo IDs
- `GetDistinctPlayers(ctx, demoIDs string) ([]PlayerInfo, error)` — distinct attacker steam_id + name from game_events

### W5-C: Supporting bindings
- `GetUniqueWeapons(demoIDs []int64) ([]string, error)` — for filter dropdowns
- `GetUniquePlayers(demoIDs []int64) ([]PlayerInfo, error)` — for player selector
- **New type** in `types.go`: `PlayerInfo{SteamID, PlayerName string}`

### W5-D: `GetWeaponStats` binding (for per-demo stats in W7)
- **New SQL** in `queries/game_events.sql`:
  ```sql
  -- name: GetWeaponStatsByDemoID :many
  SELECT weapon, COUNT(*) as kill_count, 
         SUM(CASE WHEN json_extract(extra_data, '$.headshot') = 1 THEN 1 ELSE 0 END) as hs_count
  FROM game_events WHERE demo_id = @demo_id AND event_type = 'kill' AND weapon IS NOT NULL
  GROUP BY weapon ORDER BY kill_count DESC;
  ```
- `make sqlc` to regenerate
- **New type** in `types.go`: `WeaponStat{Weapon string; KillCount, HSCount int}`

### W5-E: Frontend types + hooks
- **New file**: `frontend/src/types/heatmap.ts` — `HeatmapPoint`, `DemoSummary`, `PlayerInfo`
- **New file**: `frontend/src/hooks/use-heatmap.ts` — `useHeatmapData()`, `useUniqueWeapons()`, `useUniquePlayers()`

### W5-F: Go tests
- `app_test.go` — `TestGetHeatmapData` (with kills, empty, filter by weapon/player/side)

### Verification
- `go test -race ./...` passes, `pnpm typecheck` clean

---

## Wave 6: KDE Algorithm + PixiJS Heatmap Layer (P4-T07)

### W6-A: KDE algorithm
- **New file**: `frontend/src/lib/pixi/kde.ts`
- **Exports**: `computeKDE(points: KDEPoint[], gridWidth, gridHeight, bandwidth) → DensityGrid`
- **Algorithm**: Gaussian KDE with cutoff radius `3 * bandwidth` for O(n * (6b)^2) instead of O(n * w * h). Default grid resolution 512x512, bandwidth 15-20px.
- `KDEPoint{x, y, weight}`, `DensityGrid{data: Float32Array, width, height, maxDensity}`

### W6-B: Color map
- **New file**: `frontend/src/lib/pixi/colormap.ts`
- `densityToRGBA(normalizedDensity: number) → [r, g, b, a]`
- Blue→cyan→green→yellow→red gradient. Density 0 = transparent, high = red/opaque.
- Pre-computed 256-entry lookup table.

### W6-C: HeatmapLayer
- **New file**: `frontend/src/lib/pixi/layers/heatmap-layer.ts`
- **Class**: `HeatmapLayer{container, texture, sprite}`
- `render(points, calibration, options?)` — convert world→pixel via `worldToPixel()`, compute KDE, create RGBA ImageData via colormap, create PixiJS Texture from canvas, scale sprite to map dimensions
- `clear()` / `destroy()` — dispose texture to avoid GPU memory leaks
- **Texture pattern**: Create offscreen canvas → `ctx.getImageData` → fill from density grid + colormap → `ctx.putImageData` → `Texture.from(canvas)`

### W6-D: Tests
- **New file**: `frontend/src/lib/pixi/kde.test.ts` — single point peak, overlapping points, empty input, weight scaling, grid dimensions, maxDensity
- **New file**: `frontend/src/lib/pixi/layers/heatmap-layer.test.ts` — follow `event-layer.test.ts` pattern: mock Container, verify sprite creation/position/scaling, clear/destroy behavior

### Verification
- `pnpm test` passes, KDE algorithm verified numerically

---

## Wave 7: Heatmap UI + Per-Demo Stats (P4-T08 + P4-T09)

### W7-A: Heatmap Zustand store
- **New file**: `frontend/src/stores/heatmap.ts`
- **State**: `selectedDemoIds`, `selectedMap`, `selectedWeapons`, `selectedPlayer`, `selectedSide`, `bandwidth`, `opacity` + setters + `reset()`
- Pattern: follow `stores/viewer.ts` with `subscribeWithSelector`

### W7-B: HeatmapCanvas component
- **New file**: `frontend/src/components/heatmap/heatmap-canvas.tsx`
- Pattern: follow `viewer-canvas.tsx` — useRef container, useEffect init, Zustand subscribe bridge
- Simpler: just `MapLayer` + `HeatmapLayer`, no playback/players/events
- Subscribe to heatmap store → update map layer on map change, re-render heatmap on data change

### W7-C: Filter panel
- **New file**: `frontend/src/components/heatmap/filter-panel.tsx`
- Controls: Map `Select` (7 CS2 maps), Demo multi-select (from `useUserDemos`, filtered by map), Side radio (Both/CT/T), Weapon multi-select (from `useUniqueWeapons`), Player `Select` (from `useUniquePlayers`), Bandwidth `Slider` (5-50), Opacity `Slider` (0.1-1.0)
- Each control updates heatmap store → triggers TanStack refetch → heatmap re-renders

### W7-D: Heatmap page
- **File**: `frontend/src/routes/heatmaps.tsx` (replace stub)
- Layout: Filter panel (left sidebar ~280px) + HeatmapCanvas (fills remaining)

### W7-E: Per-demo stats panel
- **New file**: `frontend/src/components/viewer/stats-panel.tsx`
- Weapon kill breakdown (bar chart via recharts), HS% per weapon
- Uses `GetWeaponStats(demoID)` from W5-D + existing `GetScoreboard`
- Toggle from demo viewer page (tab or button alongside scoreboard)

### W7-F: Wire stats panel into viewer
- **File**: `frontend/src/routes/demo-viewer.tsx` — add tab/button to toggle between scoreboard and stats panel

### W7-G: Tests
- `frontend/src/stores/heatmap.test.ts` — state assertions
- `frontend/src/components/heatmap/filter-panel.test.tsx` — renders controls, store updates
- `frontend/src/routes/heatmaps.test.tsx` — page renders with filter + canvas
- `frontend/src/components/viewer/stats-panel.test.tsx` — weapon breakdown renders

### Verification
- `pnpm test` + `pnpm typecheck` + `go test -race ./...` pass
- `wails dev` → Heatmaps page: select map + demos → see KDE overlay, adjust filters/sliders
- `wails dev` → Demo viewer: stats panel shows weapon breakdown

---

## Task-to-Wave Mapping

| Task ID | Description | Wave(s) | Status |
|---------|-------------|---------|--------|
| P4-T01 | Faceit API client | W1, W2 | COMPLETE |
| P4-T02 | Faceit sync service | W2 | COMPLETE |
| P4-T03 | Faceit dashboard page | W1, W3 | COMPLETE |
| P4-T04 | Match history list | W1, W3 | COMPLETE |
| P4-T05 | Demo download | W4 | COMPLETE |
| P4-T06 | Heatmap data binding | W5 | COMPLETE |
| P4-T07 | KDE rendering | W6 | COMPLETE |
| P4-T08 | Heatmap filter controls | W7 | COMPLETE |
| P4-T09 | Per-demo stats view | W5, W7 | COMPLETE |

---

## Critical Path

```
TRACK A: W1 (bindings) → W2 (sync) → W3 (dashboard+sync) → W4 (demo download)
TRACK B: W5 (heatmap binding) → W6 (KDE+PixiJS) → W7 (filter UI+stats)
```

Longest chain: Track A (4 waves). Track B (3 waves) completes first.

---

## Key Files Reference

**Go (modify)**:
- `app.go` — wire 3 stubs + add SyncFaceitMatches, ImportMatchDemo, GetHeatmapData, GetUniqueWeapons, GetUniquePlayers, GetWeaponStats
- `types.go` — add HeatmapPoint, PlayerInfo, WeaponStat
- `app_test.go` — extend with Faceit + heatmap binding tests
- `internal/store/heatmaps_custom.go` — add GetDistinctWeapons, GetDistinctPlayers

**Go (create)**:
- `internal/faceit/sync.go` + `sync_test.go`
- `internal/faceit/download.go` + `download_test.go`

**Go (reuse, read-only)**:
- `internal/auth/faceit.go` — HTTPFaceitClient (reuse in sync)
- `internal/auth/service.go` — GetAccessToken, GetCurrentUser
- `internal/store/faceit_matches.sql.go` — generated queries
- `internal/testutil/mocks.go` — MockFaceitClient, MockKeyring
- `internal/demo/import.go` — ImportService (reuse in download)

**Frontend (modify)**:
- `frontend/src/routes/heatmaps.tsx` — replace stub
- `frontend/src/routes/dashboard.tsx` — add sync button
- `frontend/src/routes/demo-viewer.tsx` — add stats panel toggle
- `frontend/src/components/dashboard/match-list.tsx` — wire Import Demo
- `frontend/src/test/mocks/bindings.ts` — add new mocks

**Frontend (create)**:
- `frontend/src/types/heatmap.ts`
- `frontend/src/hooks/use-faceit-sync.ts`, `use-heatmap.ts`, `use-demo-download.ts`
- `frontend/src/stores/heatmap.ts`
- `frontend/src/lib/pixi/kde.ts`, `colormap.ts`
- `frontend/src/lib/pixi/layers/heatmap-layer.ts`
- `frontend/src/components/heatmap/filter-panel.tsx`, `heatmap-canvas.tsx`
- `frontend/src/components/viewer/stats-panel.tsx`

**Frontend (reuse, read-only)**:
- `frontend/src/lib/maps/calibration.ts` — worldToPixel()
- `frontend/src/lib/pixi/app.ts` — ViewerApp
- `frontend/src/lib/pixi/layers/map-layer.ts` — MapLayer (reuse in heatmap canvas)
- `frontend/src/components/viewer/viewer-canvas.tsx` — pattern reference
- `frontend/src/stores/viewer.ts` — pattern reference

---

## End-to-End Verification

After all waves:
1. `go test -race ./...` — all Go tests pass
2. `pnpm test` — all frontend tests pass
3. `pnpm typecheck` — clean
4. `wails dev` full integration:
   - Login with Faceit → "Sync Matches" → matches populate in dashboard
   - Profile card: correct ELO, level, streak
   - ELO chart: history with 30d/90d/180d/all selector
   - Match list: filter by map + result, pagination works
   - Import Demo: downloads from Faceit, appears in library
   - Heatmaps page: select map + demos → KDE overlay renders
   - Adjust filters (weapon/side/player) → heatmap updates
   - Bandwidth/opacity sliders → visual changes
   - Demo viewer: stats panel shows weapon breakdown with HS%
