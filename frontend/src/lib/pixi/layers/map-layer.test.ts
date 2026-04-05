import { describe, it, expect, vi, beforeEach } from "vitest"

const { mockAssets, mockSpriteInstances, createMockSpriteInstance } = vi.hoisted(() => {
  const mockTexture = { width: 1024, height: 1024 }
  const mockAssets = {
    load: vi.fn().mockResolvedValue(mockTexture),
  }
  const mockSpriteInstances: Array<{
    texture: unknown
    width: number
    height: number
    destroy: ReturnType<typeof vi.fn>
  }> = []
  const createMockSpriteInstance = (options?: { texture?: unknown }) => {
    const sprite = {
      texture: options?.texture ?? null,
      width: 0,
      height: 0,
      destroy: vi.fn(),
    }
    mockSpriteInstances.push(sprite)
    return sprite
  }
  return { mockAssets, mockSpriteInstances, createMockSpriteInstance }
})

vi.mock("pixi.js", () => {
  return {
    Assets: mockAssets,
    Sprite: vi.fn().mockImplementation(function (options?: { texture?: unknown }) {
      return createMockSpriteInstance(options)
    }),
  }
})

vi.mock("@/lib/maps/calibration", () => ({
  isCS2Map: (name: string) => name === "de_dust2" || name === "de_mirage",
  getMapCalibration: (name: string) => {
    const cals: Record<string, { originX: number; originY: number; scale: number; width: number; height: number }> = {
      de_dust2: { originX: -2476, originY: 3239, scale: 4.4, width: 1024, height: 1024 },
      de_mirage: { originX: -3230, originY: 1713, scale: 5.0, width: 1024, height: 1024 },
    }
    return cals[name]
  },
  getRadarImagePath: (name: string) => `/maps/${name}.png`,
}))

import { MapLayer } from "./map-layer"

function createMockContainer() {
  return {
    addChild: vi.fn(),
    removeChild: vi.fn(),
  }
}

function lastSprite() {
  return mockSpriteInstances[mockSpriteInstances.length - 1]
}

describe("MapLayer", () => {
  let container: ReturnType<typeof createMockContainer>
  let layer: MapLayer

  beforeEach(() => {
    vi.clearAllMocks()
    mockSpriteInstances.length = 0
    container = createMockContainer()
    layer = new MapLayer(container as never)
  })

  describe("constructor", () => {
    it("has null defaults", () => {
      expect(layer.mapName).toBeNull()
      expect(layer.calibration).toBeNull()
    })
  })

  describe("setMap", () => {
    it("loads texture from correct path", async () => {
      await layer.setMap("de_dust2")

      expect(mockAssets.load).toHaveBeenCalledWith("/maps/de_dust2.png")
    })

    it("creates sprite and adds to container", async () => {
      await layer.setMap("de_dust2")

      expect(container.addChild).toHaveBeenCalledWith(lastSprite())
    })

    it("sets sprite dimensions from calibration", async () => {
      await layer.setMap("de_dust2")

      expect(lastSprite().width).toBe(1024)
      expect(lastSprite().height).toBe(1024)
    })

    it("exposes mapName and calibration", async () => {
      await layer.setMap("de_dust2")

      expect(layer.mapName).toBe("de_dust2")
      expect(layer.calibration).toEqual({
        originX: -2476,
        originY: 3239,
        scale: 4.4,
        width: 1024,
        height: 1024,
      })
    })

    it("clears previous map when switching", async () => {
      await layer.setMap("de_dust2")
      const firstSprite = lastSprite()

      await layer.setMap("de_mirage")

      expect(container.removeChild).toHaveBeenCalledWith(firstSprite)
      expect(firstSprite.destroy).toHaveBeenCalled()
      expect(layer.mapName).toBe("de_mirage")
    })

    it("throws for unknown map", async () => {
      await expect(layer.setMap("de_unknown")).rejects.toThrow(
        'Unknown CS2 map: "de_unknown"'
      )
    })
  })

  describe("clear", () => {
    it("removes and destroys sprite", async () => {
      await layer.setMap("de_dust2")
      const sprite = lastSprite()

      layer.clear()

      expect(container.removeChild).toHaveBeenCalledWith(sprite)
      expect(sprite.destroy).toHaveBeenCalled()
      expect(layer.mapName).toBeNull()
      expect(layer.calibration).toBeNull()
    })

    it("is no-op when no map loaded", () => {
      layer.clear()

      expect(container.removeChild).not.toHaveBeenCalled()
    })
  })

  describe("destroy", () => {
    it("delegates to clear", async () => {
      await layer.setMap("de_dust2")
      const sprite = lastSprite()

      layer.destroy()

      expect(container.removeChild).toHaveBeenCalledWith(sprite)
      expect(sprite.destroy).toHaveBeenCalled()
      expect(layer.mapName).toBeNull()
    })
  })
})
