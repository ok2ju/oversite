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

### Auto-generated `wailsjs/go/main/App.{d.ts,js}` need manual edits when not running `wails dev`

The TS bindings only regenerate during `wails dev` / `wails build`. When a session adds Go App methods without spinning up the dev server (most of ours), three files need a hand-edit so the frontend can `import` the new binding immediately:

- `frontend/wailsjs/go/main/App.d.ts` — add the typed declaration.
- `frontend/wailsjs/go/main/App.js` — add the runtime `window['go'][...]` thunk.
- `frontend/wailsjs/go/models.ts` — needed whenever frontend code references `main.NewType` directly. The hook-via-cast pattern (`(... as Promise<T>)`) used by `useMistakeTimeline` / `useDuelTimeline` skips this entirely, but lane builders / shared `@/lib` modules that import `import type { main } from "@wailsjs/go/models"` do not.

A subsequent `wails dev` will overwrite these files; that's fine, the generator produces the same shape.

`wails generate module` is supposed to regenerate these without booting the dev server, but in practice it fails silently — `exit status 1` with no stderr, even at `-v 2 -nocolour`. When it errors, hand-edit the three files above (the PreToolUse hook only blocks lock files and sqlc-generated `*.sql.go`, not anything under `frontend/wailsjs/`). Phase 4 (contact moments) hit this; Phase 2's types had to be back-filled into `models.ts` retroactively as a result.

### Go string-type aliases do not emit a TS type

`type ContactOutcome string` in `types.go` is inlined as bare `string` in the generated `models.ts`. If frontend code needs the namespaced reference (`main.ContactOutcome`) — e.g. to constrain a `ContactMarker.outcome` field — add `export type ContactOutcome = string;` inside the `main` namespace by hand. Wails won't write it for you, and string-typing the field as plain `string` loses the documentation value.

A wails regen — or a formatter/hook that triggers one mid-session — will silently strip the hand-added alias. Symptom: `tsc` explodes with `TS2694: 'main' has no exported member 'ContactOutcome'` across every file that imports it (six in Phase 5). Quick fix: `git checkout HEAD -- frontend/wailsjs/go/models.ts` if the regen wasn't intended, then re-add the alias if the regen was. Check `git diff frontend/wailsjs/` before assuming typecheck is broken from your own edits.

### Diagnostic-folder binding pair convention

For any app-managed folder the user might need to attach to a bug report, expose three bindings: `XxxDir()` (returns the absolute path), `OpenXxxFolder()` (Reveals it via `logging.Reveal`), and a settings toggle/setter pair if behavior is configurable. Existing instances:

- `LogsDir` / `OpenLogsFolder` — `errors.txt` and rotated backups.
- `ProfilesDir` / `OpenProfilesFolder` — heap-watchdog pprof dumps (`{AppData}/oversite/profiles/heap-{demoID}-{ts}.pprof`).
- `GetTolerateEntityErrors` / `SetTolerateEntityErrors` — flips the parser's `IgnorePacketEntitiesPanic` flag for users who need partial parses on corrupt demos. State lives on the `App` struct (no DB migration) and applies to the next parse.
