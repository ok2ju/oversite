import { describe, it, expect } from "vitest"
import {
  tickToPercent,
  percentToTick,
  clientXToPercent,
  roundBoundaryPositions,
  formatTickDisplay,
} from "./timeline-utils"

describe("tickToPercent", () => {
  it("returns 0 at start", () => {
    expect(tickToPercent(0, 128000)).toBe(0)
  })

  it("returns 100 at end", () => {
    expect(tickToPercent(128000, 128000)).toBe(100)
  })

  it("returns 50 at midpoint", () => {
    expect(tickToPercent(64000, 128000)).toBe(50)
  })

  it("clamps negative tick to 0", () => {
    expect(tickToPercent(-10, 128000)).toBe(0)
  })

  it("clamps tick above totalTicks to 100", () => {
    expect(tickToPercent(200000, 128000)).toBe(100)
  })

  it("returns 0 when totalTicks is 0", () => {
    expect(tickToPercent(50, 0)).toBe(0)
  })

  it("returns 0 when totalTicks is negative", () => {
    expect(tickToPercent(50, -1)).toBe(0)
  })
})

describe("percentToTick", () => {
  it("returns 0 at 0%", () => {
    expect(percentToTick(0, 128000)).toBe(0)
  })

  it("returns totalTicks-1 at 100%", () => {
    expect(percentToTick(100, 128000)).toBe(127999)
  })

  it("returns midpoint at 50%", () => {
    expect(percentToTick(50, 128000)).toBe(64000)
  })

  it("floors fractional ticks", () => {
    expect(percentToTick(33.33, 100)).toBe(33)
  })

  it("clamps negative percent to 0", () => {
    expect(percentToTick(-5, 128000)).toBe(0)
  })

  it("clamps percent above 100 to totalTicks-1", () => {
    expect(percentToTick(150, 128000)).toBe(127999)
  })

  it("returns 0 when totalTicks is 0", () => {
    expect(percentToTick(50, 0)).toBe(0)
  })
})

describe("clientXToPercent", () => {
  const rect = { left: 100, width: 400 } as DOMRect

  it("returns 0 at left edge", () => {
    expect(clientXToPercent(100, rect)).toBe(0)
  })

  it("returns 100 at right edge", () => {
    expect(clientXToPercent(500, rect)).toBe(100)
  })

  it("returns 50 at midpoint", () => {
    expect(clientXToPercent(300, rect)).toBe(50)
  })

  it("clamps below left edge to 0", () => {
    expect(clientXToPercent(50, rect)).toBe(0)
  })

  it("clamps beyond right edge to 100", () => {
    expect(clientXToPercent(600, rect)).toBe(100)
  })
})

describe("roundBoundaryPositions", () => {
  it("returns empty array for no boundaries", () => {
    expect(roundBoundaryPositions([], 128000)).toEqual([])
  })

  it("computes correct position for single boundary", () => {
    const boundaries = [{ roundNumber: 2, startTick: 6400, endTick: 12800 }]
    const result = roundBoundaryPositions(boundaries, 128000)
    expect(result).toEqual([{ roundNumber: 2, percent: 5 }])
  })

  it("computes correct positions for multiple boundaries", () => {
    const boundaries = [
      { roundNumber: 2, startTick: 32000, endTick: 64000 },
      { roundNumber: 3, startTick: 64000, endTick: 96000 },
    ]
    const result = roundBoundaryPositions(boundaries, 128000)
    expect(result).toEqual([
      { roundNumber: 2, percent: 25 },
      { roundNumber: 3, percent: 50 },
    ])
  })

  it("returns empty array when totalTicks is 0", () => {
    const boundaries = [{ roundNumber: 2, startTick: 100, endTick: 200 }]
    expect(roundBoundaryPositions(boundaries, 0)).toEqual([])
  })
})

describe("formatTickDisplay", () => {
  it("formats large numbers with commas", () => {
    expect(formatTickDisplay(12345, 128000)).toBe("12,345 / 128,000")
  })

  it("formats zero state", () => {
    expect(formatTickDisplay(0, 0)).toBe("0 / 0")
  })

  it("formats single digit", () => {
    expect(formatTickDisplay(1, 10)).toBe("1 / 10")
  })
})
