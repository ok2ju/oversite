import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  createMockGraphics,
  createMockText,
  type MockGraphics,
  type MockText,
} from "@/test/mocks/pixi"

const { mockGraphicsInstances, mockTextInstances } = vi.hoisted(() => {
  const mockGraphicsInstances: MockGraphics[] = []
  const mockTextInstances: MockText[] = []
  return { mockGraphicsInstances, mockTextInstances }
})

vi.mock("pixi.js", () => ({
  Graphics: vi.fn().mockImplementation(function () {
    const g = createMockGraphics()
    mockGraphicsInstances.push(g)
    return g
  }),
  Text: vi.fn().mockImplementation(function (_opts?: {
    text?: string
    style?: Record<string, unknown>
  }) {
    const t = createMockText()
    t.text = _opts?.text ?? ""
    t.style = _opts?.style ?? {}
    mockTextInstances.push(t)
    return t
  }),
  Container: vi.fn().mockImplementation(function () {
    return {
      addChild: vi.fn(),
      removeChild: vi.fn(),
    }
  }),
}))

import { StratRenderer, computeArrowHead, type ElementData } from "./renderer"

function createMockContainer() {
  return {
    addChild: vi.fn(),
    removeChild: vi.fn(),
  }
}

let nextId = 0
function makeElement(overrides: Partial<ElementData> = {}): ElementData {
  nextId++
  return {
    id: overrides.id ?? `el-${nextId}`,
    type: overrides.type ?? "freehand",
    x: overrides.x ?? 100,
    y: overrides.y ?? 200,
    width: overrides.width ?? 50,
    height: overrides.height ?? 50,
    rotation: overrides.rotation ?? 0,
    color: overrides.color ?? "#ff0000",
    lineWidth: overrides.lineWidth ?? 2,
    stroke_data: overrides.stroke_data ?? [],
    text: overrides.text,
    icon_name: overrides.icon_name,
    label: overrides.label,
    created_by: overrides.created_by ?? "user-1",
    created_at: overrides.created_at ?? Date.now(),
  }
}

describe("computeArrowHead", () => {
  it("computes arrowhead points for a horizontal line (left to right)", () => {
    const result = computeArrowHead(0, 0, 100, 0, 10)

    // Wings angled backward at ±30° from the shaft
    expect(result.left.x).toBeCloseTo(91.34, 1)
    expect(result.right.x).toBeCloseTo(91.34, 1)
    // One above, one below the line
    expect(result.left.y).toBeCloseTo(5, 1)
    expect(result.right.y).toBeCloseTo(-5, 1)
  })

  it("computes arrowhead points for a vertical line (top to bottom)", () => {
    const result = computeArrowHead(0, 0, 0, 100, 10)

    expect(result.left.y).toBeCloseTo(91.34, 1)
    expect(result.right.y).toBeCloseTo(91.34, 1)
    expect(result.left.x).toBeCloseTo(-5, 1)
    expect(result.right.x).toBeCloseTo(5, 1)
  })

  it("computes arrowhead points for a diagonal line", () => {
    const result = computeArrowHead(0, 0, 100, 100, 14.14)

    // Both points should be at the same distance from tip
    const distL = Math.sqrt(
      (result.left.x - 100) ** 2 + (result.left.y - 100) ** 2,
    )
    const distR = Math.sqrt(
      (result.right.x - 100) ** 2 + (result.right.y - 100) ** 2,
    )
    expect(distL).toBeCloseTo(distR, 0)
  })
})

describe("StratRenderer", () => {
  let container: ReturnType<typeof createMockContainer>
  let renderer: StratRenderer

  beforeEach(() => {
    nextId = 0
    mockGraphicsInstances.length = 0
    mockTextInstances.length = 0
    vi.clearAllMocks()
    container = createMockContainer()
    renderer = new StratRenderer(container as never)
  })

  describe("addElement", () => {
    it("creates a Graphics and adds it to the container", () => {
      renderer.addElement(makeElement({ type: "line" }))

      expect(mockGraphicsInstances).toHaveLength(1)
      expect(container.addChild).toHaveBeenCalledTimes(1)
    })
  })

  describe("sync", () => {
    it("renders all elements in the data array", () => {
      renderer.sync([
        makeElement({ id: "a", type: "line" }),
        makeElement({ id: "b", type: "rectangle" }),
      ])

      expect(mockGraphicsInstances).toHaveLength(2)
      expect(container.addChild).toHaveBeenCalledTimes(2)
    })

    it("creates no Graphics for empty array", () => {
      renderer.sync([])

      expect(mockGraphicsInstances).toHaveLength(0)
      expect(container.addChild).not.toHaveBeenCalled()
    })

    it("removes elements no longer present", () => {
      const el = makeElement({ id: "a", type: "line" })
      renderer.sync([el])
      expect(mockGraphicsInstances).toHaveLength(1)

      renderer.sync([])

      expect(container.removeChild).toHaveBeenCalled()
      expect(mockGraphicsInstances[0].destroy).toHaveBeenCalled()
    })

    it("re-renders when data changes", () => {
      const el = makeElement({ id: "a", type: "line", color: "#ff0000" })
      renderer.sync([el])

      const g = mockGraphicsInstances[0]
      g.clear.mockClear()

      renderer.sync([{ ...el, color: "#00ff00" }])

      expect(g.clear).toHaveBeenCalled()
    })
  })

  describe("removeElement", () => {
    it("destroys Graphics and removes from container", () => {
      const el = makeElement({ id: "a", type: "line" })
      renderer.addElement(el)

      renderer.removeElement("a")

      expect(container.removeChild).toHaveBeenCalled()
      expect(mockGraphicsInstances[0].destroy).toHaveBeenCalled()
    })
  })

  describe("element type rendering", () => {
    it("renders freehand with moveTo/lineTo from stroke_data", () => {
      renderer.addElement(
        makeElement({
          type: "freehand",
          stroke_data: [10, 20, 30, 40, 50, 60],
        }),
      )

      const g = mockGraphicsInstances[0]
      expect(g.moveTo).toHaveBeenCalled()
      expect(g.lineTo).toHaveBeenCalled()
      expect(g.stroke).toHaveBeenCalled()
    })

    it("renders line with moveTo/lineTo", () => {
      renderer.addElement(
        makeElement({ type: "line", x: 10, y: 20, width: 100, height: 50 }),
      )

      const g = mockGraphicsInstances[0]
      expect(g.moveTo).toHaveBeenCalledWith(10, 20)
      expect(g.lineTo).toHaveBeenCalledWith(110, 70)
      expect(g.stroke).toHaveBeenCalled()
    })

    it("renders arrow with line + arrowhead polygon", () => {
      renderer.addElement(
        makeElement({ type: "arrow", x: 0, y: 0, width: 100, height: 0 }),
      )

      const g = mockGraphicsInstances[0]
      expect(g.moveTo).toHaveBeenCalled()
      expect(g.lineTo).toHaveBeenCalled()
      expect(g.poly).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })

    it("renders rectangle with rect()", () => {
      renderer.addElement(
        makeElement({
          type: "rectangle",
          x: 10,
          y: 20,
          width: 100,
          height: 80,
        }),
      )

      const g = mockGraphicsInstances[0]
      expect(g.rect).toHaveBeenCalledWith(10, 20, 100, 80)
      expect(g.stroke).toHaveBeenCalled()
    })

    it("renders circle with circle()", () => {
      renderer.addElement(
        makeElement({ type: "circle", x: 50, y: 50, width: 40, height: 40 }),
      )

      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalledWith(50, 50, 20)
      expect(g.stroke).toHaveBeenCalled()
    })

    it("renders text with PixiJS Text object", () => {
      renderer.addElement(
        makeElement({
          type: "text",
          text: "Rush B",
          x: 100,
          y: 200,
          color: "#ff0000",
        }),
      )

      expect(mockTextInstances).toHaveLength(1)
      expect(mockTextInstances[0].position.set).toHaveBeenCalledWith(100, 200)
      expect(container.addChild).toHaveBeenCalled()
    })

    it("renders icon as filled circle placeholder", () => {
      renderer.addElement(makeElement({ type: "icon", x: 50, y: 50 }))

      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })

    it("renders player_token as filled circle placeholder", () => {
      renderer.addElement(makeElement({ type: "player_token", x: 50, y: 50 }))

      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })

    it("renders grenade_marker as filled circle placeholder", () => {
      renderer.addElement(makeElement({ type: "grenade_marker", x: 50, y: 50 }))

      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })

    it("renders waypoint as filled circle placeholder", () => {
      renderer.addElement(makeElement({ type: "waypoint", x: 50, y: 50 }))

      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })
  })

  describe("clear", () => {
    it("destroys all Graphics and Text objects", () => {
      renderer.addElement(makeElement({ id: "a", type: "line" }))
      renderer.addElement(makeElement({ id: "b", type: "rectangle" }))

      renderer.clear()

      expect(mockGraphicsInstances[0].destroy).toHaveBeenCalled()
      expect(mockGraphicsInstances[1].destroy).toHaveBeenCalled()
      expect(container.removeChild).toHaveBeenCalledTimes(2)
    })
  })

  describe("destroy", () => {
    it("calls destroy on all Graphics and Text objects", () => {
      renderer.addElement(makeElement({ id: "a", type: "line" }))
      renderer.addElement(makeElement({ id: "b", type: "text", text: "test" }))

      renderer.destroy()

      for (const g of mockGraphicsInstances) {
        expect(g.destroy).toHaveBeenCalled()
      }
      for (const t of mockTextInstances) {
        expect(t.destroy).toHaveBeenCalled()
      }
    })
  })

  describe("batch rendering", () => {
    it("handles 100+ elements via sync", () => {
      const data = Array.from({ length: 100 }, (_, i) =>
        makeElement({ id: `el-batch-${i}`, type: "line", x: i, y: i }),
      )

      renderer.sync(data)

      expect(mockGraphicsInstances).toHaveLength(100)
    })
  })
})
