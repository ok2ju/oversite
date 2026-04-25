# Architecture — Wails Bindings Specification

> **Siblings:** [overview](overview.md) · [structure](structure.md) · [components](components.md) · [data-flows](data-flows.md) · [database](database.md) · [crosscutting](crosscutting.md) · [testing](testing.md)
>
> **See also:** [product binding overview](../product/wails-bindings.md) for the full binding method table, and the knowledge wiki entry on [Wails bindings patterns](../knowledge/wails-bindings.md).

---

## Wails Bindings Specification

Wails bindings replace the REST API from the web version. Go struct methods decorated with Wails annotations are automatically available as TypeScript functions in the frontend.

### Binding Architecture

```
Go struct method                    Auto-generated TS function
─────────────────                   ──────────────────────────
func (a *App) ImportDemo(           import { ImportDemo } from
  path string,                        '../../wailsjs/go/main/App';
) (*Demo, error)
                                    const demo = await ImportDemo(path);
```

Wails generates the TypeScript bindings at build time from Go method signatures. The generated files live in `frontend/wailsjs/`.

### Error Handling Convention

All binding methods return `(result, error)` in Go. In TypeScript, errors become rejected promises:

```typescript
try {
  const demo = await ImportDemo(path);
} catch (err) {
  // err contains the Go error message
  toast.error(`Import failed: ${err}`);
}
```

### Event System

For long-running operations (demo parsing), Wails runtime events provide progress updates:

```go
// Go: emit progress events
runtime.EventsEmit(a.ctx, "demo:parse:progress", DemoProgress{
    DemoID:  id,
    Percent: 42,
    Phase:   "parsing_ticks",
})
```

```typescript
// TypeScript: subscribe to progress
import { EventsOn } from '../../wailsjs/runtime/runtime';

EventsOn('demo:parse:progress', (progress: DemoProgress) => {
  setParseProgress(progress.Percent);
});
```

### Full Binding Reference

See [product/wails-bindings.md](../product/wails-bindings.md) for the complete binding method table.
