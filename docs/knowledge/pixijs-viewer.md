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
- **EventLayer** — kill lines, shot tracers, smoke fades, flash bursts (lifetime-driven sprites)
- **UILayer** — scoreboard overlay, timeline markers

## Tick rendering loop

Current tick drives everything. The app buffers a range of ticks from `GetTicks(demoId, from, to)` (Wails binding) and interpolates between them at 60 FPS. Prefer pre-loading the next ~5s of ticks while rendering the current second.

## Gotchas

- **Don't recreate the app on every render.** The effect dependency array should be empty (or stable).
- **Destroy cleanly.** Missed `destroy(true)` leaks WebGL contexts; Chrome caps you at ~16.
- **Don't call React setState from the render loop.** It triggers React reconciliation on every frame.
- **PixiJS `Graphics` has no gradient strokes.** When you need a fading line (e.g., the `weapon_fire` tracer that fades toward an unknown endpoint), approximate it with stacked short segments and a per-segment `alpha`. See `drawShot` in `event-layer.ts` — 16 segments was a good balance between visual smoothness and per-shot draw cost.
- **`drawShot` has two visual paths** keyed on the shot's `hit_x`/`hit_y` (populated by the parser's shot-impact pairing pass): solid line + impact dot when an exact endpoint is known, gradient ray otherwise. The fallback length (`SHOT_TRACER_LENGTH`) is in world units; PixiJS clips to viewport so an over-long ray is harmless on small maps but wastes draw calls — keep it modest (~2000 units, ~half a typical CS map).
- **Effect durations must be capped to round-end.** Pass `rounds` to `EventLayer.setEvents(events, rounds)`. Without the cap, long-lived effects (smoke ~18 s, molotov fire ~7 s, mid-flight trajectories thrown right before round end) persist into the next round's freeze. CS2 itself wipes world state at round transitions; the viewer mirrors that via the `cap()` helper inside `buildScheduled`, which clamps every pushed effect's `durationTicks` to its containing round's `end_tick`.
- **`extra_data.entity_id` is a JSON number, not a string.** Go's `Entity.ID()` is `int`; it survives Wails as a JS `number`. Frontend code that does `typeof id === "string"` will silently never match. Use the `entityKey()` helper in `event-layer.ts` to normalize before lookups.
- **`useLoadoutSnapshot` has a hand-rolled diff (`sameLoadout`).** The hook caches the most recent loadout per `steam_id` and only triggers a re-render when fields differ across a 250 ms poll. When you add a new field to `TickData` that the team bars need to react to (active weapon ammo, etc.), update `sameLoadout` too — otherwise the new field arrives in the buffer but never reaches the UI. The PixiJS sprite path doesn't have this problem because `PlayerLayer.update` runs every frame against fresh `TickData` from the buffer.
- **`EventLayer` uses two parallel pools.** `GraphicsPool` for vector drawing and `SpritePool` for textured icons (currently only the grenade head). Effects that need both — like `grenade_traj` — acquire one of each on activation and release both on expiry. When acquiring a pooled `Sprite`, reset `sprite.texture = null` first: the drawer's `if (sprite.texture !== texture)` branch reapplies `GRENADE_ICON_HEIGHT / texture.height` scale only when the texture identity changes, so a pooled sprite that still holds the previous grenade's texture would skip the scale recalc and render at the wrong size.

## Grenade trajectory rendering

In-flight grenades render as a colored dot lerped along a waypoint list (throw → bounces → termination), with a faint trail of the path traveled so far. Pattern in `event-layer.ts`:

1. **First pass** indexes events by `entity_id`: smoke expirations, bounce points, and terminations (`grenade_detonate` / `smoke_start` / `fire_start` / `decoy_start`).
2. **Second pass** — on each `grenade_throw`, look up the matching termination and materialize `[throw, ...bounces, termination]` as `TrajectoryWaypoint[]`. Orphaned throws (no termination — truncated demo) are skipped to avoid forever-flying icons.
3. **Drawer** uses `interpolateTrajectory(waypoints, currentTick)` to find the active segment and `progress(a.tick, b.tick, currentTick)` to lerp position. The trail is a `Graphics` polyline (color = `grenadeColor(weapon)` — Molotov → orange, Decoy → purple, others reuse the detonation color) through completed waypoints + the lerped head. The icon is a `Sprite` showing the weapon's `/equipment/<name>.svg` texture (resolved via `getWeaponTexture(weapon)` — same async cache used by `PlayerSprite`). When the texture isn't yet loaded — or the weapon name has no `weapon-icons.ts` entry — the drawer falls back to the legacy colored dot so the grenade is never invisible.

`drawEffect` was extended with a `tickOffset` parameter because `state.progress` (0..1) loses the precision needed to reconstruct `currentTick` for interpolation.
