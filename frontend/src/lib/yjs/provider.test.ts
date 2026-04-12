import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import * as Y from "yjs"
import { Awareness } from "y-protocols/awareness"
import { WebsocketProvider } from "y-websocket"
import { buildWsUrl, createStratProvider } from "./provider"

describe("provider", () => {
  beforeEach(() => {
    vi.stubGlobal("location", {
      host: "localhost:3000",
      protocol: "http:",
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe("buildWsUrl", () => {
    it("converts http to ws", () => {
      expect(buildWsUrl("localhost:3000", "http:")).toBe(
        "ws://localhost:3000/ws/strat",
      )
    })

    it("converts https to wss", () => {
      expect(buildWsUrl("example.com", "https:")).toBe(
        "wss://example.com/ws/strat",
      )
    })
  })

  describe("createStratProvider", () => {
    it("returns a real WebsocketProvider instance", () => {
      const doc = new Y.Doc()
      const result = createStratProvider({
        stratId: "abc-123",
        doc,
        connect: false,
      })

      expect(result.provider).toBeInstanceOf(WebsocketProvider)
      result.destroy()
    })

    it("returns the same doc that was passed in", () => {
      const doc = new Y.Doc()
      const result = createStratProvider({
        stratId: "abc-123",
        doc,
        connect: false,
      })

      expect(result.doc).toBe(doc)
      result.destroy()
    })

    it("returns an Awareness instance", () => {
      const doc = new Y.Doc()
      const result = createStratProvider({
        stratId: "abc-123",
        doc,
        connect: false,
      })

      expect(result.awareness).toBeInstanceOf(Awareness)
      result.destroy()
    })

    it("uses custom awareness when provided", () => {
      const doc = new Y.Doc()
      const awareness = new Awareness(doc)
      const result = createStratProvider({
        stratId: "abc-123",
        doc,
        awareness,
        connect: false,
      })

      expect(result.awareness).toBe(awareness)
      result.destroy()
    })

    it("configures provider with correct room name", () => {
      const doc = new Y.Doc()
      const result = createStratProvider({
        stratId: "my-strat",
        doc,
        connect: false,
      })

      expect(result.provider.roomname).toBe("my-strat")
      result.destroy()
    })

    it("configures provider with correct server URL", () => {
      const doc = new Y.Doc()
      const result = createStratProvider({
        stratId: "abc-123",
        doc,
        connect: false,
      })

      expect(result.provider.url).toContain("ws://localhost:3000/ws/strat")
      result.destroy()
    })

    it("destroy cleans up awareness", () => {
      const doc = new Y.Doc()
      const result = createStratProvider({
        stratId: "abc-123",
        doc,
        connect: false,
      })
      const awareness = result.awareness

      result.destroy()

      expect(awareness.getLocalState()).toBeNull()
    })
  })
})
