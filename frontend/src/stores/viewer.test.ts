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

  it("setDemoId updates demoId and resets tick", () => {
    useViewerStore.getState().setTick(500)
    useViewerStore.getState().setDemoId("demo-123")
    expect(useViewerStore.getState().demoId).toBe("demo-123")
    expect(useViewerStore.getState().currentTick).toBe(0)
  })

  it("setTotalTicks updates totalTicks", () => {
    useViewerStore.getState().setTotalTicks(128000)
    expect(useViewerStore.getState().totalTicks).toBe(128000)
  })

  it("setRound updates currentRound", () => {
    useViewerStore.getState().setRound(5)
    expect(useViewerStore.getState().currentRound).toBe(5)
  })
})
