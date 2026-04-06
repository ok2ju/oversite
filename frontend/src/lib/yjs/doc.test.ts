import { describe, it, expect } from "vitest"
import * as Y from "yjs"
import {
  createStratDoc,
  getBoardSettings,
  getDrawingElements,
  createDrawingElement,
  removeDrawingElement,
  type DrawingElement,
} from "./doc"

function wireDocs(doc1: Y.Doc, doc2: Y.Doc) {
  doc1.on("update", (update: Uint8Array) => Y.applyUpdate(doc2, update))
  doc2.on("update", (update: Uint8Array) => Y.applyUpdate(doc1, update))
}

const baseProps: Omit<DrawingElement, "id" | "created_at"> = {
  type: "freehand",
  x: 10,
  y: 20,
  width: 100,
  height: 50,
  rotation: 0,
  color: "#ff0000",
  lineWidth: 2,
  stroke_data: [1, 2, 3],
  created_by: "user-1",
}

describe("doc", () => {
  describe("createStratDoc", () => {
    it("returns a Y.Doc with gc enabled", () => {
      const doc = createStratDoc()

      expect(doc).toBeInstanceOf(Y.Doc)
      expect(doc.gc).toBe(true)
    })
  })

  describe("boardSettings convergence", () => {
    it("syncs boardSettings between two docs", () => {
      const doc1 = createStratDoc()
      const doc2 = createStratDoc()
      wireDocs(doc1, doc2)

      const settings1 = getBoardSettings(doc1)
      settings1.set("title", "Mirage A Execute")
      settings1.set("map_name", "de_mirage")
      settings1.set("zoom", 1.5)
      settings1.set("pan_x", 100)
      settings1.set("pan_y", 200)

      const settings2 = getBoardSettings(doc2)
      expect(settings2.get("title")).toBe("Mirage A Execute")
      expect(settings2.get("map_name")).toBe("de_mirage")
      expect(settings2.get("zoom")).toBe(1.5)
      expect(settings2.get("pan_x")).toBe(100)
      expect(settings2.get("pan_y")).toBe(200)
    })
  })

  describe("drawingElements sync", () => {
    it("syncs created element to second doc", () => {
      const doc1 = createStratDoc()
      const doc2 = createStratDoc()
      wireDocs(doc1, doc2)

      const elements1 = getDrawingElements(doc1)
      const id = createDrawingElement(elements1, baseProps, doc1)

      const elements2 = getDrawingElements(doc2)
      expect(elements2.length).toBe(1)

      const synced = elements2.get(0)
      expect(synced.get("id")).toBe(id)
      expect(synced.get("type")).toBe("freehand")
      expect(synced.get("x")).toBe(10)
      expect(synced.get("color")).toBe("#ff0000")
      expect(synced.get("created_by")).toBe("user-1")
    })

    it("merges concurrent edits from both docs", () => {
      const doc1 = createStratDoc()
      const doc2 = createStratDoc()

      // Add elements before wiring (simulates concurrent offline edits)
      const elements1 = getDrawingElements(doc1)
      createDrawingElement(elements1, { ...baseProps, x: 1 }, doc1)

      const elements2 = getDrawingElements(doc2)
      createDrawingElement(elements2, { ...baseProps, x: 2 }, doc2)

      // Now sync
      wireDocs(doc1, doc2)
      Y.applyUpdate(doc2, Y.encodeStateAsUpdate(doc1))
      Y.applyUpdate(doc1, Y.encodeStateAsUpdate(doc2))

      expect(elements1.length).toBe(2)
      expect(elements2.length).toBe(2)
    })
  })

  describe("createDrawingElement", () => {
    it("generates unique id and created_at", () => {
      const doc = createStratDoc()
      const elements = getDrawingElements(doc)

      const id1 = createDrawingElement(elements, baseProps, doc)
      const id2 = createDrawingElement(elements, baseProps, doc)

      expect(id1).not.toBe(id2)

      const el1 = elements.get(0)
      const el2 = elements.get(1)
      expect(typeof el1.get("created_at")).toBe("number")
      expect(typeof el2.get("created_at")).toBe("number")
    })
  })

  describe("removeDrawingElement", () => {
    it("removes element and syncs removal to second doc", () => {
      const doc1 = createStratDoc()
      const doc2 = createStratDoc()
      wireDocs(doc1, doc2)

      const elements1 = getDrawingElements(doc1)
      createDrawingElement(elements1, baseProps, doc1)
      expect(getDrawingElements(doc2).length).toBe(1)

      removeDrawingElement(elements1, 0, doc1)

      expect(elements1.length).toBe(0)
      expect(getDrawingElements(doc2).length).toBe(0)
    })
  })
})
