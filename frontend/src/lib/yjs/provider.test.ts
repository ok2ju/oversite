import { describe, it, expect, vi, beforeEach } from "vitest"
import * as Y from "yjs"
import { Awareness } from "y-protocols/awareness"

const { mockProvider, MockWebsocketProvider } = vi.hoisted(() => {
  const mockAwareness = { clientID: 1, getStates: () => new Map() }
  const mockProvider = {
    awareness: mockAwareness,
    destroy: vi.fn(),
  }
  const MockWebsocketProvider = vi.fn().mockImplementation(function () {
    return mockProvider
  })
  return { mockProvider, MockWebsocketProvider }
})

vi.mock("y-websocket", () => ({
  WebsocketProvider: MockWebsocketProvider,
}))

import { buildWsUrl, createStratProvider } from "./provider"

describe("provider", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal("location", {
      host: "localhost:3000",
      protocol: "http:",
    })
  })

  describe("buildWsUrl", () => {
    it("converts http to ws", () => {
      expect(buildWsUrl("localhost:3000", "http:")).toBe(
        "ws://localhost:3000/ws/strat"
      )
    })

    it("converts https to wss", () => {
      expect(buildWsUrl("example.com", "https:")).toBe(
        "wss://example.com/ws/strat"
      )
    })
  })

  describe("createStratProvider", () => {
    it("calls WebsocketProvider with correct args", () => {
      const doc = new Y.Doc()

      createStratProvider({ stratId: "abc-123", doc })

      expect(MockWebsocketProvider).toHaveBeenCalledWith(
        "ws://localhost:3000/ws/strat",
        "abc-123",
        doc,
        expect.objectContaining({
          connect: true,
          maxBackoffTime: 10000,
        })
      )
    })

    it("defaults connect to true", () => {
      const doc = new Y.Doc()

      createStratProvider({ stratId: "abc-123", doc })

      expect(MockWebsocketProvider).toHaveBeenCalledWith(
        expect.any(String),
        expect.any(String),
        expect.any(Y.Doc),
        expect.objectContaining({ connect: true })
      )
    })

    it("passes connect=false when specified", () => {
      const doc = new Y.Doc()

      createStratProvider({ stratId: "abc-123", doc, connect: false })

      expect(MockWebsocketProvider).toHaveBeenCalledWith(
        expect.any(String),
        expect.any(String),
        expect.any(Y.Doc),
        expect.objectContaining({ connect: false })
      )
    })

    it("passes custom awareness when provided", () => {
      const doc = new Y.Doc()
      const awareness = new Awareness(doc)

      createStratProvider({ stratId: "abc-123", doc, awareness })

      expect(MockWebsocketProvider).toHaveBeenCalledWith(
        expect.any(String),
        expect.any(String),
        expect.any(Y.Doc),
        expect.objectContaining({ awareness })
      )
    })

    it("destroy calls provider.destroy", () => {
      const doc = new Y.Doc()

      const result = createStratProvider({ stratId: "abc-123", doc })
      result.destroy()

      expect(mockProvider.destroy).toHaveBeenCalled()
    })

    it("returns doc and awareness", () => {
      const doc = new Y.Doc()

      const result = createStratProvider({ stratId: "abc-123", doc })

      expect(result.doc).toBe(doc)
      expect(result.awareness).toBe(mockProvider.awareness)
    })
  })
})
