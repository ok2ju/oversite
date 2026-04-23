import { describe, it, expect, vi, beforeEach } from "vitest"

// --- Mocks ---

type MockGraphics = {
  circle: ReturnType<typeof vi.fn>
  roundRect: ReturnType<typeof vi.fn>
  arc: ReturnType<typeof vi.fn>
  fill: ReturnType<typeof vi.fn>
  stroke: ReturnType<typeof vi.fn>
  moveTo: ReturnType<typeof vi.fn>
  lineTo: ReturnType<typeof vi.fn>
  closePath: ReturnType<typeof vi.fn>
  clear: ReturnType<typeof vi.fn>
  destroy: ReturnType<typeof vi.fn>
  visible: boolean
  rotation: number
  x: number
  y: number
}

type MockText = {
  text: string
  width: number
  height: number
  anchor: { set: ReturnType<typeof vi.fn> }
  destroy: ReturnType<typeof vi.fn>
  x: number
  y: number
}

type MockContainer = {
  addChild: ReturnType<typeof vi.fn>
  removeChild: ReturnType<typeof vi.fn>
  destroy: ReturnType<typeof vi.fn>
  x: number
  y: number
  alpha: number
  eventMode: string
  cursor: string
  on: ReturnType<typeof vi.fn>
}

const {
  mockGraphicsInstances,
  createMockGraphics,
  mockTextInstances,
  createMockText,
  mockContainerInstances,
  createMockContainer,
} = vi.hoisted(() => {
  const mockGraphicsInstances: MockGraphics[] = []
  const createMockGraphics = (): MockGraphics => {
    const instance: MockGraphics = {
      circle: vi.fn().mockReturnThis(),
      roundRect: vi.fn().mockReturnThis(),
      arc: vi.fn().mockReturnThis(),
      fill: vi.fn().mockReturnThis(),
      stroke: vi.fn().mockReturnThis(),
      moveTo: vi.fn().mockReturnThis(),
      lineTo: vi.fn().mockReturnThis(),
      closePath: vi.fn().mockReturnThis(),
      clear: vi.fn().mockReturnThis(),
      destroy: vi.fn(),
      visible: true,
      rotation: 0,
      x: 0,
      y: 0,
    }
    mockGraphicsInstances.push(instance)
    return instance
  }

  const mockTextInstances: MockText[] = []
  const createMockText = (): MockText => {
    const instance: MockText = {
      text: "",
      width: 40,
      height: 12,
      anchor: { set: vi.fn() },
      destroy: vi.fn(),
      x: 0,
      y: 0,
    }
    mockTextInstances.push(instance)
    return instance
  }

  const mockContainerInstances: MockContainer[] = []
  const createMockContainer = (): MockContainer => {
    const instance: MockContainer = {
      addChild: vi.fn(),
      removeChild: vi.fn(),
      destroy: vi.fn(),
      x: 0,
      y: 0,
      alpha: 1,
      eventMode: "",
      cursor: "",
      on: vi.fn(),
    }
    mockContainerInstances.push(instance)
    return instance
  }

  return {
    mockGraphicsInstances,
    createMockGraphics,
    mockTextInstances,
    createMockText,
    mockContainerInstances,
    createMockContainer,
  }
})

vi.mock("pixi.js", () => ({
  Container: vi.fn().mockImplementation(function () {
    return createMockContainer()
  }),
  Graphics: vi.fn().mockImplementation(function () {
    return createMockGraphics()
  }),
  Text: vi.fn().mockImplementation(function () {
    return createMockText()
  }),
}))

import { MAP_TEST_COORDINATES } from "@/lib/maps/__tests__/fixtures/coordinate-pairs"
import { getMapCalibration } from "@/lib/maps/calibration"
import { worldToPixel } from "@/lib/maps/calibration"
import {
  getTeamColor,
  getTeamOutlineColor,
  yawToRadians,
  PlayerSprite,
} from "./player"

describe("getTeamColor", () => {
  it("returns blue for CT", () => {
    expect(getTeamColor("CT")).toBe(0x5b9bd5)
  })

  it("returns orange for T", () => {
    expect(getTeamColor("T")).toBe(0xe67e22)
  })
})

describe("getTeamOutlineColor", () => {
  it("returns dark blue for CT", () => {
    expect(getTeamOutlineColor("CT")).toBe(0x1e3a5f)
  })

  it("returns dark orange for T", () => {
    expect(getTeamOutlineColor("T")).toBe(0x7a4417)
  })
})

describe("yawToRadians", () => {
  it("converts 0 to 0", () => {
    expect(yawToRadians(0)).toBeCloseTo(0)
  })

  it("converts 90 to -PI/2", () => {
    expect(yawToRadians(90)).toBeCloseTo(-Math.PI / 2)
  })

  it("converts 180 to -PI", () => {
    expect(yawToRadians(180)).toBeCloseTo(-Math.PI)
  })

  it("converts 270 to -3PI/2", () => {
    expect(yawToRadians(270)).toBeCloseTo((-3 * Math.PI) / 2)
  })

  it("converts -90 to PI/2", () => {
    expect(yawToRadians(-90)).toBeCloseTo(Math.PI / 2)
  })
})

describe("worldToPixel (de_dust2 landmarks)", () => {
  const calibration = getMapCalibration("de_dust2")!
  const pairs = MAP_TEST_COORDINATES["de_dust2"]

  pairs.forEach(({ label, world, expectedPixel }) => {
    it(`converts ${label} correctly`, () => {
      const pixel = worldToPixel(world, calibration)
      expect(Math.abs(pixel.x - expectedPixel.x)).toBeLessThan(1)
      expect(Math.abs(pixel.y - expectedPixel.y)).toBeLessThan(1)
    })
  })
})

describe("PlayerSprite", () => {
  let sprite: PlayerSprite

  // Graphics index layout (in construction order):
  //   0: body (circle)
  //   1: pointer (direction triangle)
  //   2: deathMarker
  //   3: selectionRing
  const BODY = 0
  const POINTER = 1
  const DEATH_MARKER = 2
  const SELECTION_RING = 3

  beforeEach(() => {
    vi.clearAllMocks()
    mockGraphicsInstances.length = 0
    mockTextInstances.length = 0
    mockContainerInstances.length = 0
    sprite = new PlayerSprite()
  })

  describe("constructor", () => {
    it("creates a container with interactive settings", () => {
      const container = mockContainerInstances[0]
      expect(container).toBeDefined()
      expect(container.eventMode).toBe("static")
      expect(container.cursor).toBe("pointer")
    })

    it("adds all child graphics and label to the container", () => {
      const container = mockContainerInstances[0]
      // body + pointer + deathMarker + selectionRing + nameLabel = 5
      expect(container.addChild).toHaveBeenCalledTimes(5)
    })

    it("exposes the container as a public property", () => {
      expect(sprite.container).toBe(mockContainerInstances[0])
    })

    it("pointer triangle is filled white", () => {
      expect(mockGraphicsInstances[POINTER].fill).toHaveBeenCalledWith(0xffffff)
    })

    it("pointer triangle is drawn as a closed path", () => {
      const pointer = mockGraphicsInstances[POINTER]
      expect(pointer.moveTo).toHaveBeenCalled()
      expect(pointer.lineTo).toHaveBeenCalled()
      expect(pointer.closePath).toHaveBeenCalled()
    })

    it("death marker starts hidden", () => {
      expect(mockGraphicsInstances[DEATH_MARKER].visible).toBe(false)
    })

    it("selection ring starts hidden", () => {
      expect(mockGraphicsInstances[SELECTION_RING].visible).toBe(false)
    })
  })

  describe("update()", () => {
    it("sets container x and y position", () => {
      sprite.update({
        x: 100,
        y: 200,
        yaw: 0,
        team: "CT",
        name: "player1",
        isAlive: true,
        isSelected: false,
      })
      const container = mockContainerInstances[0]
      expect(container.x).toBe(100)
      expect(container.y).toBe(200)
    })

    it("sets alpha 1.0 for alive player", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "p",
        isAlive: true,
        isSelected: false,
      })
      expect(mockContainerInstances[0].alpha).toBe(1.0)
    })

    it("sets alpha 0.3 for dead player", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "p",
        isAlive: false,
        isSelected: false,
      })
      expect(mockContainerInstances[0].alpha).toBe(0.3)
    })

    it("shows death marker when dead", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "p",
        isAlive: false,
        isSelected: false,
      })
      expect(mockGraphicsInstances[DEATH_MARKER].visible).toBe(true)
    })

    it("hides death marker when alive", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "p",
        isAlive: true,
        isSelected: false,
      })
      expect(mockGraphicsInstances[DEATH_MARKER].visible).toBe(false)
    })

    it("shows selection ring when selected", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "p",
        isAlive: true,
        isSelected: true,
      })
      expect(mockGraphicsInstances[SELECTION_RING].visible).toBe(true)
    })

    it("hides selection ring when not selected", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "p",
        isAlive: true,
        isSelected: false,
      })
      expect(mockGraphicsInstances[SELECTION_RING].visible).toBe(false)
    })

    it("draws body as a circle filled with CT team color", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "p",
        isAlive: true,
        isSelected: false,
      })
      const body = mockGraphicsInstances[BODY]
      expect(body.circle).toHaveBeenCalled()
      expect(body.fill).toHaveBeenCalledWith(0x5b9bd5)
    })

    it("draws body as a circle filled with T team color", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "T",
        name: "p",
        isAlive: true,
        isSelected: false,
      })
      const body = mockGraphicsInstances[BODY]
      expect(body.circle).toHaveBeenCalled()
      expect(body.fill).toHaveBeenCalledWith(0xe67e22)
    })

    it("strokes body with darker team outline", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "p",
        isAlive: true,
        isSelected: false,
      })
      expect(mockGraphicsInstances[BODY].stroke).toHaveBeenCalledWith(
        expect.objectContaining({ color: 0x1e3a5f }),
      )
    })

    it("sets the name label text", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "mezii",
        isAlive: true,
        isSelected: false,
      })
      expect(mockTextInstances[0].text).toBe("mezii")
    })

    it("rotates the pointer based on yaw", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 90,
        team: "CT",
        name: "p",
        isAlive: true,
        isSelected: false,
      })
      expect(mockGraphicsInstances[POINTER].rotation).toBeCloseTo(-Math.PI / 2)
    })

    it("transitions alive to dead correctly", () => {
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "p",
        isAlive: true,
        isSelected: false,
      })
      sprite.update({
        x: 0,
        y: 0,
        yaw: 0,
        team: "CT",
        name: "p",
        isAlive: false,
        isSelected: false,
      })
      expect(mockContainerInstances[0].alpha).toBe(0.3)
      expect(mockGraphicsInstances[DEATH_MARKER].visible).toBe(true)
    })
  })

  describe("setClickHandler()", () => {
    it("registers click handler on container", () => {
      const cb = vi.fn()
      sprite.setClickHandler(cb, "76561198000000001")
      expect(mockContainerInstances[0].on).toHaveBeenCalledWith(
        "pointerdown",
        expect.any(Function),
      )
    })

    it("invokes callback with steamId when clicked", () => {
      const cb = vi.fn()
      sprite.setClickHandler(cb, "76561198000000001")
      const handler = mockContainerInstances[0].on.mock
        .calls[0][1] as () => void
      handler()
      expect(cb).toHaveBeenCalledWith("76561198000000001")
    })
  })

  describe("destroy()", () => {
    it("destroys the container", () => {
      sprite.destroy()
      expect(mockContainerInstances[0].destroy).toHaveBeenCalled()
    })
  })
})
