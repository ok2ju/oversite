---
name: gen-test
description: Generate test files matching project TDD conventions for Go or TypeScript code
disable-model-invocation: true
---

# Generate Test

Generates a test file for a given source file or function, following project conventions.

## Arguments

- `$ARGUMENTS` — path to the source file or `package.function` to test (e.g., `internal/demo/import.go` or `frontend/src/hooks/use-demos.ts`)

## Workflow

1. Read the target source file to understand what needs testing
2. Check for existing tests in the same package/directory — match their style exactly
3. Determine the test type based on the file:

### Go Files (`**/*.go`)
- Create `*_test.go` in the same package
- Use table-driven tests with `tt` loop variable
- For DB-touching code, use `testutil.NewTestQueries(t)` from `internal/testutil/db.go` (in-memory SQLite + migrations)
- For golden files, use `testutil.CompareGolden(t, name, got)` and `testutil.LoadFixture(t, name, &v)` from `internal/testutil/golden.go`
- Include: happy path, error cases, edge cases, validation failures

### React Components (`frontend/src/components/**/*.tsx`)
- Create `__tests__/*.test.tsx` or `*.test.tsx` co-located
- Use Vitest + React Testing Library
- Mock API calls with MSW handlers
- Test: rendering, user interactions, loading/error states, accessibility

### React Hooks (`frontend/src/hooks/**/*.ts`)
- Use `renderHook()` from `@testing-library/react`
- Wrap TanStack Query hooks in `QueryClientProvider`
- Mock API calls with MSW
- Test: initial state, data fetching, mutations, error handling

### Zustand Stores (`frontend/src/stores/**/*.ts`)
- Pure unit tests — no React rendering
- Reset store state in `beforeEach`
- Test: initial state, actions, derived state, subscriptions

### Utility Functions (`**/utils/**/*.ts`)
- Pure unit tests with Vitest
- Table-driven style where applicable
- Test: normal inputs, edge cases, error inputs

4. Write the complete test file with all imports
5. Run the tests to verify they pass (for Go: `go test ./...`, for TS: `pnpm test`)

## Example

```
/gen-test internal/demo/import.go
/gen-test frontend/src/hooks/use-demos.ts
/gen-test frontend/src/stores/demo.ts
```
