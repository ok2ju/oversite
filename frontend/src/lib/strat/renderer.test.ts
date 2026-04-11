import { describe, it, expect, vi, beforeEach } from "vitest"
import * as Y from "yjs"

type MockGraphics = {
  clear: ReturnType<typeof vi.fn>
  circle: ReturnType<typeof vi.fn>
  rect: ReturnType<typeof vi.fn>
  moveTo: ReturnType<typeof vi.fn>
  lineTo: ReturnType<typeof vi.fn>
  fill: ReturnType<typeof vi.fn>
  stroke: ReturnType<typeof vi.fn>
  destroy: ReturnType<typeof vi.fn>
  removeFromParent: ReturnType<typeof vi.fn>
  poly: ReturnType<typeof vi.fn>
}

type MockText = {
  text: string
  style: Record<string, unknown>
  position: { set: ReturnType<typeof vi.fn> }
  destroy: ReturnType<typeof vi.fn>
  removeFromParent: ReturnType<typeof vi.fn>
}

const { mockGraphicsInstances, createMockGraphics, mockTextInstances, createMockText } =
  vi.hoisted(() => {
    const mockGraphicsInstances: MockGraphics[] = []
    const mockTextInstances: MockText[] = []

    function createMockGraphics(): MockGraphics {
      const g: MockGraphics = {
        clear: vi.fn(),
        circle: vi.fn(),
        rect: vi.fn(),
        moveTo: vi.fn(),
        lineTo: vi.fn(),
        fill: vi.fn(),
        stroke: vi.fn(),
        destroy: vi.fn(),
        removeFromParent: vi.fn(),
        poly: vi.fn(),
      }
      g.clear.mockReturnValue(g)
      g.circle.mockReturnValue(g)
      g.rect.mockReturnValue(g)
      g.moveTo.mockReturnValue(g)
      g.lineTo.mockReturnValue(g)
      g.fill.mockReturnValue(g)
      g.stroke.mockReturnValue(g)
      g.poly.mockReturnValue(g)
      mockGraphicsInstances.push(g)
      return g
    }

    function createMockText(): MockText {
      const t: MockText = {
        text: "",
        style: {},
        position: { set: vi.fn() },
        destroy: vi.fn(),
        removeFromParent: vi.fn(),
      }
      mockTextInstances.push(t)
      return t
    }

    return { mockGraphicsInstances, createMockGraphics, mockTextInstances, createMockText }
  })

vi.mock("pixi.js", () => ({
  Graphics: vi.fn().mockImplementation(function () {
    return createMockGraphics()
  }),
  Text: vi.fn().mockImplementation(function (_opts?: { text?: string; style?: Record<string, unknown> }) {
    const t = createMockText()
    t.text = _opts?.text ?? ""
    t.style = _opts?.style ?? {}
    return t
  }),
  Container: vi.fn().mockImplementation(function () {
    return {
      addChild: vi.fn(),
      removeChild: vi.fn(),
    }
  }),
}))

import { StratRenderer, readElementData, computeArrowHead } from "./renderer"
import { createDrawingElement, getDrawingElements } from "@/lib/yjs/doc"

function createMockContainer() {
  return {
    addChild: vi.fn(),
    removeChild: vi.fn(),
  }
}

function addElement(
  doc: Y.Doc,
  overrides: Partial<{
    type: string
    x: number
    y: number
    width: number
    height: number
    rotation: number
    color: string
    lineWidth: number
    stroke_data: number[]
    text: string
    icon_name: string
    label: string
    created_by: string
  }> = {}
): string {
  const elements = getDrawingElements(doc)
  return createDrawingElement(
    elements,
    {
      type: (overrides.type ?? "freehand") as "freehand",
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
    },
    doc
  )
}

describe("readElementData", () => {
  it("extracts all properties from Y.Map into a plain object", () => {
    const doc = new Y.Doc()
    const elements = getDrawingElements(doc)
    addElement(doc, { type: "line", x: 10, y: 20, width: 100, height: 50 })

    const elementMap = elements.get(0)
    const data = readElementData(elementMap)

    expect(data.type).toBe("line")
    expect(data.x).toBe(10)
    expect(data.y).toBe(20)
    expect(data.width).toBe(100)
    expect(data.height).toBe(50)
    expect(data.color).toBe("#ff0000")
    expect(data.lineWidth).toBe(2)
    expect(data.id).toBeDefined()
    expect(data.created_by).toBe("user-1")
  })

  it("handles empty stroke_data", () => {
    const doc = new Y.Doc()
    const elements = getDrawingElements(doc)
    addElement(doc, { stroke_data: [] })

    const data = readElementData(elements.get(0))
    expect(data.stroke_data).toEqual([])
  })

  it("reads stroke_data with values", () => {
    const doc = new Y.Doc()
    const elements = getDrawingElements(doc)
    addElement(doc, { stroke_data: [10, 20, 30, 40] })

    const data = readElementData(elements.get(0))
    expect(data.stroke_data).toEqual([10, 20, 30, 40])
  })

  it("handles optional text field", () => {
    const doc = new Y.Doc()
    const elements = getDrawingElements(doc)
    addElement(doc, { type: "text", text: "Rush B" })

    const data = readElementData(elements.get(0))
    expect(data.text).toBe("Rush B")
  })
})

describe("computeArrowHead", () => {
  it("computes arrowhead points for a horizontal line (left to right)", () => {
    const result = computeArrowHead(0, 0, 100, 0, 10)

    // Arrow points should be behind the tip at (100, 0)
    expect(result.left.x).toBeCloseTo(90, 0)
    expect(result.right.x).toBeCloseTo(90, 0)
    // One above, one below the line
    expect(result.left.y).toBeCloseTo(10, 0)
    expect(result.right.y).toBeCloseTo(-10, 0)
  })

  it("computes arrowhead points for a vertical line (top to bottom)", () => {
    const result = computeArrowHead(0, 0, 0, 100, 10)

    expect(result.left.y).toBeCloseTo(90, 0)
    expect(result.right.y).toBeCloseTo(90, 0)
    expect(result.left.x).toBeCloseTo(-10, 0)
    expect(result.right.x).toBeCloseTo(10, 0)
  })

  it("computes arrowhead points for a diagonal line", () => {
    const result = computeArrowHead(0, 0, 100, 100, 14.14)

    // Both points should be at the same distance from tip
    const distL = Math.sqrt(
      (result.left.x - 100) ** 2 + (result.left.y - 100) ** 2
    )
    const distR = Math.sqrt(
      (result.right.x - 100) ** 2 + (result.right.y - 100) ** 2
    )
    expect(distL).toBeCloseTo(distR, 0)
  })
})

describe("StratRenderer", () => {
  let container: ReturnType<typeof createMockContainer>
  let renderer: StratRenderer

  beforeEach(() => {
    mockGraphicsInstances.length = 0
    mockTextInstances.length = 0
    vi.clearAllMocks()
    container = createMockContainer()
    renderer = new StratRenderer(container as never)
  })

  describe("attach / full sync", () => {
    it("renders all existing elements on attach", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "line" })
      addElement(doc, { type: "rectangle" })

      renderer.attach(doc)

      expect(mockGraphicsInstances).toHaveLength(2)
      expect(container.addChild).toHaveBeenCalledTimes(2)
    })

    it("creates no Graphics for empty array", () => {
      const doc = new Y.Doc()

      renderer.attach(doc)

      expect(mockGraphicsInstances).toHaveLength(0)
      expect(container.addChild).not.toHaveBeenCalled()
    })
  })

  describe("Yjs change -> re-render", () => {
    it("adds a new Graphics when element is added after attach", () => {
      const doc = new Y.Doc()
      renderer.attach(doc)

      addElement(doc, { type: "freehand" })

      expect(mockGraphicsInstances).toHaveLength(1)
      expect(container.addChild).toHaveBeenCalledTimes(1)
    })

    it("removes Graphics when element is removed", () => {
      const doc = new Y.Doc()
      const id = addElement(doc, { type: "line" })
      renderer.attach(doc)

      const elements = getDrawingElements(doc)
      expect(mockGraphicsInstances).toHaveLength(1)

      // Remove element
      doc.transact(() => {
        for (let i = 0; i < elements.length; i++) {
          if (elements.get(i).get("id") === id) {
            elements.delete(i, 1)
            break
          }
        }
      })

      expect(container.removeChild).toHaveBeenCalled()
      expect(mockGraphicsInstances[0].destroy).toHaveBeenCalled()
    })

    it("re-renders when a property changes", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "line" })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      g.clear.mockClear()

      // Change a property
      const elements = getDrawingElements(doc)
      elements.get(0).set("color", "#00ff00")

      expect(g.clear).toHaveBeenCalled()
    })

    it("re-renders when stroke_data is appended", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "freehand", stroke_data: [0, 0] })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      g.clear.mockClear()

      // Append stroke data
      const elements = getDrawingElements(doc)
      const strokeArr = elements.get(0).get("stroke_data") as Y.Array<number>
      strokeArr.push([10, 20, 30, 40])

      expect(g.clear).toHaveBeenCalled()
    })
  })

  describe("element type rendering", () => {
    it("renders freehand with moveTo/lineTo from stroke_data", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "freehand", stroke_data: [10, 20, 30, 40, 50, 60] })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      expect(g.moveTo).toHaveBeenCalled()
      expect(g.lineTo).toHaveBeenCalled()
      expect(g.stroke).toHaveBeenCalled()
    })

    it("renders line with moveTo/lineTo", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "line", x: 10, y: 20, width: 100, height: 50 })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      expect(g.moveTo).toHaveBeenCalledWith(10, 20)
      expect(g.lineTo).toHaveBeenCalledWith(110, 70)
      expect(g.stroke).toHaveBeenCalled()
    })

    it("renders arrow with line + arrowhead polygon", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "arrow", x: 0, y: 0, width: 100, height: 0 })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      expect(g.moveTo).toHaveBeenCalled()
      expect(g.lineTo).toHaveBeenCalled()
      expect(g.poly).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })

    it("renders rectangle with rect()", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "rectangle", x: 10, y: 20, width: 100, height: 80 })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      expect(g.rect).toHaveBeenCalledWith(10, 20, 100, 80)
      expect(g.stroke).toHaveBeenCalled()
    })

    it("renders circle with circle()", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "circle", x: 50, y: 50, width: 40, height: 40 })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalledWith(50, 50, 20)
      expect(g.stroke).toHaveBeenCalled()
    })

    it("renders text with PixiJS Text object", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "text", text: "Rush B", x: 100, y: 200, color: "#ff0000" })
      renderer.attach(doc)

      // Text type creates both a Graphics (placeholder) and a Text
      expect(mockTextInstances).toHaveLength(1)
      expect(mockTextInstances[0].position.set).toHaveBeenCalledWith(100, 200)
      expect(container.addChild).toHaveBeenCalled()
    })

    it("renders icon as filled circle placeholder", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "icon", x: 50, y: 50 })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })

    it("renders player_token as filled circle placeholder", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "player_token", x: 50, y: 50 })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })

    it("renders grenade_marker as filled circle placeholder", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "grenade_marker", x: 50, y: 50 })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })

    it("renders waypoint as filled circle placeholder", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "waypoint", x: 50, y: 50 })
      renderer.attach(doc)

      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })
  })

  describe("detach", () => {
    it("clears all Graphics and stops observing", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "line" })
      addElement(doc, { type: "rectangle" })
      renderer.attach(doc)

      expect(mockGraphicsInstances).toHaveLength(2)

      renderer.detach()

      expect(mockGraphicsInstances[0].destroy).toHaveBeenCalled()
      expect(mockGraphicsInstances[1].destroy).toHaveBeenCalled()
      expect(container.removeChild).toHaveBeenCalledTimes(2)
    })

    it("further Yjs changes have no effect after detach", () => {
      const doc = new Y.Doc()
      renderer.attach(doc)
      renderer.detach()

      const addChildCalls = container.addChild.mock.calls.length

      addElement(doc, { type: "line" })

      // No new Graphics should be created
      expect(container.addChild.mock.calls.length).toBe(addChildCalls)
    })
  })

  describe("destroy", () => {
    it("calls destroy on all Graphics and Text objects", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "line" })
      addElement(doc, { type: "text", text: "test" })
      renderer.attach(doc)

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
    it("handles 100+ elements on attach", () => {
      const doc = new Y.Doc()
      doc.transact(() => {
        for (let i = 0; i < 100; i++) {
          addElement(doc, { type: "line", x: i, y: i })
        }
      })

      renderer.attach(doc)

      expect(mockGraphicsInstances).toHaveLength(100)
    })

    it("targeted update only affects changed element", () => {
      const doc = new Y.Doc()
      addElement(doc, { type: "line" })
      addElement(doc, { type: "rectangle" })
      renderer.attach(doc)

      const g0 = mockGraphicsInstances[0]
      const g1 = mockGraphicsInstances[1]
      g0.clear.mockClear()
      g1.clear.mockClear()

      // Change only the first element
      const elements = getDrawingElements(doc)
      elements.get(0).set("color", "#00ff00")

      expect(g0.clear).toHaveBeenCalled()
      expect(g1.clear).not.toHaveBeenCalled()
    })
  })
})
