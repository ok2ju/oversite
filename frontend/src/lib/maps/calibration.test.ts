import { describe, it, expect } from "vitest"
import {
  MAP_CALIBRATIONS,
  worldToPixel,
  pixelToWorld,
  getMapCalibration,
  isCS2Map,
  getRadarImagePath,
  type CS2MapName,
} from "./calibration"
import { MAP_TEST_COORDINATES } from "./__tests__/fixtures/coordinate-pairs"

const ALL_MAPS: CS2MapName[] = [
  "de_dust2",
  "de_mirage",
  "de_inferno",
  "de_nuke",
  "de_ancient",
  "de_vertigo",
  "de_anubis",
]

describe("MAP_CALIBRATIONS", () => {
  it.each(ALL_MAPS)("has calibration entry for %s", (mapName) => {
    const cal = MAP_CALIBRATIONS[mapName]
    expect(cal).toBeDefined()
    expect(cal.width).toBe(1024)
    expect(cal.height).toBe(1024)
    expect(cal.scale).toBeGreaterThan(0)
  })

  it("has exactly 7 maps", () => {
    expect(Object.keys(MAP_CALIBRATIONS)).toHaveLength(7)
  })
})

describe("worldToPixel", () => {
  for (const mapName of ALL_MAPS) {
    describe(mapName, () => {
      const cal = MAP_CALIBRATIONS[mapName]
      const pairs = MAP_TEST_COORDINATES[mapName]

      it.each(pairs)("$label → pixel", ({ world, expectedPixel }) => {
        const pixel = worldToPixel(world, cal)
        expect(Math.abs(pixel.x - expectedPixel.x)).toBeLessThan(1)
        expect(Math.abs(pixel.y - expectedPixel.y)).toBeLessThan(1)
      })
    })
  }
})

describe("pixelToWorld round-trip", () => {
  for (const mapName of ALL_MAPS) {
    describe(mapName, () => {
      const cal = MAP_CALIBRATIONS[mapName]
      const pairs = MAP_TEST_COORDINATES[mapName]

      it.each(pairs)("$label round-trips", ({ world }) => {
        const pixel = worldToPixel(world, cal)
        const back = pixelToWorld(pixel, cal)
        expect(back.x).toBeCloseTo(world.x, 1)
        expect(back.y).toBeCloseTo(world.y, 1)
      })
    })
  }
})

describe("getMapCalibration", () => {
  it("returns calibration for known maps", () => {
    expect(getMapCalibration("de_dust2")).toBe(MAP_CALIBRATIONS.de_dust2)
    expect(getMapCalibration("de_mirage")).toBe(MAP_CALIBRATIONS.de_mirage)
  })

  it("returns undefined for unknown maps", () => {
    expect(getMapCalibration("de_unknown")).toBeUndefined()
    expect(getMapCalibration("")).toBeUndefined()
  })
})

describe("isCS2Map", () => {
  it.each(ALL_MAPS)("returns true for %s", (mapName) => {
    expect(isCS2Map(mapName)).toBe(true)
  })

  it("returns false for unknown maps", () => {
    expect(isCS2Map("de_cbble")).toBe(false)
    expect(isCS2Map("")).toBe(false)
    expect(isCS2Map("dust2")).toBe(false)
  })
})

describe("getRadarImagePath", () => {
  it.each(ALL_MAPS)("returns correct path for %s", (mapName) => {
    expect(getRadarImagePath(mapName)).toBe(`/maps/${mapName}.png`)
  })
})
