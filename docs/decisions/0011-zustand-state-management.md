# ADR-0011: Zustand for Frontend State Management

**Date:** 2026-04-12
**Status:** Accepted

## Context

The frontend needs client-side state management for multiple domains: viewer playback state, strategy board tool state, UI preferences, demo library filters, and Faceit data. The state library must also bridge React UI controls to the PixiJS render loop running outside React ([ADR-0001](0001-pixijs-outside-react.md)).

### Alternatives considered

| Approach | Why rejected |
|----------|-------------|
| **Redux Toolkit** | Powerful but heavy for this use case. Boilerplate (slices, reducers, actions, selectors) is overkill for a single-user desktop app. Redux's middleware model (thunks, sagas) adds complexity we don't need — Wails bindings are already async. |
| **Jotai** | Atomic state model is elegant but makes it harder to subscribe to state slices from outside React (PixiJS bridge). Zustand's `subscribe()` with a selector is a simpler bridge pattern. |
| **React Context + useReducer** | Native React, but triggers re-renders on any state change in the context tree. Performance-critical viewer state (current tick, player positions) would cause unnecessary React reconciliation, defeating the purpose of PixiJS outside React. |
| **MobX** | Observable-based reactivity works well but adds a paradigm shift (decorators, observables, reactions). Zustand is simpler and more predictable for a codebase that already separates rendering concerns. |

## Decision

Use **Zustand** with per-domain stores:

| Store | Domain | Key State |
|-------|--------|-----------|
| `viewerStore` | Demo playback | Current tick, speed, playing/paused, selected round |
| `stratStore` | Strategy board | Active tool, selected elements, undo stack |
| `uiStore` | App chrome | Theme, sidebar collapsed, modal state |
| `faceitStore` | Faceit data | Sync status, selected profile |
| `demoStore` | Demo library | Filters, sort order, selected demos |

### PixiJS Bridge Pattern

PixiJS code subscribes to Zustand stores outside React using `store.subscribe(selector, callback)`. This fires only when the selected slice changes, avoiding React reconciliation entirely:

```ts
viewerStore.subscribe(
  (s) => s.currentTick,
  (tick) => pixiApp.updatePlayerPositions(tick)
);
```

React components use the same stores via `useViewerStore((s) => s.currentTick)` with automatic selector-based re-rendering.

## Consequences

### Positive

- Minimal boilerplate — stores are plain objects with functions, no reducers/actions/dispatch
- `subscribe()` API enables clean PixiJS bridge without React involvement
- Selector-based subscriptions prevent unnecessary re-renders in both React and PixiJS
- Easy to test — stores are plain functions, testable without React rendering
- Small bundle size (~1KB gzipped)

### Negative

- No built-in devtools as rich as Redux DevTools (zustand/middleware provides basic logging)
- No enforced patterns — easy to create inconsistent store shapes without discipline
- `subscribe()` callbacks must be cleaned up manually (in `useEffect` cleanup or PixiJS destroy)
- Splitting state across many stores can make cross-store coordination verbose (mitigated by keeping stores domain-scoped)
