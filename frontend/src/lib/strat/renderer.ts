import { Graphics, Text, type Container } from "pixi.js"
import * as Y from "yjs"
import { getDrawingElements, getStrokeData, type DrawingElement } from "@/lib/yjs/doc"

const ARROW_HEAD_LENGTH = 12
const PLACEHOLDER_RADIUS = 8

export type ElementData = Pick<
  DrawingElement,
  | "id"
  | "type"
  | "x"
  | "y"
  | "width"
  | "height"
  | "rotation"
  | "color"
  | "lineWidth"
  | "stroke_data"
  | "text"
  | "icon_name"
  | "label"
  | "created_by"
  | "created_at"
>

export function readElementData(elementMap: Y.Map<unknown>): ElementData {
  return {
    id: elementMap.get("id") as string,
    type: elementMap.get("type") as DrawingElement["type"],
    x: (elementMap.get("x") as number) ?? 0,
    y: (elementMap.get("y") as number) ?? 0,
    width: (elementMap.get("width") as number) ?? 0,
    height: (elementMap.get("height") as number) ?? 0,
    rotation: (elementMap.get("rotation") as number) ?? 0,
    color: (elementMap.get("color") as string) ?? "#ffffff",
    lineWidth: (elementMap.get("lineWidth") as number) ?? 2,
    stroke_data: getStrokeData(elementMap),
    text: elementMap.get("text") as string | undefined,
    icon_name: elementMap.get("icon_name") as string | undefined,
    label: elementMap.get("label") as string | undefined,
    created_by: (elementMap.get("created_by") as string) ?? "",
    created_at: (elementMap.get("created_at") as number) ?? 0,
  }
}

export function computeArrowHead(
  x1: number,
  y1: number,
  x2: number,
  y2: number,
  headLength: number
): { left: { x: number; y: number }; right: { x: number; y: number } } {
  const angle = Math.atan2(y2 - y1, x2 - x1)
  const leftAngle = angle + Math.PI / 2
  const rightAngle = angle - Math.PI / 2

  return {
    left: {
      x: x2 - headLength * Math.cos(angle) + headLength * Math.cos(leftAngle),
      y: y2 - headLength * Math.sin(angle) + headLength * Math.sin(leftAngle),
    },
    right: {
      x: x2 - headLength * Math.cos(angle) + headLength * Math.cos(rightAngle),
      y: y2 - headLength * Math.sin(angle) + headLength * Math.sin(rightAngle),
    },
  }
}

function renderElement(
  g: Graphics,
  data: ElementData,
  container: Container,
  texts: Map<string, Text>
): void {
  g.clear()

  const strokeStyle = { color: data.color, width: data.lineWidth }

  switch (data.type) {
    case "freehand": {
      const pts = data.stroke_data
      if (pts.length >= 4) {
        g.moveTo(pts[0], pts[1])
        for (let i = 2; i < pts.length; i += 2) {
          g.lineTo(pts[i], pts[i + 1])
        }
        g.stroke(strokeStyle)
      }
      break
    }
    case "line": {
      g.moveTo(data.x, data.y)
        .lineTo(data.x + data.width, data.y + data.height)
        .stroke(strokeStyle)
      break
    }
    case "arrow": {
      const x2 = data.x + data.width
      const y2 = data.y + data.height
      g.moveTo(data.x, data.y).lineTo(x2, y2).stroke(strokeStyle)

      const head = computeArrowHead(data.x, data.y, x2, y2, ARROW_HEAD_LENGTH)
      g.poly([x2, y2, head.left.x, head.left.y, head.right.x, head.right.y])
        .fill({ color: data.color })
      break
    }
    case "rectangle": {
      g.rect(data.x, data.y, data.width, data.height).stroke(strokeStyle)
      break
    }
    case "circle": {
      g.circle(data.x, data.y, data.width / 2).stroke(strokeStyle)
      break
    }
    case "text": {
      // Remove old text if re-rendering
      const oldText = texts.get(data.id)
      if (oldText) {
        oldText.removeFromParent()
        oldText.destroy()
      }
      const t = new Text({
        text: data.text ?? "",
        style: { fill: data.color, fontSize: 16 },
      })
      t.position.set(data.x, data.y)
      container.addChild(t)
      texts.set(data.id, t)
      break
    }
    // Placeholder types - simple filled circle (full rendering in P5-T06)
    case "icon":
    case "player_token":
    case "grenade_marker":
    case "waypoint": {
      g.circle(data.x, data.y, PLACEHOLDER_RADIUS)
        .fill({ color: data.color })
      break
    }
  }
}

export class StratRenderer {
  private container: Container
  private elements = new Map<string, Graphics>()
  private texts = new Map<string, Text>()
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private observer: ((events: Y.YEvent<any>[]) => void) | null = null
  private drawingElements: Y.Array<Y.Map<unknown>> | null = null

  constructor(container: Container) {
    this.container = container
  }

  attach(doc: Y.Doc): void {
    this.detach()

    this.drawingElements = getDrawingElements(doc)

    // Full sync: render all existing elements
    for (let i = 0; i < this.drawingElements.length; i++) {
      const elementMap = this.drawingElements.get(i)
      this.addElement(elementMap)
    }

    // Observe deep changes
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    this.observer = (events: Y.YEvent<any>[]) => {
      for (const event of events) {
        if (event.target === this.drawingElements) {
          // Array-level: items added or deleted
          this.handleArrayEvent(event as Y.YArrayEvent<Y.Map<unknown>>)
        } else if (event.target instanceof Y.Map) {
          // Map-level: property changed on an element
          const id = event.target.get("id") as string
          if (id && this.elements.has(id)) {
            this.rerenderElement(event.target)
          }
        } else if (event.target instanceof Y.Array) {
          // Nested Y.Array (stroke_data) changed - find parent element
          const parentMap = event.target.parent as Y.Map<unknown> | null
          if (parentMap instanceof Y.Map) {
            const id = parentMap.get("id") as string
            if (id && this.elements.has(id)) {
              this.rerenderElement(parentMap)
            }
          }
        }
      }
    }

    this.drawingElements.observeDeep(this.observer)
  }

  detach(): void {
    if (this.drawingElements && this.observer) {
      this.drawingElements.unobserveDeep(this.observer)
    }
    this.observer = null
    this.drawingElements = null

    // Clean up all Graphics
    for (const [, g] of this.elements) {
      this.container.removeChild(g)
      g.destroy()
    }
    this.elements.clear()

    // Clean up all Text objects
    for (const [, t] of this.texts) {
      this.container.removeChild(t)
      t.destroy()
    }
    this.texts.clear()
  }

  destroy(): void {
    this.detach()
  }

  private addElement(elementMap: Y.Map<unknown>): void {
    const data = readElementData(elementMap)
    const g = new Graphics()
    this.elements.set(data.id, g)
    this.container.addChild(g)
    renderElement(g, data, this.container, this.texts)
  }

  private rerenderElement(elementMap: Y.Map<unknown>): void {
    const data = readElementData(elementMap)
    const g = this.elements.get(data.id)
    if (!g) return
    renderElement(g, data, this.container, this.texts)
  }

  private removeElement(id: string): void {
    const g = this.elements.get(id)
    if (g) {
      this.container.removeChild(g)
      g.destroy()
      this.elements.delete(id)
    }
    const t = this.texts.get(id)
    if (t) {
      this.container.removeChild(t)
      t.destroy()
      this.texts.delete(id)
    }
  }

  private handleArrayEvent(event: Y.YArrayEvent<Y.Map<unknown>>): void {
    // Process insertions from delta
    for (const delta of event.changes.delta) {
      if (delta.insert) {
        const inserted = delta.insert as Y.Map<unknown>[]
        for (const elementMap of inserted) {
          this.addElement(elementMap)
        }
      }
    }

    // Handle removals by diffing known IDs against current array
    if (event.changes.deleted.size > 0) {
      const currentIds = new Set<string>()
      if (this.drawingElements) {
        for (let i = 0; i < this.drawingElements.length; i++) {
          const id = this.drawingElements.get(i).get("id") as string
          if (id) currentIds.add(id)
        }
      }

      const removedIds: string[] = []
      for (const id of this.elements.keys()) {
        if (!currentIds.has(id)) {
          removedIds.push(id)
        }
      }

      for (const id of removedIds) {
        this.removeElement(id)
      }
    }
  }
}
