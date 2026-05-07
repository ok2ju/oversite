// Tick durations
export const SMOKE_DURATION_TICKS = 1152
export const SMOKE_FADE_IN_TICKS = 32
export const SMOKE_FADE_OUT_TICKS = 64
export const FLASH_DURATION_TICKS = 19
export const HE_DURATION_TICKS = 32
export const HE_EXPAND_TICKS = 13
export const KILL_DURATION_TICKS = 192
export const SHOT_DURATION_TICKS = 12
export const BOMB_FLASH_INTERVAL_TICKS = 32
export const BOMB_DEFUSE_TICKS = 640
export const BOMB_DEFUSE_KIT_TICKS = 320

// World-space radii
export const SMOKE_RADIUS = 144
export const HE_RADIUS = 350
export const FLASH_RADIUS = 200
export const BOMB_ICON_RADIUS = 30
// Fallback length for unpaired shots (misses or wall hits — the demo format
// does not expose non-player impact points). For shots that hit a player, the
// parser writes hit_x/hit_y so the tracer ends at the exact impact instead.
export const SHOT_TRACER_LENGTH = 2000

// Colors (PixiJS hex)
export const COLOR_SMOKE = 0x888888
export const COLOR_HE = 0xff3333
export const COLOR_FLASH = 0xffff00
export const COLOR_KILL = 0xff0000
export const COLOR_SHOT = 0xffcc00
export const COLOR_BOMB_PLANT = 0xff4444
export const COLOR_BOMB_DEFUSE = 0x4444ff
export const COLOR_MOLOTOV = 0xff8800
export const COLOR_DECOY = 0xaa44ff
export const COLOR_FIRE = 0xff6600
export const COLOR_GRENADE_DEFAULT = 0xffffff

// Grenade in-flight icon (pixel-space, doesn't scale with map)
export const GRENADE_ICON_RADIUS = 4
export const GRENADE_TRAIL_ALPHA = 0.35

// Molotov / Incendiary burn duration (~7s × 64 tick rate)
export const FIRE_DURATION_TICKS = 448
export const FIRE_FADE_IN_TICKS = 16
export const FIRE_FADE_OUT_TICKS = 64
export const FIRE_RADIUS = 120

export interface EffectState {
  active: boolean
  alpha: number
  radius: number
  progress: number
}

const INACTIVE: EffectState = {
  active: false,
  alpha: 0,
  radius: 0,
  progress: 0,
}

export function computeSmokeState(
  tickOffset: number,
  durationTicks = SMOKE_DURATION_TICKS,
): EffectState {
  if (tickOffset < 0 || tickOffset >= durationTicks) return INACTIVE

  let alpha: number
  if (tickOffset < SMOKE_FADE_IN_TICKS) {
    alpha = tickOffset / SMOKE_FADE_IN_TICKS
  } else if (tickOffset < durationTicks - SMOKE_FADE_OUT_TICKS) {
    alpha = 1
  } else {
    alpha = (durationTicks - tickOffset) / SMOKE_FADE_OUT_TICKS
  }

  return {
    active: true,
    alpha,
    radius: SMOKE_RADIUS,
    progress: tickOffset / durationTicks,
  }
}

export function computeHEState(tickOffset: number): EffectState {
  if (tickOffset < 0 || tickOffset >= HE_DURATION_TICKS) return INACTIVE

  let radius: number
  let alpha: number

  if (tickOffset < HE_EXPAND_TICKS) {
    radius = (tickOffset / HE_EXPAND_TICKS) * HE_RADIUS
    alpha = 1
  } else {
    radius = HE_RADIUS
    alpha =
      1 - (tickOffset - HE_EXPAND_TICKS) / (HE_DURATION_TICKS - HE_EXPAND_TICKS)
  }

  return {
    active: true,
    alpha,
    radius,
    progress: tickOffset / HE_DURATION_TICKS,
  }
}

export function computeFlashState(tickOffset: number): EffectState {
  if (tickOffset < 0 || tickOffset >= FLASH_DURATION_TICKS) return INACTIVE

  const alpha = 1 - tickOffset / FLASH_DURATION_TICKS

  return {
    active: true,
    alpha,
    radius: FLASH_RADIUS,
    progress: tickOffset / FLASH_DURATION_TICKS,
  }
}

export function computeShotState(tickOffset: number): EffectState {
  if (tickOffset < 0 || tickOffset >= SHOT_DURATION_TICKS) return INACTIVE

  const alpha = 1 - tickOffset / SHOT_DURATION_TICKS

  return {
    active: true,
    alpha,
    radius: 0,
    progress: tickOffset / SHOT_DURATION_TICKS,
  }
}

export function computeKillState(tickOffset: number): EffectState {
  if (tickOffset < 0 || tickOffset >= KILL_DURATION_TICKS) return INACTIVE

  const fadeStart = Math.floor((KILL_DURATION_TICKS * 2) / 3)
  const alpha =
    tickOffset < fadeStart
      ? 1
      : 1 - (tickOffset - fadeStart) / (KILL_DURATION_TICKS - fadeStart)

  return {
    active: true,
    alpha,
    radius: 0,
    progress: tickOffset / KILL_DURATION_TICKS,
  }
}

export function computeBombPlantState(
  tickOffset: number,
  durationTicks: number,
): EffectState {
  if (tickOffset < 0 || tickOffset >= durationTicks) return INACTIVE

  const phase = tickOffset % BOMB_FLASH_INTERVAL_TICKS
  const alpha = phase < BOMB_FLASH_INTERVAL_TICKS / 2 ? 1.0 : 0.3

  return {
    active: true,
    alpha,
    radius: BOMB_ICON_RADIUS,
    progress: 0,
  }
}

export function computeBombDefuseState(
  tickOffset: number,
  hasKit: boolean,
): EffectState {
  const duration = hasKit ? BOMB_DEFUSE_KIT_TICKS : BOMB_DEFUSE_TICKS
  if (tickOffset < 0 || tickOffset >= duration) return INACTIVE

  const progress = tickOffset / duration

  return {
    active: true,
    alpha: 1,
    radius: BOMB_ICON_RADIUS,
    progress,
  }
}

export function worldRadiusToPixel(worldRadius: number, scale: number): number {
  return worldRadius / scale
}

// Clamped lerp factor — fraction of the way through [start, end] at `current`,
// pinned to [0, 1] outside the range. Lifted from the Healey article;
// consumed by grenade trajectory + fire interpolation.
export function progress(start: number, end: number, current: number): number {
  if (end <= start) return 1
  const pct = (current - start) / (end - start)
  if (pct < 0) return 0
  if (pct > 1) return 1
  return pct
}

export interface TrajectoryWaypoint {
  tick: number
  x: number
  y: number
}

// Linear-interpolate a position along ordered trajectory waypoints. Waypoints
// must be sorted ascending by tick. Returns the endpoint position when
// `currentTick` falls outside the bounds — callers gate by tickOffset before
// drawing, so out-of-range queries shouldn't happen during normal playback.
export function interpolateTrajectory(
  waypoints: readonly TrajectoryWaypoint[],
  currentTick: number,
): { x: number; y: number; segmentIndex: number } {
  if (waypoints.length === 0) {
    return { x: 0, y: 0, segmentIndex: 0 }
  }
  if (waypoints.length === 1 || currentTick <= waypoints[0].tick) {
    return { x: waypoints[0].x, y: waypoints[0].y, segmentIndex: 0 }
  }
  const last = waypoints.length - 1
  if (currentTick >= waypoints[last].tick) {
    return {
      x: waypoints[last].x,
      y: waypoints[last].y,
      segmentIndex: last - 1,
    }
  }

  let i = 0
  while (i < last && waypoints[i + 1].tick <= currentTick) {
    i++
  }
  const a = waypoints[i]
  const b = waypoints[i + 1]
  const t = progress(a.tick, b.tick, currentTick)
  return {
    x: a.x + (b.x - a.x) * t,
    y: a.y + (b.y - a.y) * t,
    segmentIndex: i,
  }
}

// In-flight grenade icon — fully visible while the projectile lives, then
// vanishes when it detonates (no fade; the detonation effect takes over).
export function computeGrenadeTrajectoryState(
  tickOffset: number,
  durationTicks: number,
): EffectState {
  if (tickOffset < 0 || tickOffset >= durationTicks) return INACTIVE
  return {
    active: true,
    alpha: 1,
    radius: GRENADE_ICON_RADIUS,
    progress: durationTicks > 0 ? tickOffset / durationTicks : 0,
  }
}

// Molotov / Incendiary burn — fade in fast, hold, fade out at end. Mirrors
// the smoke shape, just shorter and oranger.
export function computeFireState(
  tickOffset: number,
  durationTicks = FIRE_DURATION_TICKS,
): EffectState {
  if (tickOffset < 0 || tickOffset >= durationTicks) return INACTIVE

  let alpha: number
  if (tickOffset < FIRE_FADE_IN_TICKS) {
    alpha = tickOffset / FIRE_FADE_IN_TICKS
  } else if (tickOffset < durationTicks - FIRE_FADE_OUT_TICKS) {
    alpha = 1
  } else {
    alpha = (durationTicks - tickOffset) / FIRE_FADE_OUT_TICKS
  }

  return {
    active: true,
    alpha,
    radius: FIRE_RADIUS,
    progress: tickOffset / durationTicks,
  }
}

// Map demoinfocs weapon names to the rendering color used for the in-flight
// icon and trail. Falls back to white for unrecognized weapons.
export function grenadeColor(weapon: string | null | undefined): number {
  switch (weapon) {
    case "HE Grenade":
      return COLOR_HE
    case "Flashbang":
      return COLOR_FLASH
    case "Smoke Grenade":
      return COLOR_SMOKE
    case "Molotov":
    case "Incendiary Grenade":
      return COLOR_MOLOTOV
    case "Decoy Grenade":
      return COLOR_DECOY
    default:
      return COLOR_GRENADE_DEFAULT
  }
}
