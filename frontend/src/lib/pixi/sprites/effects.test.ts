import { describe, it, expect } from "vitest"
import {
  computeSmokeState,
  computeHEState,
  computeFlashState,
  computeKillState,
  computeShotState,
  computeBombPlantState,
  computeBombDefuseState,
  computeGrenadeTrajectoryState,
  computeFireState,
  grenadeColor,
  interpolateTrajectory,
  progress,
  worldRadiusToPixel,
  COLOR_DECOY,
  COLOR_FLASH,
  COLOR_GRENADE_DEFAULT,
  COLOR_HE,
  COLOR_MOLOTOV,
  COLOR_SMOKE,
  FIRE_DURATION_TICKS,
  FIRE_FADE_IN_TICKS,
  FIRE_FADE_OUT_TICKS,
  FIRE_RADIUS,
  GRENADE_ICON_RADIUS,
  SMOKE_DURATION_TICKS,
  SMOKE_FADE_IN_TICKS,
  SMOKE_FADE_OUT_TICKS,
  HE_DURATION_TICKS,
  HE_EXPAND_TICKS,
  HE_RADIUS,
  FLASH_DURATION_TICKS,
  FLASH_RADIUS,
  KILL_DURATION_TICKS,
  SHOT_DURATION_TICKS,
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

describe("computeShotState", () => {
  it("is inactive before tick 0", () => {
    expect(computeShotState(-1).active).toBe(false)
  })

  it("alpha starts at 1 and fades linearly", () => {
    const start = computeShotState(0)
    expect(start.active).toBe(true)
    expect(start.alpha).toBeCloseTo(1)

    const mid = computeShotState(Math.floor(SHOT_DURATION_TICKS / 2))
    expect(mid.alpha).toBeLessThan(start.alpha)
    expect(mid.alpha).toBeGreaterThan(0)
  })

  it("is inactive at SHOT_DURATION_TICKS", () => {
    expect(computeShotState(SHOT_DURATION_TICKS).active).toBe(false)
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

describe("progress", () => {
  it("returns 0 below the start", () => {
    expect(progress(100, 200, 50)).toBe(0)
  })

  it("returns 1 at or after the end", () => {
    expect(progress(100, 200, 200)).toBe(1)
    expect(progress(100, 200, 5_000)).toBe(1)
  })

  it("returns the linear fraction inside the range", () => {
    expect(progress(100, 200, 150)).toBeCloseTo(0.5)
    expect(progress(100, 200, 175)).toBeCloseTo(0.75)
  })

  it("returns 1 when end <= start (degenerate range)", () => {
    expect(progress(100, 100, 100)).toBe(1)
    expect(progress(100, 50, 75)).toBe(1)
  })
})

describe("interpolateTrajectory", () => {
  const waypoints = [
    { tick: 100, x: 0, y: 0 },
    { tick: 110, x: 100, y: 0 },
    { tick: 120, x: 100, y: 100 },
  ]

  it("returns origin for empty waypoints", () => {
    expect(interpolateTrajectory([], 100)).toEqual({
      x: 0,
      y: 0,
      segmentIndex: 0,
    })
  })

  it("clamps to first waypoint before its tick", () => {
    const r = interpolateTrajectory(waypoints, 50)
    expect(r.x).toBe(0)
    expect(r.y).toBe(0)
    expect(r.segmentIndex).toBe(0)
  })

  it("clamps to last waypoint after its tick", () => {
    const r = interpolateTrajectory(waypoints, 200)
    expect(r.x).toBe(100)
    expect(r.y).toBe(100)
    expect(r.segmentIndex).toBe(1)
  })

  it("returns the exact waypoint at boundary tick", () => {
    const r = interpolateTrajectory(waypoints, 110)
    expect(r.x).toBeCloseTo(100)
    expect(r.y).toBeCloseTo(0)
  })

  it("lerps mid-segment", () => {
    const r = interpolateTrajectory(waypoints, 105)
    expect(r.x).toBeCloseTo(50)
    expect(r.y).toBeCloseTo(0)
    expect(r.segmentIndex).toBe(0)
  })

  it("picks the correct segment in multi-segment trajectories", () => {
    const r = interpolateTrajectory(waypoints, 115)
    expect(r.x).toBeCloseTo(100)
    expect(r.y).toBeCloseTo(50)
    expect(r.segmentIndex).toBe(1)
  })

  it("returns the only point for single-waypoint trajectories", () => {
    const single = [{ tick: 100, x: 42, y: 7 }]
    const r = interpolateTrajectory(single, 100)
    expect(r.x).toBe(42)
    expect(r.y).toBe(7)
  })
})

describe("computeGrenadeTrajectoryState", () => {
  it("is inactive before tick 0", () => {
    expect(computeGrenadeTrajectoryState(-1, 64).active).toBe(false)
  })

  it("is fully visible while in flight", () => {
    const state = computeGrenadeTrajectoryState(10, 64)
    expect(state.active).toBe(true)
    expect(state.alpha).toBe(1)
    expect(state.radius).toBe(GRENADE_ICON_RADIUS)
  })

  it("is inactive at exactly durationTicks", () => {
    expect(computeGrenadeTrajectoryState(64, 64).active).toBe(false)
  })

  it("reports linear progress through the flight", () => {
    const state = computeGrenadeTrajectoryState(32, 64)
    expect(state.progress).toBeCloseTo(0.5)
  })
})

describe("computeFireState", () => {
  it("is inactive before tick 0", () => {
    expect(computeFireState(-1).active).toBe(false)
  })

  it("fades in during first FIRE_FADE_IN_TICKS", () => {
    const half = Math.floor(FIRE_FADE_IN_TICKS / 2)
    const state = computeFireState(half)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeGreaterThan(0)
    expect(state.alpha).toBeLessThan(1)
  })

  it("holds alpha at 1 in the middle of the burn", () => {
    const mid = Math.floor(FIRE_DURATION_TICKS / 2)
    expect(computeFireState(mid).alpha).toBeCloseTo(1)
  })

  it("fades out near the end", () => {
    const nearEnd = FIRE_DURATION_TICKS - Math.floor(FIRE_FADE_OUT_TICKS / 2)
    const state = computeFireState(nearEnd)
    expect(state.active).toBe(true)
    expect(state.alpha).toBeGreaterThan(0)
    expect(state.alpha).toBeLessThan(1)
  })

  it("is inactive at exactly FIRE_DURATION_TICKS", () => {
    expect(computeFireState(FIRE_DURATION_TICKS).active).toBe(false)
  })

  it("returns FIRE_RADIUS as the radius", () => {
    expect(computeFireState(FIRE_FADE_IN_TICKS).radius).toBe(FIRE_RADIUS)
  })
})

describe("grenadeColor", () => {
  it("maps known weapons to their colors", () => {
    expect(grenadeColor("HE Grenade")).toBe(COLOR_HE)
    expect(grenadeColor("Flashbang")).toBe(COLOR_FLASH)
    expect(grenadeColor("Smoke Grenade")).toBe(COLOR_SMOKE)
    expect(grenadeColor("Molotov")).toBe(COLOR_MOLOTOV)
    expect(grenadeColor("Incendiary Grenade")).toBe(COLOR_MOLOTOV)
    expect(grenadeColor("Decoy Grenade")).toBe(COLOR_DECOY)
  })

  it("returns the default for unknown or null inputs", () => {
    expect(grenadeColor(null)).toBe(COLOR_GRENADE_DEFAULT)
    expect(grenadeColor(undefined)).toBe(COLOR_GRENADE_DEFAULT)
    expect(grenadeColor("AK-47")).toBe(COLOR_GRENADE_DEFAULT)
  })
})
