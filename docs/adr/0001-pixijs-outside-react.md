# ADR-0001: Run PixiJS Outside the React Render Tree

**Date:** 2026-03-31
**Status:** Accepted

## Context

The 2D demo viewer renders 10 player sprites, weapon trails, grenade arcs, and map overlays at 60 FPS. React's reconciliation cycle is designed for UI updates at human-interaction frequency (clicks, typing), not per-frame canvas mutations. Wrapping PixiJS objects in React components (e.g., via react-pixi) would force every frame update through React's diffing, causing dropped frames and unnecessary re-renders across the component tree.

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **react-pixi / @pixi/react** | Bridges PixiJS into React's render cycle. Fine for static or low-frequency scenes, but adds overhead on every frame when animating 10+ sprites at 64 ticks/sec. |
| **Pure Canvas 2D API** | No React coupling, but loses PixiJS's WebGL batching, sprite sheets, and built-in interaction system. Would require reimplementing a lot. |
| **Three.js (2D mode)** | Overkill for a 2D top-down viewer. Larger bundle, steeper learning curve, no meaningful benefit over PixiJS for this use case. |

## Decision

PixiJS Application is instantiated imperatively inside a React `useEffect`. React renders a container `<div>`; PixiJS owns the `<canvas>` and its own render loop. Zustand `subscribe()` bridges React UI controls (play/pause, speed, tick scrubbing) to PixiJS state without triggering React re-renders.

```
React world                         PixiJS world
┌──────────────┐   subscribe()   ┌──────────────────┐
│ Zustand store │ ──────────────▶│ PixiJS Application│
│ (viewer state)│                │ (own render loop) │
└──────────────┘                └──────────────────┘
       ▲                                │
       │  setState (UI controls)        │  reads tick data
       │                                ▼
  React components              Canvas (60 FPS)
```

## Consequences

### Positive

- Viewer runs at consistent 60 FPS regardless of React component tree complexity
- React DevTools and PixiJS DevTools work independently without interfering
- Zustand subscriptions are surgical — only the values PixiJS cares about trigger updates

### Negative

- Two mental models: React for UI, imperative PixiJS for canvas — contributors need to understand both
- Cleanup in `useEffect` return must be thorough (destroy Application, remove listeners, cancel animation frames)
- Cannot use React's declarative patterns (JSX, hooks) for PixiJS scene graph objects
