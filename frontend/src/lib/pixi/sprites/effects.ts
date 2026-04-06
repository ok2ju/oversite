// Tick durations
export const SMOKE_DURATION_TICKS = 1152
export const SMOKE_FADE_IN_TICKS = 32
export const SMOKE_FADE_OUT_TICKS = 64
export const FLASH_DURATION_TICKS = 19
export const HE_DURATION_TICKS = 32
export const HE_EXPAND_TICKS = 13
export const KILL_DURATION_TICKS = 192
export const BOMB_FLASH_INTERVAL_TICKS = 32
export const BOMB_DEFUSE_TICKS = 640
export const BOMB_DEFUSE_KIT_TICKS = 320

// World-space radii
export const SMOKE_RADIUS = 144
export const HE_RADIUS = 350
export const FLASH_RADIUS = 200
export const BOMB_ICON_RADIUS = 30

// Colors (PixiJS hex)
export const COLOR_SMOKE = 0x888888
export const COLOR_HE = 0xff3333
export const COLOR_FLASH = 0xffff00
export const COLOR_KILL = 0xff0000
export const COLOR_BOMB_PLANT = 0xff4444
export const COLOR_BOMB_DEFUSE = 0x4444ff

export interface EffectState {
  active: boolean
  alpha: number
  radius: number
  progress: number
}

const INACTIVE: EffectState = { active: false, alpha: 0, radius: 0, progress: 0 }

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
    alpha = 1 - (tickOffset - HE_EXPAND_TICKS) / (HE_DURATION_TICKS - HE_EXPAND_TICKS)
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

export function computeBombPlantState(tickOffset: number): EffectState {
  if (tickOffset < 0) return INACTIVE

  const phase = tickOffset % BOMB_FLASH_INTERVAL_TICKS
  const alpha = phase < BOMB_FLASH_INTERVAL_TICKS / 2 ? 1.0 : 0.3

  return {
    active: true,
    alpha,
    radius: BOMB_ICON_RADIUS,
    progress: 0,
  }
}

export function computeBombDefuseState(tickOffset: number, hasKit: boolean): EffectState {
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
