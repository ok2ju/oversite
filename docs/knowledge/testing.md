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

### Querying portaled dialogs that echo on-page text

Radix portals (`AlertDialog`, `Dialog`, `Popover`) render into `document.body`, so `screen.getByText(...)` searches both the open dialog **and** the underlying page. When the dialog repeats data already visible — e.g. a row's filename inside a "Remove this demo?" confirmation — a plain `getByText` matches two nodes and throws. Scope to the dialog:

```tsx
const dialog = screen.getByRole("alertdialog", { name: "Remove this demo?" })
expect(within(dialog).getByText(/awesome-clutch\.dem/)).toBeInTheDocument()
```

The portaled surface is reachable via `getByRole("alertdialog" | "dialog")` with the title as `name`; pair with `within()` for any further assertions.
