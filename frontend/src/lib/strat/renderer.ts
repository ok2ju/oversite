import { Graphics, Text, type Container } from "pixi.js"

// Types extracted from the former lib/yjs/doc.ts — kept here until
// the desktop strat board is implemented (P5 tasks).
export type DrawingElementType =
  | "freehand"
  | "line"
  | "arrow"
  | "rectangle"
  | "circle"
  | "text"
  | "icon"
  | "player_token"
  | "grenade_marker"
  | "waypoint"

export interface ElementData {
  id: string
  type: DrawingElementType
  x: number
  y: number
  width: number
  height: number
  rotation: number
  color: string
  lineWidth: number
  stroke_data: number[]
  text?: string
  icon_name?: string
  label?: string
  created_by: string
  created_at: number
}

const ARROW_HEAD_LENGTH = 12
const PLACEHOLDER_RADIUS = 8

export function computeArrowHead(
  x1: number,
  y1: number,
  x2: number,
  y2: number,
  headLength: number,
): { left: { x: number; y: number }; right: { x: number; y: number } } {
  const angle = Math.atan2(y2 - y1, x2 - x1)

  return {
    left: {
      x: x2 - headLength * Math.cos(angle - Math.PI / 6),
      y: y2 - headLength * Math.sin(angle - Math.PI / 6),
    },
    right: {
      x: x2 - headLength * Math.cos(angle + Math.PI / 6),
      y: y2 - headLength * Math.sin(angle + Math.PI / 6),
    },
  }
}

function renderElement(
  g: Graphics,
  data: ElementData,
  container: Container,
  texts: Map<string, Text>,
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
      g.poly([
        x2,
        y2,
        head.left.x,
        head.left.y,
        head.right.x,
        head.right.y,
      ]).fill({ color: data.color })
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
      g.circle(data.x, data.y, PLACEHOLDER_RADIUS).fill({ color: data.color })
      break
    }
  }
}

export class StratRenderer {
  private container: Container
  private elements = new Map<string, Graphics>()
  private texts = new Map<string, Text>()

  constructor(container: Container) {
    this.container = container
  }

  /** Replace the full set of elements and re-render. */
  sync(data: ElementData[]): void {
    const incoming = new Set(data.map((d) => d.id))

    // Remove elements no longer present
    for (const id of [...this.elements.keys()]) {
      if (!incoming.has(id)) {
        this.removeElement(id)
      }
    }

    // Add or update
    for (const d of data) {
      if (this.elements.has(d.id)) {
        this.rerenderElement(d)
      } else {
        this.addElement(d)
      }
    }
  }

  /** Add a single element. */
  addElement(data: ElementData): void {
    const g = new Graphics()
    this.elements.set(data.id, g)
    this.container.addChild(g)
    renderElement(g, data, this.container, this.texts)
  }

  /** Remove a single element by id. */
  removeElement(id: string): void {
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

  /** Clear all rendered elements. */
  clear(): void {
    for (const [, g] of this.elements) {
      this.container.removeChild(g)
      g.destroy()
    }
    this.elements.clear()

    for (const [, t] of this.texts) {
      this.container.removeChild(t)
      t.destroy()
    }
    this.texts.clear()
  }

  destroy(): void {
    this.clear()
  }

  private rerenderElement(data: ElementData): void {
    const g = this.elements.get(data.id)
    if (!g) return
    renderElement(g, data, this.container, this.texts)
  }
}
