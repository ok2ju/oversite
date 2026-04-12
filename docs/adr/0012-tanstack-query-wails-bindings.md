# ADR-0012: TanStack Query for Wails Binding Responses

**Date:** 2026-04-12
**Status:** Accepted

## Context

The frontend calls Go backend functions via auto-generated Wails bindings (async TypeScript functions). These calls return data that components need to display — demo lists, round details, tick data, Faceit stats. We need a strategy for caching, refetching, loading states, and error handling around these binding calls.

Using TanStack Query (a "server state" library) for local function calls is unconventional — it was designed for HTTP APIs. This ADR explains why it's still the right choice.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **Direct calls + useState** | Each component manages its own loading/error/data state. Leads to duplicated fetch logic, no caching (multiple components calling the same binding redundantly), and inconsistent loading UX. |
| **Zustand stores for all data** | Zustand handles client state well, but using it for async data means manually implementing: loading states, error states, cache invalidation, background refetch, deduplication. This is exactly what TanStack Query does. |
| **SWR** | Similar concept to TanStack Query but less mature query invalidation and no built-in mutation support. TanStack Query's `useMutation` + `invalidateQueries` pattern maps well to Wails binding calls that modify data. |

## Decision

Use **TanStack Query v5** to wrap all Wails binding calls that fetch data. Mutations use `useMutation` with cache invalidation.

### Pattern

```ts
// Query: fetch data
const { data: demos } = useQuery({
  queryKey: ['demos'],
  queryFn: () => ListDemos(),  // Wails binding
});

// Mutation: modify data + invalidate cache
const importDemo = useMutation({
  mutationFn: (path: string) => ImportDemo(path),
  onSuccess: () => queryClient.invalidateQueries({ queryKey: ['demos'] }),
});
```

### Why it works for local bindings

TanStack Query's value comes from its **state machine** (idle → loading → success/error), **deduplication** (multiple components using the same queryKey share one call), and **cache invalidation** (mutations trigger refetch of affected queries). These benefits apply regardless of whether the data source is an HTTP API or a local function call.

Key behaviors:
- **Stale-while-revalidate**: Show cached data immediately, refetch in background
- **Deduplication**: Multiple components mounting with `queryKey: ['demos']` share one `ListDemos()` call
- **Automatic refetch**: On window focus (useful for Faceit data that may update externally)
- **Structural sharing**: Only triggers re-renders if data actually changed

### Where NOT to use TanStack Query

- **Tick data for the viewer**: High-frequency data (128 ticks/second) uses a custom buffer in the playback engine, not TanStack Query. The buffer fetches ahead and manages its own lifecycle tied to the PixiJS render loop.
- **Client-only state**: UI preferences, playback controls, tool selection — these live in Zustand stores.

## Consequences

### Positive

- Consistent loading/error/data pattern across all data-fetching components
- Automatic cache management — no manual "fetch on mount, clear on unmount" boilerplate
- `invalidateQueries` after mutations keeps the UI consistent without manual refetch orchestration
- DevTools available for debugging query state during development
- Familiar API for developers who've used it with REST/GraphQL

### Negative

- Adds a dependency (~12KB gzipped) for wrapping local function calls — arguably over-engineered for pure local data
- `queryKey` management requires discipline (stale keys = stale data, over-broad invalidation = unnecessary refetches)
- Developers unfamiliar with the library may find the caching behavior surprising (e.g., stale data shown before refetch completes)
- Window focus refetching is unnecessary for most local data (disabled per-query where not useful)
