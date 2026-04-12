import { describe, it, expect, beforeEach } from "vitest"
import { useViewerStore } from "./viewer"

describe("viewerStore", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
  })

  it("has correct initial state", () => {
    const state = useViewerStore.getState()
    expect(state.currentTick).toBe(0)
    expect(state.totalTicks).toBe(0)
    expect(state.isPlaying).toBe(false)
    expect(state.speed).toBe(1)
    expect(state.currentRound).toBe(1)
    expect(state.demoId).toBeNull()
    expect(state.mapName).toBeNull()
  })

  it("setTick updates currentTick", () => {
    useViewerStore.getState().setTick(500)
    expect(useViewerStore.getState().currentTick).toBe(500)
  })

  it("setSpeed updates speed", () => {
    useViewerStore.getState().setSpeed(2)
    expect(useViewerStore.getState().speed).toBe(2)
  })

  it("togglePlay toggles isPlaying", () => {
    useViewerStore.getState().togglePlay()
    expect(useViewerStore.getState().isPlaying).toBe(true)
    useViewerStore.getState().togglePlay()
    expect(useViewerStore.getState().isPlaying).toBe(false)
  })

  it("setMapName updates mapName", () => {
    useViewerStore.getState().setMapName("de_dust2")
    expect(useViewerStore.getState().mapName).toBe("de_dust2")
  })

  it("setDemoId updates demoId and resets tick and mapName", () => {
    useViewerStore.getState().setTick(500)
    useViewerStore.getState().setMapName("de_dust2")
    useViewerStore.getState().setDemoId("demo-123")
    expect(useViewerStore.getState().demoId).toBe("demo-123")
    expect(useViewerStore.getState().currentTick).toBe(0)
    expect(useViewerStore.getState().mapName).toBeNull()
  })

  it("setTotalTicks updates totalTicks", () => {
    useViewerStore.getState().setTotalTicks(128000)
    expect(useViewerStore.getState().totalTicks).toBe(128000)
  })

  it("setRound updates currentRound", () => {
    useViewerStore.getState().setRound(5)
    expect(useViewerStore.getState().currentRound).toBe(5)
  })

  it("reset clears mapName", () => {
    useViewerStore.getState().setMapName("de_mirage")
    useViewerStore.getState().reset()
    expect(useViewerStore.getState().mapName).toBeNull()
  })

  it("has null selectedPlayerSteamId initially", () => {
    expect(useViewerStore.getState().selectedPlayerSteamId).toBeNull()
  })

  it("setSelectedPlayer updates selectedPlayerSteamId", () => {
    useViewerStore.getState().setSelectedPlayer("76561198000000001")
    expect(useViewerStore.getState().selectedPlayerSteamId).toBe(
      "76561198000000001",
    )
  })

  it("setSelectedPlayer(null) clears selectedPlayerSteamId", () => {
    useViewerStore.getState().setSelectedPlayer("76561198000000001")
    useViewerStore.getState().setSelectedPlayer(null)
    expect(useViewerStore.getState().selectedPlayerSteamId).toBeNull()
  })

  it("setDemoId resets selectedPlayerSteamId", () => {
    useViewerStore.getState().setSelectedPlayer("76561198000000001")
    useViewerStore.getState().setDemoId("demo-456")
    expect(useViewerStore.getState().selectedPlayerSteamId).toBeNull()
  })

  it("pause sets isPlaying to false", () => {
    useViewerStore.getState().togglePlay()
    expect(useViewerStore.getState().isPlaying).toBe(true)
    useViewerStore.getState().pause()
    expect(useViewerStore.getState().isPlaying).toBe(false)
  })

  it("pause is idempotent when already paused", () => {
    expect(useViewerStore.getState().isPlaying).toBe(false)
    useViewerStore.getState().pause()
    expect(useViewerStore.getState().isPlaying).toBe(false)
  })

  it("reset clears selectedPlayerSteamId", () => {
    useViewerStore.getState().setSelectedPlayer("76561198000000001")
    useViewerStore.getState().reset()
    expect(useViewerStore.getState().selectedPlayerSteamId).toBeNull()
  })

  describe("viewport state", () => {
    it("has default viewport initially", () => {
      const state = useViewerStore.getState()
      expect(state.viewport).toEqual({ x: 0, y: 0, zoom: 1 })
      expect(state.screenWidth).toBe(0)
      expect(state.screenHeight).toBe(0)
      expect(state.resetViewportCounter).toBe(0)
    })

    it("setViewport updates viewport", () => {
      useViewerStore.getState().setViewport({ x: -100, y: -50, zoom: 2 })
      expect(useViewerStore.getState().viewport).toEqual({
        x: -100,
        y: -50,
        zoom: 2,
      })
    })

    it("setScreenSize updates screenWidth and screenHeight", () => {
      useViewerStore.getState().setScreenSize(800, 600)
      expect(useViewerStore.getState().screenWidth).toBe(800)
      expect(useViewerStore.getState().screenHeight).toBe(600)
    })

    it("resetViewport resets viewport to default and increments counter", () => {
      useViewerStore.getState().setViewport({ x: -200, y: -100, zoom: 3 })
      useViewerStore.getState().resetViewport()
      expect(useViewerStore.getState().viewport).toEqual({
        x: 0,
        y: 0,
        zoom: 1,
      })
      expect(useViewerStore.getState().resetViewportCounter).toBe(1)
    })

    it("resetViewport increments counter each call", () => {
      useViewerStore.getState().resetViewport()
      useViewerStore.getState().resetViewport()
      expect(useViewerStore.getState().resetViewportCounter).toBe(2)
    })

    it("setDemoId resets viewport", () => {
      useViewerStore.getState().setViewport({ x: -200, y: -100, zoom: 3 })
      useViewerStore.getState().setDemoId("demo-789")
      expect(useViewerStore.getState().viewport).toEqual({
        x: 0,
        y: 0,
        zoom: 1,
      })
    })

    it("reset resets viewport state", () => {
      useViewerStore.getState().setViewport({ x: -200, y: -100, zoom: 3 })
      useViewerStore.getState().setScreenSize(800, 600)
      useViewerStore.getState().reset()
      expect(useViewerStore.getState().viewport).toEqual({
        x: 0,
        y: 0,
        zoom: 1,
      })
      expect(useViewerStore.getState().screenWidth).toBe(0)
      expect(useViewerStore.getState().screenHeight).toBe(0)
    })
  })
})
