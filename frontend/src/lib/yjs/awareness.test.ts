import { describe, it, expect, vi } from "vitest"
import * as Y from "yjs"
import {
  Awareness,
  encodeAwarenessUpdate,
  applyAwarenessUpdate,
} from "y-protocols/awareness"
import {
  COLLABORATION_COLORS,
  setLocalUser,
  updateCursorPosition,
  clearCursor,
  getRemoteStates,
  onAwarenessChange,
} from "./awareness"

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

describe("awareness", () => {
  describe("COLLABORATION_COLORS", () => {
    it("has 8 valid hex colors", () => {
      expect(COLLABORATION_COLORS).toHaveLength(8)
      for (const color of COLLABORATION_COLORS) {
        expect(color).toMatch(/^#[0-9a-f]{6}$/i)
      }
    })
  })

  describe("setLocalUser", () => {
    it("sets user and initializes cursor to null", () => {
      const awareness = new Awareness(new Y.Doc())
      const user = { name: "Alice", color: "#f94144", userId: "user-1" }

      setLocalUser(awareness, user)

      const state = awareness.getLocalState()
      expect(state?.user).toEqual(user)
      expect(state?.cursor).toBeNull()
    })
  })

  describe("updateCursorPosition", () => {
    it("updates cursor field", () => {
      const awareness = new Awareness(new Y.Doc())

      updateCursorPosition(awareness, 150, 250)

      const state = awareness.getLocalState()
      expect(state?.cursor).toEqual({ x: 150, y: 250 })
    })
  })

  describe("clearCursor", () => {
    it("sets cursor to null", () => {
      const awareness = new Awareness(new Y.Doc())
      updateCursorPosition(awareness, 100, 200)

      clearCursor(awareness)

      expect(awareness.getLocalState()?.cursor).toBeNull()
    })
  })

  describe("getRemoteStates", () => {
    it("excludes local client", () => {
      const doc1 = new Y.Doc()
      const doc2 = new Y.Doc()
      const a1 = new Awareness(doc1)
      const a2 = new Awareness(doc2)
      wireAwareness(a1, a2)

      setLocalUser(a1, { name: "Alice", color: "#f94144", userId: "u1" })
      setLocalUser(a2, { name: "Bob", color: "#43aa8b", userId: "u2" })

      const remoteFromA1 = getRemoteStates(a1)
      expect(remoteFromA1.size).toBe(1)
      expect(remoteFromA1.has(a1.clientID)).toBe(false)
      const remoteState = remoteFromA1.get(a2.clientID)
      expect(remoteState?.user.name).toBe("Bob")
    })
  })

  describe("onAwarenessChange", () => {
    it("fires callback on remote state change", () => {
      const doc1 = new Y.Doc()
      const doc2 = new Y.Doc()
      const a1 = new Awareness(doc1)
      const a2 = new Awareness(doc2)
      wireAwareness(a1, a2)

      const callback = vi.fn()
      onAwarenessChange(a1, callback)

      setLocalUser(a2, { name: "Bob", color: "#43aa8b", userId: "u2" })

      expect(callback).toHaveBeenCalled()
    })

    it("stops firing after unsubscribe", () => {
      const doc1 = new Y.Doc()
      const doc2 = new Y.Doc()
      const a1 = new Awareness(doc1)
      const a2 = new Awareness(doc2)
      wireAwareness(a1, a2)

      const callback = vi.fn()
      const unsubscribe = onAwarenessChange(a1, callback)

      setLocalUser(a2, { name: "Bob", color: "#43aa8b", userId: "u2" })
      const callCount = callback.mock.calls.length

      unsubscribe()
      updateCursorPosition(a2, 50, 50)

      expect(callback).toHaveBeenCalledTimes(callCount)
    })
  })
})
