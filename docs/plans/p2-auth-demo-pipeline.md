# Phase 2: Auth & Demo Pipeline -- Sprint Plan

> **Note (2026-05-06):** Faceit / auth scope was removed from the project after
> this plan was executed. The auth tasks (loopback OAuth, keyring storage, auth
> service, AuthProvider, login page) and references to the `users` /
> `faceit_matches` tables describe historical direction. The demo-pipeline
> portion (import, parse, library UI) is still active. See `docs/log.md`.

## Context

Phase 1 established the desktop app skeleton: Wails scaffold, SQLite with WAL mode, sqlc-generated queries (56 functions across 9 files), Vite+React SPA with all routes, shadcn/ui components, Zustand stores, and full test infrastructure on both sides. Every Wails binding in `app.go` is a stub returning `errNotImplemented`. The frontend renders mock/placeholder data.

Phase 2 delivers the two foundational data pipelines the entire app depends on:

1. **Authentication** -- Faceit OAuth via loopback redirect (RFC 8252), OS keychain token storage, user persistence in SQLite. This unlocks every feature that requires a logged-in user.
2. **Demo ingestion** -- Import `.dem` files from disk, parse them with `demoinfocs-golang` v5, and populate SQLite with tick data (~1.28M rows/demo), game events, rounds, and player stats. This unlocks the Phase 3 viewer, Phase 4 heatmaps, and Phase 5 strategy features.

At phase completion, a user can sign in with Faceit, import a demo file (or folder), watch it parse with progress feedback, and see it appear in their demo library with correct metadata.

**Key advantage:** The web-era `backend/internal/demo/` package contains 1,432 lines of working parser code (`parser.go` 642L, `ingest.go` 309L, `stats.go` 273L, `grenade_extractor.go` 156L, `validate.go` 52L). Additionally, `backend/internal/auth/` has 333 lines of OAuth + Faceit client code. These need adaptation (PostgreSQL/UUID types to SQLite/int64, remove Redis state store, add loopback flow) but core logic is battle-tested and can be ported rather than rewritten.

---

## Dependency Graph

```
                     ┌──────────────┐
                     │  P2-S01      │
                     │  Parser Spike│ (concurrent, 1 day)
                     └──────┬───────┘
                            │ informs design
                            ▼
┌──────────┐         ┌──────────────┐
│ P2-T01   │         │  P2-T05      │
│ OAuth    │         │  Demo Import │
│ [M]      │         │  [M]         │
└────┬─────┘         └──┬──┬──┬─────┘
     │                   │  │  │
┌────┴─────┐    ┌───────┘  │  └────────┐
│ P2-T02   │    │          │           │
│ Keyring  │    │     ┌────┴─────┐  ┌──┴──────┐
│ [S]      │    │     │ P2-T10   │  │ P2-T11  │
└────┬─────┘    │     │ Demo UI  │  │ Folder  │
     │          │     │ [M]      │  │ Import  │
     │          │     └──────────┘  │ [S]     │
┌────┴──────┐   │                   └─────────┘
│ P2-T03    │   │
│ Auth Svc  │   ▼
│ [M]       │  ┌──────────────┐
└────┬──────┘  │  P2-T06      │
     │         │  Parser Core │◄── spike output
┌────┴──────┐  │  [XL]        │
│ P2-T04    │  └──┬──┬──┬─────┘
│ AuthProv. │     │  │  │
│ [M]       │     │  │  │
└───────────┘     │  │  │
            ┌─────┘  │  └──────┐
            ▼        ▼         ▼
      ┌──────────┐ ┌────────┐ ┌──────────┐
      │ P2-T07   │ │ P2-T08 │ │ P2-T09   │
      │ Ticks    │ │ Events │ │ Rounds   │
      │ [L]      │ │ [L]    │ │ [M]      │
      └──────────┘ └────────┘ └──────────┘
```

## Execution Order

| Wave | Tasks | Parallel? | Notes |
|------|-------|-----------|-------|
| 0 | P2-S01 (spike) | **COMPLETE** | Done 2026-04-15. Findings in `docs/spike-parser-findings.md`. |
| 1 | P2-T01 (OAuth), P2-T02 (Keyring), P2-T05 (Demo Import) | **COMPLETE** | Done 2026-04-15. All three implemented with 65 passing tests. |
| 2 | P2-T03 (Auth Service), P2-T10 (Demo UI), P2-T11 (Folder Import) | **COMPLETE** | Done 2026-04-15. All three implemented with full test coverage. |
| 3 | P2-T04 (AuthProvider), P2-T06 (Parser Core) | **COMPLETE** | Done 2026-04-15. T04 was completed in W2. T06: parser ported with incendiary fix + progress callback, 138 tests passing. |
| 4 | P2-T07 (Ticks), P2-T08 (Events), P2-T09 (Rounds) | **COMPLETE** | Done 2026-04-15. All three implemented in parallel with 166 tests passing across the demo package. |

**Critical path:** S01 -> T05 -> T06 -> T07/T08/T09 (demo pipeline).

---

## Task Plans

### P2-S01: Demo Parser Spike (Pre-Phase, Concurrent) -- COMPLETE

**Status:** Done (2026-04-15). Findings in `docs/spike-parser-findings.md`.

**Summary:** Parser from `backend/internal/demo/` compiles and runs against 3 real Faceit CS2 demos (394-862 MB) with zero code changes. demoinfocs-golang v5.1.2 API is stable. All demos pass performance targets (<10s parse, <500 MB heap). Identified incendiary/molotov handler gap (~25% orphaned grenade throws) and MaxUploadSize too low (500 MB vs 860+ MB decompressed demos). Both documented as action items for P2-T06 and P2-T05.

**Deliverables completed:**
- Tick interval confirmed: 4 (default) -- no performance concern
- Memory/perf baselines: 6.5s/+118 MB (862 MB demo), 3.1s/-68 MB (394 MB), 3.8s/+12 MB (454 MB)
- CS2 edge cases inventoried: overtime, warmup, world kills, orphaned grenades
- API changes needed: none (zero changes to compile); gaps to add: incendiary/molotov handler, raise MaxUploadSize
- Spike code: `cmd/spike-parser/main.go`, `internal/demo/` (parser copied from backend)

---

### P2-T01: Implement Loopback OAuth Flow -- COMPLETE

**Status:** Done (2026-04-15). Files: `internal/auth/pkce.go`, `internal/auth/oauth.go` + tests. 32 tests passing.

**Why:** RFC 8252 loopback redirect: start temp HTTP listener on random port, open system browser to Faceit auth URL, capture callback code, exchange for tokens with PKCE.

**Key decisions:**
- PKCE (RFC 7636): 32 random bytes base64url-encoded as verifier, SHA-256 as challenge
- Listener binds to `127.0.0.1:0` (OS assigns port)
- `pkg/browser` (already indirect dep in go.mod) opens system browser
- Token exchange: POST `https://api.faceit.com/auth/v1/oauth/token` with `grant_type=authorization_code`
- No client secret (PKCE replaces it for public clients)
- 120s timeout on user completing browser auth
- Callback page serves minimal HTML: "Authentication successful! You can close this tab."
- **Reference:** `backend/internal/auth/oauth.go` (238L) for Faceit endpoint URLs and token exchange logic

**Implementation plan:**

1. **Create `internal/auth/pkce.go`:**
   - `GenerateCodeVerifier() (string, error)` -- 32 random bytes, base64url-no-pad
   - `GenerateCodeChallenge(verifier string) string` -- SHA-256, base64url-no-pad
   - Tests: table-driven for length (43-128 chars), determinism, no `+`/`/`/`=` chars

2. **Create `internal/auth/oauth.go`:**
   - `type OAuthConfig struct` -- `ClientID`, `AuthURL`, `TokenURL`, `RedirectURIBase`
   - `type TokenResponse struct` -- `AccessToken`, `RefreshToken`, `ExpiresIn`, `TokenType`
   - `StartLoopbackFlow(ctx context.Context, cfg OAuthConfig) (*TokenResponse, error)`:
     a. Generate PKCE verifier + challenge
     b. `net.Listen("tcp", "127.0.0.1:0")` for random port
     c. Build auth URL with query params (`response_type=code`, `client_id`, `redirect_uri`, `code_challenge`, `code_challenge_method=S256`, `scope=openid profile email membership`)
     d. Open system browser
     e. HTTP handler captures `?code=` from GET `/callback`, sends to channel
     f. Shut down server
     g. Exchange code for tokens via POST
   - `exchangeCode(ctx, cfg, code, verifier, redirectURI string) (*TokenResponse, error)` -- POST with `application/x-www-form-urlencoded`

3. **Tests (`internal/auth/oauth_test.go`):**
   - Use `httptest.NewServer` as fake Faceit token endpoint
   - Test full flow: start `StartLoopbackFlow`, programmatically GET `http://127.0.0.1:{port}/callback?code=test-code`, verify exchange called with correct params
   - Test: timeout (120s) produces clear error
   - Test: invalid code response (400)

**Key files:**
- `internal/auth/pkce.go`, `internal/auth/pkce_test.go`
- `internal/auth/oauth.go`, `internal/auth/oauth_test.go`

---

### P2-T02: Implement Keychain Token Storage -- COMPLETE

**Status:** Done (2026-04-15). Files: `internal/auth/keyring.go`, `internal/auth/keyring_real.go` + tests. 6 tests passing.

**Why:** Refresh tokens must persist across app restarts in OS-encrypted storage. The `Keyring` interface and `MockKeyring` already exist in `internal/testutil/mocks.go` -- this task implements the real one and wraps it in a `TokenStore`.

**Key decisions:**
- Service name: `oversite-faceit-auth`
- Key for refresh token: `refresh-token`; key for user ID: `user-id`
- Access token NOT stored in keychain -- held in memory only (short-lived)
- `TokenStore` wraps `testutil.Keyring` interface (not `go-keyring` directly) for testability

**Implementation plan:**

1. **Create `internal/auth/keyring.go`:**
   - `type RealKeyring struct{}` implementing `testutil.Keyring`
   - Delegates to `keyring.Set/Get/Delete`, maps `keyring.ErrNotFound` to `testutil.ErrKeyNotFound`
   - `type TokenStore struct` -- holds `testutil.Keyring` + service name
   - Methods: `SaveRefreshToken(token)`, `GetRefreshToken()`, `DeleteRefreshToken()`, `SaveUserID(id)`, `GetUserID()`, `Clear()`

2. **Tests (`internal/auth/keyring_test.go`):**
   - All tests use `testutil.NewMockKeyring()` (no real keychain)
   - Save + retrieve round-trip, get non-existent returns `ErrKeyNotFound`, `Clear()` removes both keys

**New dependency:** `github.com/zalando/go-keyring`

**Key files:** `internal/auth/keyring.go`, `internal/auth/keyring_test.go`

---

### P2-T03: Create Auth Service -- COMPLETE

**Status:** Done (2026-04-15). Files: `internal/auth/faceit.go`, `internal/auth/service.go`, `internal/auth/service_test.go`, `app.go` wiring. 7 tests passing.

**Why:** Orchestration layer tying OAuth, keychain, Faceit API, and SQLite. Login flow: OAuth -> token exchange -> fetch Faceit profile -> upsert user -> store refresh token. Startup flow: check keychain -> lookup user in SQLite -> refresh access token.

**Key decisions:**
- `AuthService` holds: `TokenStore`, `OAuthConfig`, `FaceitClient` (testutil interface), `*store.Queries`, in-memory `accessToken` + `currentUser`
- `GetCurrentUser()`: check keychain for stored user ID -> lookup SQLite -> refresh token if possible -> return user or nil (nil = not logged in, NOT an error)
- `Logout()`: clear keychain, nil out in-memory state
- **Reference:** `backend/internal/auth/faceit_client.go` (95L) for Faceit API HTTP calls. Port to implement real `FaceitClient`

**Implementation plan:**

1. **Create `internal/auth/faceit.go`:**
   - `type HTTPFaceitClient struct` -- implements `testutil.FaceitClient`
   - `GetPlayer(ctx, "me")` with `Authorization: Bearer {token}` for profile fetch
   - `GetPlayerHistory()` and `GetMatchDetails()` for future Faceit sync
   - **Source:** adapt from `backend/internal/auth/faceit_client.go`

2. **Create `internal/auth/service.go`:**
   ```go
   type AuthService struct {
       oauth       OAuthConfig
       tokens      *TokenStore
       faceit      testutil.FaceitClient
       queries     *store.Queries
       accessToken string
       currentUser *store.User
       mu          sync.RWMutex
   }
   ```
   - `Login(ctx) (*store.User, error)` -- StartLoopbackFlow -> store tokens -> fetch profile -> upsert user
   - `GetCurrentUser(ctx) (*store.User, error)` -- check memory -> check keychain -> refresh -> return or nil
   - `Logout(ctx) error` -- clear keychain + memory

3. **Wire to `app.go`:**
   - Add `authService *auth.AuthService` field to `App`
   - `Startup()`: init DB, create `AuthService` with real deps
   - `GetCurrentUser()` -> `authService.GetCurrentUser()`
   - `LoginWithFaceit()` -> `authService.Login()`
   - Add `Logout()` binding

4. **Tests (`internal/auth/service_test.go`):**
   - Use `testutil.NewMockKeyring()`, `testutil.MockFaceitClient{}`, `testutil.NewTestQueries(t)`
   - Test Login: mock OAuth result, mock Faceit profile, verify user created in SQLite + refresh token in keyring
   - Test GetCurrentUser with stored session: pre-populate keyring + SQLite, verify user returned
   - Test GetCurrentUser with no session: empty keyring, verify nil (no error)
   - Test Logout: verify keyring cleared, memory nil

**Key files:**
- `internal/auth/faceit.go`, `internal/auth/service.go`, `internal/auth/service_test.go`
- `app.go` (wire AuthService, implement bindings)

---

### P2-T04: Wire AuthProvider + Login Page to Real Backend -- COMPLETE

**Status:** Done (2026-04-15). Implemented during Wave 2. AuthProvider wraps all routes, login page has loading/error states, header has logout button with user nickname display. 16 auth-related tests passing (auth-provider, login, header).

**Why:** `AuthProvider` and login page already exist from P1 and call `GetCurrentUser()`/`LoginWithFaceit()`. This task wires them to real bindings, adds logout support, and handles edge cases (loading during OAuth, errors, token refresh).

**Key decisions:**
- With loopback OAuth, the `/callback` route in the frontend is never hit directly (callback goes to Go temp listener). Repurpose or remove.
- `LoginWithFaceit()` is a long-running call (blocks while user completes browser auth, up to 120s). Frontend needs a loading state.
- Add `Logout()` binding call + logout button in header/settings

**Implementation plan:**

1. **Update `App.tsx`:** wrap `RootLayout` routes in `<AuthProvider>` (currently only used in tests)
2. **Update `routes/login.tsx`:** add loading spinner while `LoginWithFaceit()` runs, error toast on failure, retry button
3. **Update `auth-provider.tsx`:** add `logout()` method to context, calls `Logout()` binding + invalidates auth query + redirects
4. **Add logout button** to header or settings
5. **Update `test/mocks/bindings.ts`:** add `Logout` mock
6. **Tests:** existing AuthProvider tests should pass as-is; add tests for login loading state, error display, logout flow

**Key files:**
- `frontend/src/App.tsx`
- `frontend/src/routes/login.tsx`
- `frontend/src/components/providers/auth-provider.tsx`
- `frontend/src/test/mocks/bindings.ts`
- `app.go` (add Logout binding)

---

### P2-T05: Implement Demo Import Binding -- COMPLETE

**Status:** Done (2026-04-15). Files: `internal/demo/validate.go` (MaxUploadSize raised to 1GB), `internal/demo/import.go` + tests. 33 tests passing.

**Why:** Entry point for all demo data. Validate `.dem` file format, register in SQLite with status `imported`. Web-era `backend/internal/demo/validate.go` (52L) has working validation code that ports directly.

**Key decisions:**
- Magic bytes: CS2 uses `PBDEMS2\x00`, CS:GO uses `HL2DEMO\x00` -- support both (from `backend/internal/demo/validate.go`)
- Max file size: 1GB (raised from 500MB -- spike showed Faceit `.dem.zst` files decompress to 400-860+ MB)
- File dialog: Wails `runtime.OpenFileDialog()` with `.dem` filter
- Demo record: `store.CreateDemo()` with `status=imported`, `map_name=""` (populated after parse)
- Requires current user ID for `user_id` FK -- get from `authService.currentUser`

**Implementation plan:**

1. **Create `internal/demo/validate.go`** -- port from `backend/internal/demo/validate.go`:
   - `ValidateExtension()`, `ValidateSize()`, `ValidateMagicBytes()`, magic byte constants, sentinel errors
   - Direct copy -- no PostgreSQL/UUID deps to remove

2. **Create `internal/demo/import.go`:**
   - `type ImportService struct` -- `*store.Queries`, `*sql.DB`
   - `ImportFile(ctx, filePath string, userID int64) (*store.Demo, error)`:
     a. Validate extension, stat file, validate size, read magic bytes
     b. `store.CreateDemo()` with `status=imported`
   - `ValidateFile(filePath string) error` -- public validation without DB insert

3. **Wire to `app.go`:**
   - `ImportDemoFile()`: `runtime.OpenFileDialog()` -> validate -> import -> return
   - `DeleteDemo(id)`: `queries.DeleteDemo()` (cascade handles related data)

4. **Tests:** table-driven for extension, size, magic bytes (port from `backend/internal/demo/validate_test.go`); integration test with `testutil.NewTestDB()` for import + delete cascade

**Key files:**
- `internal/demo/validate.go`, `internal/demo/validate_test.go`
- `internal/demo/import.go`, `internal/demo/import_test.go`
- `app.go` (wire ImportDemoFile, DeleteDemo)

---

### P2-T06: Implement Demo Parser Core (XL) -- COMPLETE

**Status:** Done (2026-04-15). Files: `internal/demo/parser.go` (ported with FireGrenadeStart handler + ProgressFunc callback), `internal/demo/stats.go` (direct copy), `internal/demo/grenade_extractor.go` (added `fire_start` to detonationTypes). 99 new tests (48 parser + 32 stats + 19 grenade extractor), 138 total in demo package.

**Why:** Highest-complexity task -- foundation for every data-dependent feature. Reads `.dem` and extracts: match metadata, player positions (sampled), kills, grenades, bombs, rounds.

**Key advantage:** `backend/internal/demo/parser.go` (642L) is a fully working parser. The desktop version needs three adaptations:
1. Replace `uuid.UUID` demo IDs with `int64`
2. Remove PostgreSQL type wrappers (`pqtype.NullRawMessage` -> `string` JSON)
3. Add Wails progress events for real-time UI feedback

The parser does NOT write to SQLite -- it returns a `ParseResult`. Tasks T07/T08/T09 handle ingestion.

**Spike findings (2026-04-15):**

The spike validated the web-era parser against 3 real Faceit CS2 demos (394-862 MB decompressed). Key results:

| Demo | Size | Rounds | Parse Time | Heap Delta | Result |
|------|------|--------|------------|------------|--------|
| de_ancient (OT marathon) | 862 MB | 54 (30 OT) | 6.5s | +118 MB | PASS |
| de_ancient | 394 MB | 25 (1 OT) | 3.1s | -68 MB* | PASS |
| de_dust2 | 454 MB | 30 (6 OT) | 3.8s | +12 MB | PASS |

*Negative delta = GC reclaimed prior demo's allocations.

Findings that inform implementation:
1. **Incendiary/Molotov gap:** `parser.go` has no handler for `events.FireGrenadeStart` (or equivalent incendiary event). `grenade_extractor.go` `detonationTypes` only matches `grenade_detonate`, `smoke_start`, `decoy_start`. This causes ~25% of grenade throws to be orphaned (no matching detonation). **Action:** Add incendiary/molotov event handler and include the corresponding detonation type in `detonationTypes`.
2. **MaxUploadSize too low:** `validate.go` caps at 500 MB, but Faceit `.dem.zst` files decompress to 400-860+ MB. **Action:** Raise to 1 GB for desktop use (updated in P2-T05).
3. **Tick interval confirmed:** 4 (default) works well. No performance concern at this interval.
4. **demoinfocs-golang v5.1.2 stable:** Zero API changes needed vs. web-era code. All types compile as-is.
5. **Overtime + warmup detection works correctly** out of the box.

**Key decisions:**
- Tick interval: 4 (every 4th tick -- ~16 samples/sec at 64 tick). **Confirmed by spike.**
- Skip warmup: true (default)
- Include bots: false (default)
- Panic recovery: wrap `ParseToEnd()` in defer/recover (already done in web-era)
- Truncated demos: partial data returned, not a crash

**Implementation plan:**

1. **Create `internal/demo/parser.go`** -- port from `backend/internal/demo/parser.go` (642L):
   - Types: `ParseResult`, `MatchHeader`, `RoundData`, `TickSnapshot`, `GameEvent`
   - `DemoParser` struct with `Option` functions (`WithTickInterval`, `WithSkipWarmup`, `WithIncludeBots`)
   - `Parse(r io.Reader) (*ParseResult, error)` with all event handlers:
     - `CDemoFileHeader` -> map name
     - `IsWarmupPeriodChanged` -> warmup tracking
     - `RoundStart`/`RoundEnd` -> round data with score tracking
     - `FrameDone` -> tick sampling
     - `Kill` -> kill events with extra data (headshot, penetration, flash assist, wallbang)
     - `GrenadeProjectileThrow`, `HeExplode`, `FlashExplode`, `SmokeStart`, etc.
     - **`FireGrenadeStart` (incendiary/molotov)** -- missing from web-era, identified by spike
     - `BombPlanted`, `BombDefused`, `BombExplode`
   - Add progress callback: `ProgressFunc func(stage string, percent float64)`

2. **Create `internal/demo/stats.go`** -- port from `backend/internal/demo/stats.go` (273L):
   - `CalculatePlayerRoundStats()` for per-player-per-round K/D/A/damage
   - No changes needed (pure primitive types)

3. **Create `internal/demo/grenade_extractor.go`** -- port from `backend/internal/demo/grenade_extractor.go` (156L):
   - `ExtractGrenadeLineups()`
   - **Add `"fire_start"` to `detonationTypes`** to correlate incendiary/molotov throws

4. **Golden file tests (`internal/demo/parser_test.go`):**
   - Place 1-2 small test `.dem` files in `testdata/`
   - Parse -> serialize `ParseResult` to JSON -> `testutil.CompareGolden()`
   - Test cases: normal match, overtime, truncated demo
   - Unit tests for helpers: `shouldSampleTick`, `isOvertime`, `teamSideString`

**New dependency:** `github.com/markus-wa/demoinfocs-golang/v5`

**Key files:**
- `internal/demo/parser.go`, `internal/demo/parser_test.go`
- `internal/demo/stats.go`, `internal/demo/stats_test.go`
- `internal/demo/grenade_extractor.go`, `internal/demo/grenade_extractor_test.go`
- `testdata/*.dem`, `testdata/*.golden`

**Acceptance criteria:**
- Golden file tests pass for 3+ demo scenarios
- Warmup excluded, bots excluded from ticks
- Truncated demos produce partial results (not crash)
- Memory < 500MB for 100MB demos
- Parse time < 10s for average demo

---

### P2-T07: Batch Insert Ticks into SQLite -- COMPLETE

**Status:** Done (2026-04-15). Files: `internal/demo/ingest.go`, `internal/demo/ingest_test.go`, `internal/demo/convert.go` (shared helpers). TickIngester with 10K batch size, idempotent delete-then-insert in single tx. 5 tests (basic ingestion, idempotent, empty, bool conversion, chunk logic).

**Why:** ~1.28M tick rows per demo. One-at-a-time inserts take minutes; batched transactions (10K rows/tx) bring this to seconds. **Reference:** `backend/internal/demo/ingest.go` (309L) `TickIngester` pattern, adapted from PostgreSQL COPY to SQLite INSERT.

**Key decisions:**
- Batch size: 10,000 rows/transaction (tunable)
- Use `store.Queries.WithTx(tx)` for transactional inserts via sqlc
- Idempotent: delete existing tick_data for demo before inserting
- Bool-to-int: `IsAlive` is `bool` in parser but `int64` in SQLite (0/1)
- Progress events: `demo:ingest:progress` with `{demoId, stage: "ticks", percent}`
- Optimization option: multi-value INSERT if per-row is too slow

**Implementation plan:**

1. **Create `internal/demo/ingest.go`:**
   - `type TickIngester struct` -- `*sql.DB`, `batchSize int`
   - `Ingest(ctx, demoID int64, ticks []TickSnapshot) (int64, error)`:
     a. Delete existing: `DeleteTickDataByDemoID(ctx, demoID)`
     b. Chunk ticks into batches
     c. Per batch: `BeginTx` -> loop `InsertTickData` -> `Commit` -> emit progress

2. **Tests (`internal/demo/ingest_test.go`):**
   - Integration test: create user + demo, generate synthetic 1000-tick data, ingest, verify row count + sample values
   - Idempotent re-ingestion: ingest twice, no duplicates
   - Verify `GetTickDataByRange` returns correct data after ingestion

**Key files:** `internal/demo/ingest.go`, `internal/demo/ingest_test.go`

---

### P2-T08: Insert Game Events into SQLite -- COMPLETE

**Status:** Done (2026-04-15). Files: `internal/demo/events.go`, `internal/demo/events_test.go`. IngestGameEvents with single tx, roundMap FK resolution, JSON ExtraData serialization, nullable string handling. 7 tests (basic, idempotent, empty, extra_data JSON, nullable fields, resolveRoundID, marshalExtraData).

**Why:** Kills, grenades, bombs drive the viewer event layer, heatmaps, and per-round stats. **Reference:** `IngestGameEvents()` from `backend/internal/demo/ingest.go`, adapted from PostgreSQL UUIDs to int64 IDs.

**Key decisions:**
- Events inserted within a single transaction per demo
- Round ID mapping: requires `roundNumber -> roundID` map from T09. Pipeline orchestrator calls T09 before T08.
- `extra_data`: marshal `map[string]interface{}` to JSON string (empty = `"{}"`)
- Idempotent: delete existing events before inserting

**Implementation plan:**

1. **Create `internal/demo/events.go`:**
   - `IngestGameEvents(ctx, tx *sql.Tx, demoID int64, events []GameEvent, roundMap map[int]int64) (int, error)`:
     a. Delete existing events
     b. Convert each event to `store.CreateGameEventParams` (NullString for optional fields, JSON marshal for extra_data)
     c. Insert each event

2. **Tests (`internal/demo/events_test.go`):**
   - Integration: create user + demo + round, build roundMap, insert test events (kill, grenade, bomb), verify fields + extra_data JSON

**Key files:** `internal/demo/events.go`, `internal/demo/events_test.go`

---

### P2-T09: Insert Rounds + Player Rounds into SQLite -- COMPLETE

**Status:** Done (2026-04-15). Files: `internal/demo/rounds.go`, `internal/demo/rounds_test.go`. IngestRounds calculates player stats before tx, inserts rounds + player_rounds in single tx, returns roundMap for T08. 5 tests (basic, player rounds with stats verification, idempotent, empty, roundMap keys).

**Why:** Rounds provide structural backbone for the viewer (round selector, scoreboard) and per-player stats. **Reference:** `IngestRounds()` from `backend/internal/demo/ingest.go` + `CalculatePlayerRoundStats()` from `stats.go`.

**Key decisions:**
- Insert rounds first, capture `roundID` from RETURNING, build `roundNumber -> roundID` map
- Then insert player_rounds for each round
- Pipeline orchestrator calls T09 BEFORE T08 (T08 needs the roundMap)
- `first_kill`/`first_death`: `bool` in parser -> `int64` (0/1) in SQLite
- Single transaction for all rounds + player_rounds

**Implementation plan:**

1. **Create `internal/demo/rounds.go`:**
   - `IngestRounds(ctx, db *sql.DB, demoID int64, result *ParseResult) (map[int]int64, error)`:
     a. Calculate player stats via `CalculatePlayerRoundStats()`
     b. Begin tx, delete existing rounds (cascades to player_rounds)
     c. For each round: `CreateRound()` -> get ID -> store in roundMap -> insert player_rounds
     d. Commit, return roundMap

2. **Tests (`internal/demo/rounds_test.go`):**
   - Integration: synthetic ParseResult with 3 rounds, 2 players each; verify DB records, roundMap entries, cascade delete

**Key files:** `internal/demo/rounds.go`, `internal/demo/rounds_test.go`

---

### P2-T10: Wire Demo Library UI to Real Backend -- COMPLETE

**Status:** Done (2026-04-15). DropZone integrated, folder import button wired, parse progress hook + DemoCard progress bar implemented, `ImportDemoByPath` binding added for drag-and-drop. 4 new tests added.

**Why:** `DemoList`, `DemoCard`, `UploadDialog` exist from P1 with mock data. This task wires to real bindings, adds drag-and-drop, folder import, and parse progress UI.

**Key decisions:**
- `useDemos()` already calls `ListDemos()` + has `refetchInterval` polling for active imports. Works as-is.
- `useImportDemo()` already calls `ImportDemoFile()`. Works as-is.
- New: drag-and-drop via Wails `OnFileDrop` runtime event
- New: folder import button calling `ImportFolder()` binding
- New: parse progress bar via `EventsOn("demo:parse:progress")` -> `useDemoStore.importProgress`

**Implementation plan:**

1. **Add `frontend/src/components/demos/drop-zone.tsx`:** Wails `OnFileDrop` listener, `.dem` filter, visual indicator
2. **Add folder import button** to demos page, calls `ImportFolder()` binding
3. **Add parse progress hook:** subscribe to `demo:parse:progress` events, update `useDemoStore.importProgress`
4. **Update `DemoCard`:** show progress bar when status is `parsing`
5. **Tests:** existing component tests pass with mocks; add drop-zone and progress tests

**Key files:**
- `frontend/src/routes/demos.tsx`
- `frontend/src/components/demos/drop-zone.tsx` (new)
- `frontend/src/hooks/use-demos.ts` (add progress subscription)
- `frontend/src/components/demos/demo-card.tsx` (progress bar)
- `frontend/src/test/mocks/bindings.ts` (add ImportFolder mock)

---

### P2-T11: Implement Folder Import Binding -- COMPLETE

**Status:** Done (2026-04-15). Files: `internal/demo/folder.go` with `FolderProgressFunc` callback, `app.go` wiring with Wails event emission. 6 tests passing.

**Why:** Users have replay folders with many `.dem` files. Folder import recursively scans and imports all valid demos.

**Implementation plan:**

1. **Create `internal/demo/folder.go`:**
   - `type FolderImportResult struct` -- `Imported []store.Demo`, `Errors []FolderImportError`
   - `ImportFolder(ctx, dirPath string, userID int64) (*FolderImportResult, error)`:
     a. `filepath.WalkDir` collecting `.dem` paths
     b. Call `ImportFile()` for each; successes -> `Imported`, failures -> `Errors`
     c. Emit progress events: `{total, current, fileName}`

2. **Wire to `app.go`:** `runtime.OpenDirectoryDialog()` -> `importService.ImportFolder()`

3. **Tests:** temp directory with `.dem` + non-`.dem` files; verify recursive scan, invalid file handling, empty directory

**Key files:** `internal/demo/folder.go`, `internal/demo/folder_test.go`, `app.go`

---

## Pipeline Orchestrator

Tasks T06-T09 produce independent functions needing orchestration. The pipeline ensures correct order:

**Create `internal/demo/pipeline.go`:**
```
ProcessDemo(ctx, demoID int64) error:
  1. Update demo status to "parsing"
  2. Open .dem file by file_path from DB
  3. Parse: parser.Parse(file) -> ParseResult
  4. Ingest rounds: IngestRounds() -> roundMap  (T09 before T08)
  5. Ingest events: IngestGameEvents(roundMap)
  6. Ingest ticks: TickIngester.Ingest()
  7. Update demo via UpdateDemoAfterParse() -- status="ready", map_name, tick_rate, duration
  8. On error at any step: UpdateDemoStatus() to "error"
```

Wire to `app.go`: called automatically after `ImportFile()` succeeds. Also exposable as `ParseDemo(demoID)` binding for re-parse.

---

## New Dependencies

| Package | Purpose | Task |
|---------|---------|------|
| `github.com/zalando/go-keyring` | OS keychain access | T02 |
| `github.com/markus-wa/demoinfocs-golang/v5` | CS2 demo parser | S01, T06 |
| `github.com/pkg/browser` (already indirect) | Open system browser for OAuth | T01 |

**Note:** `golang.org/x/oauth2` is NOT needed -- the token exchange is a single POST, raw `net/http` suffices.

---

## Reuse Strategy

**Port with type adaptations (PostgreSQL -> SQLite):**
- `backend/internal/demo/parser.go` (642L) -> `internal/demo/parser.go`
- `backend/internal/demo/ingest.go` (309L) -> `internal/demo/ingest.go` + `events.go` + `rounds.go`
- `backend/internal/demo/stats.go` (273L) -> `internal/demo/stats.go`
- `backend/internal/demo/grenade_extractor.go` (156L) -> `internal/demo/grenade_extractor.go`
- `backend/internal/demo/validate.go` (52L) -> `internal/demo/validate.go` (copy verbatim)
- `backend/internal/auth/faceit_client.go` (95L) -> `internal/auth/faceit.go`

**Port tests:**
- `backend/internal/demo/validate_test.go` -> `internal/demo/validate_test.go`
- `backend/internal/demo/parser_test.go` -> reference for golden file test structure

**Do NOT port (replaced by desktop patterns):**
- `backend/internal/auth/middleware.go` (no HTTP middleware in Wails)
- `backend/internal/auth/session.go` (no Redis sessions -- keychain instead)
- `backend/internal/auth/redis_state_store.go` (no Redis)

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| Faceit rejects `http://localhost` redirect with random port | Blocks all auth | Test during spike. Faceit docs support localhost. Fallback: fixed port (19847) with port-in-use handling. |
| `demoinfocs-golang` v5 API changes since web-era code | Delays T06 by 1-2 days | **RESOLVED by spike:** v5.1.2 API is stable, zero code changes needed. |
| 1.28M tick rows cause slow SQLite inserts | Perf below target | Multi-value INSERT batches. `PRAGMA synchronous=NORMAL` during import. Web-era batching pattern proven. |
| macOS Keychain access prompt confuses users | UX friction on first login | Add clear UI messaging: "macOS may ask for permission to store your login securely." |
| Large demos (300MB+) cause OOM during parse | Crash | **RESOLVED by spike:** 454 MB demo used only +12 MB heap; 394 MB demo +50 MB. Streaming parser confirmed. MaxUploadSize raised to 1 GB. |
| Token refresh fails silently | User appears logged out | Distinguish "no session" from "expired session" in `GetCurrentUser()`. Show appropriate UI message. |

---

## Verification Plan

After all 11 tasks complete:

### Auth Smoke Test
1. `wails dev` -- app launches, redirected to `/login` (not logged in)
2. Click "Sign in with Faceit" -- system browser opens to Faceit
3. Complete login -- browser shows "Authentication successful"
4. App redirects to `/dashboard` with nickname displayed
5. Quit + restart -- still logged in (keychain persistence)
6. Logout -- redirected to `/login`

### Demo Import Smoke Test
1. Navigate to `/demos`, click "Import Demo" -- file dialog opens
2. Select valid `.dem` file -- appears in list with status `imported`
3. Transitions to `parsing` (progress bar), then `ready`
4. Card shows map name, duration, file size
5. Delete with confirmation -- removed from list

### Folder Import Smoke Test
1. Click "Import Folder" -- directory picker opens
2. Select folder with mixed files -- only `.dem` files imported
3. Progress shown during scan

### Data Integrity Verification
```sql
SELECT count(*) FROM rounds WHERE demo_id = ?;       -- matches expected round count
SELECT count(*) FROM tick_data WHERE demo_id = ?;     -- ~1.28M rows
SELECT count(*) FROM game_events WHERE demo_id = ?;   -- hundreds of events
SELECT count(DISTINCT steam_id) FROM player_rounds
  WHERE round_id IN (SELECT id FROM rounds WHERE demo_id = ?);  -- 10 players
```

### Automated Test Verification
```bash
go test -race ./internal/auth/... ./internal/demo/...
cd frontend && pnpm test
make lint && make typecheck
```
