# PixiJS Viewer

**Related:** [[wails-bindings]] · [ADR-0001](../decisions/0001-pixijs-outside-react.md) · [architecture/components](../architecture/components.md)

## Core pattern: PixiJS lives outside React

PixiJS Application is **not** rendered by React. The flow is:

1. React renders an empty container `<div>` with a ref.
2. A `useEffect` hook instantiates `new PIXI.Application()` attached to that ref.
3. PixiJS owns its own render loop (WebGL / requestAnimationFrame).
4. Cleanup in the effect's return: `app.destroy(true)`.

This avoids React re-render overhead on every frame. The trade-off: React doesn't know about PixiJS state, so the bridge is explicit.

## Zustand bridge

Controls (play/pause, speed, current tick) live in `viewerStore` (Zustand). The PixiJS app subscribes via `viewerStore.subscribe((state, prev) => ...)` inside the `useEffect`. Unsubscribe on cleanup.

React UI components call Zustand actions (`setPlaying(true)`) without touching PixiJS directly.

## Layers inside the PixiJS app

- **MapLayer** — radar image at the base
- **PlayerLayer** — circles + view cones; redrawn per tick
- **EventLayer** — kill lines, smoke fades, flash bursts (lifetime-driven sprites)
- **UILayer** — scoreboard overlay, timeline markers

## Tick rendering loop

Current tick drives everything. The app buffers a range of ticks from `GetTicks(demoId, from, to)` (Wails binding) and interpolates between them at 60 FPS. Prefer pre-loading the next ~5s of ticks while rendering the current second.

## Gotchas

- **Don't recreate the app on every render.** The effect dependency array should be empty (or stable).
- **Destroy cleanly.** Missed `destroy(true)` leaks WebGL contexts; Chrome caps you at ~16.
- **Don't call React setState from the render loop.** It triggers React reconciliation on every frame.
