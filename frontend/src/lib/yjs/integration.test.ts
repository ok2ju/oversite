import { describe, it, expect } from "vitest"
import * as Y from "yjs"
import {
  Awareness,
  encodeAwarenessUpdate,
  applyAwarenessUpdate,
} from "y-protocols/awareness"
import {
  createStratDoc,
  getBoardSettings,
  getDrawingElements,
  createDrawingElement,
  removeDrawingElement,
  getStrokeData,
  type DrawingElement,
} from "./doc"
import {
  setLocalUser,
  updateCursorPosition,
  getRemoteStates,
} from "./awareness"

function wireDocs(doc1: Y.Doc, doc2: Y.Doc) {
  doc1.on("update", (update: Uint8Array) => Y.applyUpdate(doc2, update))
  doc2.on("update", (update: Uint8Array) => Y.applyUpdate(doc1, update))
}

function connectDocs(doc1: Y.Doc, doc2: Y.Doc) {
  Y.applyUpdate(doc2, Y.encodeStateAsUpdate(doc1))
  Y.applyUpdate(doc1, Y.encodeStateAsUpdate(doc2))
  wireDocs(doc1, doc2)
}

function wireAwareness(a1: Awareness, a2: Awareness) {
  a1.on(
    "update",
    ({
      added,
      updated,
      removed,
    }: {
      added: number[]
      updated: number[]
      removed: number[]
    }) => {
      const update = encodeAwarenessUpdate(a1, [
        ...added,
        ...updated,
        ...removed,
      ])
      applyAwarenessUpdate(a2, update, "peer")
    }
  )
  a2.on(
    "update",
    ({
      added,
      updated,
      removed,
    }: {
      added: number[]
      updated: number[]
      removed: number[]
    }) => {
      const update = encodeAwarenessUpdate(a2, [
        ...added,
        ...updated,
        ...removed,
      ])
      applyAwarenessUpdate(a1, update, "peer")
    }
  )
}

const baseProps: Omit<DrawingElement, "id" | "created_at"> = {
  type: "freehand",
  x: 0,
  y: 0,
  width: 100,
  height: 100,
  rotation: 0,
  color: "#ff0000",
  lineWidth: 2,
  stroke_data: [1, 2, 3],
  created_by: "user-1",
}

describe("Yjs integration: two-client sync", () => {
  it("two clients sync drawing elements and board settings in real time", () => {
    const docA = createStratDoc()
    const docB = createStratDoc()
    wireDocs(docA, docB)

    const settingsA = getBoardSettings(docA)
    settingsA.set("title", "Mirage A Execute")
    settingsA.set("map_name", "de_mirage")

    const settingsB = getBoardSettings(docB)
    expect(settingsB.get("title")).toBe("Mirage A Execute")
    expect(settingsB.get("map_name")).toBe("de_mirage")

    const elementsB = getDrawingElements(docB)
    const idB = createDrawingElement(
      elementsB,
      { ...baseProps, type: "arrow", color: "#00ff00", created_by: "user-b" },
      docB
    )

    const elementsA = getDrawingElements(docA)
    expect(elementsA.length).toBe(1)
    expect(elementsA.get(0).get("id")).toBe(idB)

    const idA = createDrawingElement(
      elementsA,
      { ...baseProps, type: "line", created_by: "user-a" },
      docA
    )

    expect(elementsB.length).toBe(2)

    removeDrawingElement(elementsA, idB, docA)
    expect(elementsA.length).toBe(1)
    expect(elementsB.length).toBe(1)
    expect(elementsB.get(0).get("id")).toBe(idA)
  })

  it("two clients sync awareness (cursors and user presence)", () => {
    const docA = createStratDoc()
    const docB = createStratDoc()
    const awarenessA = new Awareness(docA)
    const awarenessB = new Awareness(docB)
    wireAwareness(awarenessA, awarenessB)

    setLocalUser(awarenessA, { name: "Alice", color: "#f94144", userId: "u1" })
    setLocalUser(awarenessB, { name: "Bob", color: "#43aa8b", userId: "u2" })

    updateCursorPosition(awarenessA, 100, 200)
    updateCursorPosition(awarenessB, 300, 400)

    const remoteFromA = getRemoteStates(awarenessA)
    expect(remoteFromA.size).toBe(1)
    const bobState = remoteFromA.get(awarenessB.clientID)
    expect(bobState?.user.name).toBe("Bob")
    expect(bobState?.cursor).toEqual({ x: 300, y: 400 })

    const remoteFromB = getRemoteStates(awarenessB)
    expect(remoteFromB.size).toBe(1)
    const aliceState = remoteFromB.get(awarenessA.clientID)
    expect(aliceState?.user.name).toBe("Alice")
    expect(aliceState?.cursor).toEqual({ x: 100, y: 200 })

    awarenessA.destroy()
    awarenessB.destroy()
  })

  it("state reconciles after simulated disconnect/reconnect", () => {
    const docA = createStratDoc()
    const docB = createStratDoc()
    wireDocs(docA, docB)

    const elementsA = getDrawingElements(docA)
    createDrawingElement(
      elementsA,
      { ...baseProps, type: "rectangle", created_by: "user-a" },
      docA
    )
    expect(getDrawingElements(docB).length).toBe(1)

    // Save B's state before disconnect
    const stateBeforeDisconnect = Y.encodeStateAsUpdate(docB)

    // A makes changes while B is offline (no wiring)
    createDrawingElement(
      elementsA,
      { ...baseProps, type: "circle", created_by: "user-a" },
      docA
    )
    expect(getDrawingElements(docA).length).toBe(2)

    // B reconnects: new doc from saved state, then full sync
    const docBReconnected = createStratDoc()
    Y.applyUpdate(docBReconnected, stateBeforeDisconnect)
    expect(getDrawingElements(docBReconnected).length).toBe(1)

    connectDocs(docA, docBReconnected)

    expect(getDrawingElements(docA).length).toBe(2)
    expect(getDrawingElements(docBReconnected).length).toBe(2)
  })

  it("concurrent offline edits merge correctly on reconnect", () => {
    const docA = createStratDoc()
    const docB = createStratDoc()

    createDrawingElement(
      getDrawingElements(docA),
      { ...baseProps, type: "freehand", created_by: "user-a" },
      docA
    )

    createDrawingElement(
      getDrawingElements(docB),
      { ...baseProps, type: "line", created_by: "user-b" },
      docB
    )

    connectDocs(docA, docB)

    expect(getDrawingElements(docA).length).toBe(2)
    expect(getDrawingElements(docB).length).toBe(2)

    const idsA = new Set(
      Array.from({ length: getDrawingElements(docA).length }, (_, i) =>
        getDrawingElements(docA).get(i).get("id")
      )
    )
    const idsB = new Set(
      Array.from({ length: getDrawingElements(docB).length }, (_, i) =>
        getDrawingElements(docB).get(i).get("id")
      )
    )
    expect(idsA).toEqual(idsB)
  })

  it("concurrent stroke_data appends merge across clients", () => {
    const docA = createStratDoc()
    const docB = createStratDoc()
    wireDocs(docA, docB)

    const elementsA = getDrawingElements(docA)
    createDrawingElement(
      elementsA,
      { ...baseProps, stroke_data: [1, 2] },
      docA
    )

    const elA = elementsA.get(0)
    const elB = getDrawingElements(docB).get(0)
    const strokesA = elA.get("stroke_data") as Y.Array<number>
    const strokesB = elB.get("stroke_data") as Y.Array<number>

    strokesA.push([3, 4])
    strokesB.push([5, 6])

    // Both Y.Arrays should have merged all appends
    expect(strokesA.length).toBe(6)
    expect(strokesB.length).toBe(6)
    expect(getStrokeData(elA)).toEqual(getStrokeData(elB))
  })
})
