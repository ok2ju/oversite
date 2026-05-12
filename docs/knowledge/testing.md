# Testing Patterns

**Related:** [architecture/testing](../architecture/testing.md) · CLAUDE.md "Test-Writing Discipline"

## Read before you write

Before writing any test: **open an existing test in the same directory/package** and match its patterns (imports, wrappers, mock style). Don't invent a new wrapper when one exists.

## Reusable utilities

### Frontend (React / Vitest)

- `renderWithProviders()` — `src/test/render.tsx`. Provides `QueryClientProvider` and `ThemeProvider`. **Always use this**; never build a raw `QueryClientProvider` wrapper in a test file.
- MSW handlers — `src/test/msw/handlers.ts`. Use for any HTTP mocking the test environment needs.
- PixiJS mocks — `src/test/mocks/pixi.ts`.
- Wails binding mocks — `src/test/mocks/bindings.ts`.

### Go

- `testutil.NewTestDB(t)` / `testutil.NewTestQueries(t)` — `internal/testutil/db.go`. In-memory SQLite with migrations applied. Never open a test DB manually.
- Golden files: `testutil.CompareGolden(t, name, got)` and `testutil.LoadFixture(t, name, &v)` from `internal/testutil/golden.go`. Update with `go test -update`. Fixtures live in `testdata/`.

## Run tests immediately

Stop hook runs tests automatically at turn end — but don't rely on it. Run your test yourself after writing it (`go test ./path/...` or `pnpm vitest --run <file>`) and don't move to the next file until it passes.

## Go tests: always `-race`

Use `go test -race ./...` (not `go test ./...`) for unit tests. The race detector is cheap on this size of codebase and catches goroutine bugs the Stop hook would otherwise miss.

## Layer-specific conventions

| Layer | Convention |
|-------|------------|
| Go services | Table-driven tests with named subcases |
| Go parser | Golden-file regression tests |
| sqlc queries | Run against `testutil.NewTestQueries` — real SQL, not mocks |
| Zustand stores | Pure state + action tests in Vitest |
| React components | `renderWithProviders` + RTL `screen.getBy…` |
| Wails binding hooks | Mock the auto-generated TS function from `bindings.ts` |
| PixiJS rendering | TDD the logic; don't unit-test the render loop — use Playwright screenshot tests |
| E2E | Written **after** the feature works, not TDD |

## Gotchas

### Golden-file `-update` flag order

`go test -update <path>` is wrong — Go parses `-update` as a package pattern when it precedes the path. Put the flag after the package:

```bash
go test ./internal/demo/analysis/ -update        # path first, flag last
go test ./internal/demo/analysis/ -run Foo -update
```

`go test -update ./...` silently runs nothing — Go interprets `-update` as a package pattern and finds zero packages. Bumping `AnalysisVersion` is a common trigger for stale goldens (every `MatchSummaryRow.version` flips); regenerate with the above.

### Loose binding-mock shapes let new fields land without test churn

`mockAppBindings.GetMistakeTimeline` in `src/test/mocks/bindings.ts` declares its return shape inline with only the fields the test files actually inspect. When the real `MistakeEntry` gained `duel_id` (slice 13), no existing test had to change — the mock didn't enforce the new field, and rendering code keys on `m.duel_id != null` which is false for `undefined` too. Keep mock shapes narrow on purpose so additive field changes don't cascade into fixture rewrites.

### Querying portaled dialogs that echo on-page text

Radix portals (`AlertDialog`, `Dialog`, `Popover`) render into `document.body`, so `screen.getByText(...)` searches both the open dialog **and** the underlying page. When the dialog repeats data already visible — e.g. a row's filename inside a "Remove this demo?" confirmation — a plain `getByText` matches two nodes and throws. Scope to the dialog:

```tsx
const dialog = screen.getByRole("alertdialog", { name: "Remove this demo?" })
expect(within(dialog).getByText(/awesome-clutch\.dem/)).toBeInTheDocument()
```

The portaled surface is reachable via `getByRole("alertdialog" | "dialog")` with the title as `name`; pair with `within()` for any further assertions.

### `testutil.NewTestDB` is single-connection — don't hold a cursor across statements

`internal/testutil/db.go:26` calls `db.SetMaxOpenConns(1)` (matches production's WAL-mode single-writer policy). Holding a `*sql.Rows` from `db.QueryContext` and then issuing another statement on the **same DB** — `db.PrepareContext`, `db.QueryRowContext`, etc. — deadlocks: the cursor owns the only connection and the second call blocks waiting for it. Symptom is a test that hangs for the full timeout with a goroutine stuck in `database/sql.(*DB).conn → connectionOpener`.

Fix: materialize the cursor into a slice before the next statement, or run both inside an explicit `*sql.Tx`.

```go
// WRONG — second statement on db deadlocks while rows is open.
rows, _ := db.QueryContext(ctx, "SELECT ... FROM kills")
defer rows.Close()
for rows.Next() {
    db.QueryRowContext(ctx, "SELECT COUNT(*) FROM visibility WHERE ...")  // hangs
}

// RIGHT — drain the cursor first.
var kills []kill
rows, _ := db.QueryContext(ctx, "SELECT ... FROM kills")
for rows.Next() { /* scan into kills */ }
_ = rows.Close()
for _, k := range kills {
    db.QueryRowContext(ctx, "SELECT COUNT(*) FROM visibility WHERE ...")
}
```

Hit during the Phase 1 visibility test's kill-correlation query — 5-minute timeout before the cause was obvious from the goroutine dump.
