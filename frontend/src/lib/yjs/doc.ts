import * as Y from "yjs"

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

export interface DrawingElement {
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
  order_index?: number
  created_by: string
  created_at: number
}

export interface BoardSettings {
  title: string
  map_name: string
  zoom: number
  pan_x: number
  pan_y: number
}

export function createStratDoc(): Y.Doc {
  return new Y.Doc({ gc: true })
}

export function getBoardSettings(doc: Y.Doc): Y.Map<unknown> {
  return doc.getMap("boardSettings")
}

export function getDrawingElements(doc: Y.Doc): Y.Array<Y.Map<unknown>> {
  return doc.getArray("drawingElements")
}

export function createDrawingElement(
  elements: Y.Array<Y.Map<unknown>>,
  props: Omit<DrawingElement, "id" | "created_at">,
  doc: Y.Doc
): string {
  const id = crypto.randomUUID()
  const created_at = Date.now()

  doc.transact(() => {
    const element = new Y.Map<unknown>()
    const entries = { ...props, id, created_at }
    for (const [key, value] of Object.entries(entries)) {
      element.set(key, value)
    }
    elements.push([element])
  })

  return id
}

export function removeDrawingElement(
  elements: Y.Array<Y.Map<unknown>>,
  index: number,
  doc: Y.Doc
): void {
  doc.transact(() => {
    elements.delete(index, 1)
  })
}
