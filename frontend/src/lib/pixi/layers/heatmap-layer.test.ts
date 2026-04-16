import { describe, it, expect, vi, beforeEach } from "vitest"

const mockDestroy = vi.fn()
const mockSpriteInstances: Array<{
  width: number
  height: number
  texture: unknown
  destroy: ReturnType<typeof vi.fn>
}> = []

vi.mock("pixi.js", () => ({
  Sprite: vi.fn().mockImplementation(function (opts?: { texture?: unknown }) {
    const sprite = {
      width: 0,
      height: 0,
      texture: opts?.texture ?? null,
      destroy: mockDestroy,
    }
    mockSpriteInstances.push(sprite)
    return sprite
  }),
  Texture: {
    from: vi.fn().mockReturnValue({ width: 512, height: 512 }),
  },
  Container: vi.fn().mockImplementation(function () {
    return {
      addChild: vi.fn(),
      removeChild: vi.fn(),
    }
  }),
}))

import { HeatmapLayer } from "./heatmap-layer"
import type { HeatmapPoint } from "@/types/heatmap"
import type { MapCalibration } from "@/lib/maps/calibration"

const testCalibration: MapCalibration = {
  originX: -2476,
  originY: 3239,
  scale: 4.4,
  width: 1024,
  height: 1024,
}

// Stub canvas 2d context — jsdom doesn't support it
function createFake2DContext() {
  return {
    createImageData: vi.fn((w: number, h: number) => ({
      data: new Uint8ClampedArray(w * h * 4),
      width: w,
      height: h,
    })),
    putImageData: vi.fn(),
  }
}

describe("HeatmapLayer", () => {
  let container: {
    addChild: ReturnType<typeof vi.fn>
    removeChild: ReturnType<typeof vi.fn>
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockSpriteInstances.length = 0
    container = {
      addChild: vi.fn(),
      removeChild: vi.fn(),
    }

    // Provide a fake canvas 2d context for jsdom
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(
      createFake2DContext() as unknown as GPUCanvasContext,
    )
  })

  it("initializes with no sprite", () => {
    const layer = new HeatmapLayer(container as never)
    expect(mockSpriteInstances).toHaveLength(0)
    layer.destroy()
  })

  it("renders a sprite for valid points", () => {
    const layer = new HeatmapLayer(container as never)
    const points: HeatmapPoint[] = [
      { x: 0, y: 0, kill_count: 3 },
      { x: 500, y: 500, kill_count: 1 },
    ]

    layer.render(points, testCalibration)

    expect(mockSpriteInstances).toHaveLength(1)
    expect(container.addChild).toHaveBeenCalledTimes(1)
    // Sprite should be sized to map dimensions
    expect(mockSpriteInstances[0].width).toBe(1024)
    expect(mockSpriteInstances[0].height).toBe(1024)

    layer.destroy()
  })

  it("does not render for empty points", () => {
    const layer = new HeatmapLayer(container as never)
    layer.render([], testCalibration)

    expect(mockSpriteInstances).toHaveLength(0)
    expect(container.addChild).not.toHaveBeenCalled()

    layer.destroy()
  })

  it("clears previous render before new render", () => {
    const layer = new HeatmapLayer(container as never)
    const points: HeatmapPoint[] = [{ x: 100, y: 200, kill_count: 5 }]

    layer.render(points, testCalibration)
    expect(mockSpriteInstances).toHaveLength(1)

    // Render again — should clear old sprite first
    layer.render(points, testCalibration)
    expect(container.removeChild).toHaveBeenCalledTimes(1)
    expect(mockDestroy).toHaveBeenCalledTimes(1)
    expect(mockSpriteInstances).toHaveLength(2)
  })

  it("clear removes the sprite", () => {
    const layer = new HeatmapLayer(container as never)
    const points: HeatmapPoint[] = [{ x: 100, y: 200, kill_count: 1 }]

    layer.render(points, testCalibration)
    layer.clear()

    expect(container.removeChild).toHaveBeenCalledTimes(1)
    expect(mockDestroy).toHaveBeenCalledWith({ texture: true })
  })

  it("clear on empty layer is safe", () => {
    const layer = new HeatmapLayer(container as never)
    layer.clear()
    expect(container.removeChild).not.toHaveBeenCalled()
  })

  it("destroy delegates to clear", () => {
    const layer = new HeatmapLayer(container as never)
    const points: HeatmapPoint[] = [{ x: 100, y: 200, kill_count: 1 }]

    layer.render(points, testCalibration)
    layer.destroy()

    expect(container.removeChild).toHaveBeenCalledTimes(1)
    expect(mockDestroy).toHaveBeenCalled()
  })

  it("accepts custom options", () => {
    const layer = new HeatmapLayer(container as never)
    const points: HeatmapPoint[] = [{ x: 100, y: 200, kill_count: 5 }]

    layer.render(points, testCalibration, {
      gridSize: 256,
      bandwidth: 25,
      opacity: 0.5,
    })

    expect(mockSpriteInstances).toHaveLength(1)
    layer.destroy()
  })
})
