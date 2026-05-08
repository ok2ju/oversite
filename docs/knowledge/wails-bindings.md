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

`runtime.EventsEmit(ctx, "demo:parse:progress", payload)` on the Go side; `EventsOn('demo:parse:progress', handler)` on the TS side. Use for demo parse and batch import.

Event names in use:
- `demo:parse:progress` — tick parsing, phases: `parsing` / `complete` / `error`
- `demo:folder:progress` — batch folder import (current/total file count)

## Synchronous processing

There is no background worker. Demo parsing runs in-process on the Go backend. If the user kicks off two parses at once, they queue naturally on the SQLite single connection (see [[sqlite-wal]]).

## TanStack Query wrapping

Wrap bindings in `useQuery` with stable query keys so TanStack Query handles caching and background refetch. Invalidate on mutations (import, delete, sync).

```typescript
const { data } = useQuery({ queryKey: ['demos'], queryFn: () => ListDemos({}) });
```

## Patterns

### List vs. detail type split

The viewer needs the full `Demo` (path, size, etc.); the library list only needs a name + counts. `types.go` exposes both `Demo` and `DemoSummary`; `ListDemos` returns `[]DemoSummary`, `GetDemoByID` returns `*Demo`. Saves ~10–20 KB on a 100-row page and avoids leaking `FilePath` to anything that doesn't need it.

### `json.RawMessage` for opaque blobs

`GameEvent.ExtraData` is typed `json.RawMessage` so the SQLite TEXT column passes through to Wails' JSON encoder verbatim — no per-row `map[string]any` allocation on the Go side. The frontend decodes once. Use this whenever the binding doesn't need to inspect the JSON.

### Coalesce progress emits

`runtime.EventsEmit` is cheap individually but cumulative. `app.go emitProgress` skips emits within 100 ms of the previous emit *for the same stage*; errors and terminal stages (`complete`, `error`) bypass the throttle. Defensive — current callers (round boundaries + 10K-frame heartbeat) wouldn't hit it, but a future per-tick caller would generate 64K+ emits/match.

### Cancellable context plumbed through

`Startup` wraps the Wails ctx in `context.WithCancel`; `Shutdown` cancels before `db.Close` so in-flight DB work bails out instead of fighting a closed pool. Pass the derived ctx into any new long-running binding.

### Serialize bulk file imports

`fileImportMu` (in `app.go`) wraps `ImportService.ImportFile` so a 10-zst drag-and-drop only runs one zstd decompression at a time. Each decoder window holds tens of MB; in parallel they could spike RAM by 500+ MB on top of whatever parse was already running. Parse continues to serialize on its own `parseMu` — copy of file 2 *can* overlap with parse of file 1 (different mutex), which is intentional: full end-to-end serialization would block the Wails caller for the duration of a parse and lose synchronous error returns.
