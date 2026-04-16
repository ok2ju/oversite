import { describe, it, expect } from "vitest"
import { computeKDE, type KDEPoint } from "./kde"

describe("computeKDE", () => {
  it("returns zero grid for empty input", () => {
    const grid = computeKDE([], 64, 64, 10)
    expect(grid.width).toBe(64)
    expect(grid.height).toBe(64)
    expect(grid.maxDensity).toBe(0)
    expect(grid.data.length).toBe(64 * 64)
  })

  it("returns zero grid for zero bandwidth", () => {
    const points: KDEPoint[] = [{ x: 32, y: 32, weight: 1 }]
    const grid = computeKDE(points, 64, 64, 0)
    expect(grid.maxDensity).toBe(0)
  })

  it("has peak at the point location for a single point", () => {
    const points: KDEPoint[] = [{ x: 32, y: 32, weight: 1 }]
    const grid = computeKDE(points, 64, 64, 10)

    expect(grid.maxDensity).toBeGreaterThan(0)

    // Peak should be at or near (32, 32)
    const peakIdx = 32 * 64 + 32
    expect(grid.data[peakIdx]).toBe(grid.maxDensity)
  })

  it("respects weight scaling", () => {
    const single: KDEPoint[] = [{ x: 32, y: 32, weight: 1 }]
    const double: KDEPoint[] = [{ x: 32, y: 32, weight: 2 }]

    const gridSingle = computeKDE(single, 64, 64, 10)
    const gridDouble = computeKDE(double, 64, 64, 10)

    const peakIdx = 32 * 64 + 32
    expect(gridDouble.data[peakIdx]).toBeCloseTo(
      gridSingle.data[peakIdx] * 2,
      5,
    )
  })

  it("overlapping points produce higher density", () => {
    const one: KDEPoint[] = [{ x: 32, y: 32, weight: 1 }]
    const two: KDEPoint[] = [
      { x: 32, y: 32, weight: 1 },
      { x: 33, y: 33, weight: 1 },
    ]

    const gridOne = computeKDE(one, 64, 64, 10)
    const gridTwo = computeKDE(two, 64, 64, 10)

    expect(gridTwo.maxDensity).toBeGreaterThan(gridOne.maxDensity)
  })

  it("produces correct grid dimensions", () => {
    const points: KDEPoint[] = [{ x: 50, y: 50, weight: 1 }]
    const grid = computeKDE(points, 128, 256, 10)
    expect(grid.width).toBe(128)
    expect(grid.height).toBe(256)
    expect(grid.data.length).toBe(128 * 256)
  })

  it("density decreases with distance from point", () => {
    const points: KDEPoint[] = [{ x: 32, y: 32, weight: 1 }]
    const grid = computeKDE(points, 64, 64, 10)

    const center = grid.data[32 * 64 + 32]
    const nearby = grid.data[35 * 64 + 35]
    const farther = grid.data[45 * 64 + 45]

    expect(center).toBeGreaterThan(nearby)
    expect(nearby).toBeGreaterThan(farther)
  })

  it("handles points at grid edges", () => {
    const points: KDEPoint[] = [
      { x: 0, y: 0, weight: 1 },
      { x: 63, y: 63, weight: 1 },
    ]
    const grid = computeKDE(points, 64, 64, 5)

    expect(grid.maxDensity).toBeGreaterThan(0)
    expect(grid.data[0]).toBeGreaterThan(0) // corner (0,0)
    expect(grid.data[63 * 64 + 63]).toBeGreaterThan(0) // corner (63,63)
  })
})
