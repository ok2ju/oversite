import { describe, it, expect } from "vitest"
import {
  computeSmokeState,
  computeHEState,
  computeFlashState,
  computeKillState,
  computeBombPlantState,
  computeBombDefuseState,
  worldRadiusToPixel,
  SMOKE_DURATION_TICKS,
  SMOKE_FADE_IN_TICKS,
  SMOKE_FADE_OUT_TICKS,
  HE_DURATION_TICKS,
  HE_EXPAND_TICKS,
  HE_RADIUS,
  FLASH_DURATION_TICKS,
  FLASH_RADIUS,
  KILL_DURATION_TICKS,
  BOMB_FLASH_INTERVAL_TICKS,
  BOMB_DEFUSE_TICKS,
  BOMB_DEFUSE_KIT_TICKS,
  SMOKE_RADIUS,
  BOMB_ICON_RADIUS,
} from "./effects"

describe("computeSmokeState", () => {
  it("is inactive before tick 0", () => {
    const state = computeSmokeState(-1)
    expect(state.active).toBe(false)
  })

  it("fades in during first SMOKE_FADE_IN_TICKS", () => {
    const half = Math.floor(SMOKE_FADE_IN_TICKS / 2)
    const state = computeSmokeState(half)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeGreaterThan(0)
    expect(state.alpha).toBeLessThan(1)
  })

  it("alpha is 1 at end of fade-in", () => {
    const state = computeSmokeState(SMOKE_FADE_IN_TICKS)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeCloseTo(1)
  })

  it("holds alpha at 1 in middle of duration", () => {
    const mid = Math.floor(SMOKE_DURATION_TICKS / 2)
    const state = computeSmokeState(mid)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeCloseTo(1)
  })

  it("fades out near end of duration", () => {
    const nearEnd = SMOKE_DURATION_TICKS - Math.floor(SMOKE_FADE_OUT_TICKS / 2)
    const state = computeSmokeState(nearEnd)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeGreaterThan(0)
    expect(state.alpha).toBeLessThan(1)
  })

  it("is inactive at exactly duration ticks", () => {
    const state = computeSmokeState(SMOKE_DURATION_TICKS)
    expect(state.active).toBe(false)
  })

  it("respects custom durationTicks", () => {
    const customDuration = 200
    expect(computeSmokeState(customDuration - 1, customDuration).active).toBe(
      true,
    )
    expect(computeSmokeState(customDuration, customDuration).active).toBe(false)
  })

  it("returns SMOKE_RADIUS as the radius", () => {
    const state = computeSmokeState(SMOKE_FADE_IN_TICKS)
    expect(state.radius).toBe(SMOKE_RADIUS)
  })
})

describe("computeHEState", () => {
  it("is inactive before tick 0", () => {
    expect(computeHEState(-1).active).toBe(false)
  })

  it("radius expands during HE_EXPAND_TICKS", () => {
    const mid = Math.floor(HE_EXPAND_TICKS / 2)
    const state = computeHEState(mid)
    expect(state.active).toBe(true)
    expect(state.radius).toBeGreaterThan(0)
    expect(state.radius).toBeLessThan(HE_RADIUS)
  })

  it("radius is max at end of expand phase", () => {
    const state = computeHEState(HE_EXPAND_TICKS)
    expect(state.active).toBe(true)
    expect(state.radius).toBeCloseTo(HE_RADIUS)
  })

  it("fades out after expand phase", () => {
    const afterExpand =
      HE_EXPAND_TICKS + Math.floor((HE_DURATION_TICKS - HE_EXPAND_TICKS) / 2)
    const state = computeHEState(afterExpand)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeGreaterThan(0)
    expect(state.alpha).toBeLessThan(1)
  })

  it("is inactive at HE_DURATION_TICKS", () => {
    expect(computeHEState(HE_DURATION_TICKS).active).toBe(false)
  })
})

describe("computeFlashState", () => {
  it("is inactive before tick 0", () => {
    expect(computeFlashState(-1).active).toBe(false)
  })

  it("alpha is 1 at tick 0", () => {
    const state = computeFlashState(0)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeCloseTo(1)
  })

  it("alpha decays rapidly", () => {
    const mid = Math.floor(FLASH_DURATION_TICKS / 2)
    const state = computeFlashState(mid)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeGreaterThan(0)
    expect(state.alpha).toBeLessThan(1)
  })

  it("is inactive at FLASH_DURATION_TICKS", () => {
    expect(computeFlashState(FLASH_DURATION_TICKS).active).toBe(false)
  })

  it("returns FLASH_RADIUS as the radius", () => {
    expect(computeFlashState(0).radius).toBe(FLASH_RADIUS)
  })
})

describe("computeKillState", () => {
  it("is inactive before tick 0", () => {
    expect(computeKillState(-1).active).toBe(false)
  })

  it("alpha is 1 in the hold phase (first two thirds)", () => {
    const holdTick = Math.floor((KILL_DURATION_TICKS * 2) / 3) - 1
    const state = computeKillState(holdTick)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeCloseTo(1)
  })

  it("fades out in the last third", () => {
    const fadeStart = Math.floor((KILL_DURATION_TICKS * 2) / 3)
    const midFade =
      fadeStart + Math.floor((KILL_DURATION_TICKS - fadeStart) / 2)
    const state = computeKillState(midFade)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeGreaterThan(0)
    expect(state.alpha).toBeLessThan(1)
  })

  it("is inactive at KILL_DURATION_TICKS", () => {
    expect(computeKillState(KILL_DURATION_TICKS).active).toBe(false)
  })
})

describe("computeBombPlantState", () => {
  const duration = BOMB_DEFUSE_TICKS

  it("is inactive before tick 0", () => {
    expect(computeBombPlantState(-1, duration).active).toBe(false)
  })

  it("is active at tick 0", () => {
    expect(computeBombPlantState(0, duration).active).toBe(true)
  })

  it("oscillates alpha based on BOMB_FLASH_INTERVAL_TICKS", () => {
    const stateOn = computeBombPlantState(0, duration)
    const stateOff = computeBombPlantState(
      Math.floor(BOMB_FLASH_INTERVAL_TICKS * 0.75),
      duration,
    )
    expect(stateOn.alpha).not.toEqual(stateOff.alpha)
  })

  it("returns BOMB_ICON_RADIUS", () => {
    expect(computeBombPlantState(0, duration).radius).toBe(BOMB_ICON_RADIUS)
  })

  it("is inactive at durationTicks", () => {
    expect(computeBombPlantState(duration, duration).active).toBe(false)
  })
})

describe("computeBombDefuseState", () => {
  it("is inactive before tick 0", () => {
    expect(computeBombDefuseState(-1, false).active).toBe(false)
  })

  it("progress increases linearly without kit", () => {
    const half = Math.floor(BOMB_DEFUSE_TICKS / 2)
    const state = computeBombDefuseState(half, false)
    expect(state.active).toBe(true)
    expect(state.progress).toBeCloseTo(0.5, 1)
  })

  it("completes faster with kit", () => {
    const halfKit = Math.floor(BOMB_DEFUSE_KIT_TICKS / 2)
    const state = computeBombDefuseState(halfKit, true)
    expect(state.active).toBe(true)
    expect(state.progress).toBeCloseTo(0.5, 1)
  })

  it("is inactive after BOMB_DEFUSE_TICKS without kit", () => {
    expect(computeBombDefuseState(BOMB_DEFUSE_TICKS, false).active).toBe(false)
  })

  it("is inactive after BOMB_DEFUSE_KIT_TICKS with kit", () => {
    expect(computeBombDefuseState(BOMB_DEFUSE_KIT_TICKS, true).active).toBe(
      false,
    )
  })

  it("is still active at BOMB_DEFUSE_KIT_TICKS without kit", () => {
    expect(computeBombDefuseState(BOMB_DEFUSE_KIT_TICKS, false).active).toBe(
      true,
    )
  })
})

describe("worldRadiusToPixel", () => {
  it("divides world radius by scale", () => {
    expect(worldRadiusToPixel(440, 4.4)).toBeCloseTo(100)
    expect(worldRadiusToPixel(500, 5.0)).toBeCloseTo(100)
    expect(worldRadiusToPixel(SMOKE_RADIUS, 4.4)).toBeCloseTo(
      SMOKE_RADIUS / 4.4,
    )
  })
})
