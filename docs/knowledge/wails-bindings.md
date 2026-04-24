# Wails Bindings

**Related:** [[sqlc-workflow]] · [product/wails-bindings](../product/wails-bindings.md) (method catalog) · [architecture/wails-bindings](../architecture/wails-bindings.md) (architecture details)

## Go method → TS function

Methods on the `App` struct in `app.go` with exported names become async TypeScript functions under `frontend/wailsjs/go/main/App.ts`. Wails regenerates these bindings on `wails dev` / `wails build`.

```go
func (a *App) ImportDemo(path string) (*Demo, error) { ... }
```
becomes
```typescript
export function ImportDemo(path: string): Promise<Demo>;
```

## Error handling

Go `error` → rejected TS promise with the error's message string. Wrap calls in `try/catch` on the frontend, surface errors via toast. The frontend never sees Go error types; it sees strings.

## Progress events for long operations

`runtime.EventsEmit(ctx, "demo:parse:progress", payload)` on the Go side; `EventsOn('demo:parse:progress', handler)` on the TS side. Use for demo parse, demo download, batch import.

Event names in use:
- `demo:parse:progress` — tick parsing, phases: `validating` / `parsing_ticks` / `parsing_events` / `complete`
- `faceit:demo:download:progress` — Faceit demo download (bytes transferred)
- `faceit:sync:progress` — batch match sync

## Synchronous processing

There is no background worker. Demo parsing, Faceit sync, and demo downloads all run in-process on the Go backend. If the user kicks off two parses at once, they queue naturally on the SQLite single connection (see [[sqlite-wal]]).

## TanStack Query wrapping

Wrap bindings in `useQuery` with stable query keys so TanStack Query handles caching and background refetch. Invalidate on mutations (import, delete, sync).

```typescript
const { data } = useQuery({ queryKey: ['demos'], queryFn: () => ListDemos({}) });
```
