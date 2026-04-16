import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { renderHook } from "@testing-library/react"
import { useViewerStore } from "@/stores/viewer"
import { useViewerKeyboard } from "./use-viewer-keyboard"

function fireKey(key: string, target?: Partial<HTMLElement>) {
  const event = new KeyboardEvent("keydown", {
    key,
    bubbles: true,
    cancelable: true,
  })
  if (target) {
    Object.defineProperty(event, "target", { value: target })
  }
  window.dispatchEvent(event)
  return event
}

describe("useViewerKeyboard", () => {
  const onToggleScoreboard = vi.fn()

  beforeEach(() => {
    useViewerStore.getState().reset()
    useViewerStore.getState().setTotalTicks(10000)
    onToggleScoreboard.mockClear()
  })

  afterEach(() => {
    useViewerStore.getState().reset()
  })

  it("toggles play/pause on Space", () => {
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    expect(useViewerStore.getState().isPlaying).toBe(false)
    fireKey(" ")
    expect(useViewerStore.getState().isPlaying).toBe(true)
    fireKey(" ")
    expect(useViewerStore.getState().isPlaying).toBe(false)
  })

  it("seeks backward on ArrowLeft", () => {
    useViewerStore.getState().setTick(1000)
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("ArrowLeft")
    expect(useViewerStore.getState().currentTick).toBe(680) // 1000 - 320
  })

  it("clamps seek backward to 0", () => {
    useViewerStore.getState().setTick(100)
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("ArrowLeft")
    expect(useViewerStore.getState().currentTick).toBe(0)
  })

  it("seeks forward on ArrowRight", () => {
    useViewerStore.getState().setTick(1000)
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("ArrowRight")
    expect(useViewerStore.getState().currentTick).toBe(1320) // 1000 + 320
  })

  it("clamps seek forward to totalTicks", () => {
    useViewerStore.getState().setTick(9900)
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("ArrowRight")
    expect(useViewerStore.getState().currentTick).toBe(10000)
  })

  it("increases speed on ArrowUp", () => {
    useViewerStore.getState().setSpeed(1)
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("ArrowUp")
    expect(useViewerStore.getState().speed).toBe(2)
    fireKey("ArrowUp")
    expect(useViewerStore.getState().speed).toBe(4)
  })

  it("does not increase speed beyond max", () => {
    useViewerStore.getState().setSpeed(4)
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("ArrowUp")
    expect(useViewerStore.getState().speed).toBe(4)
  })

  it("decreases speed on ArrowDown", () => {
    useViewerStore.getState().setSpeed(1)
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("ArrowDown")
    expect(useViewerStore.getState().speed).toBe(0.5)
    fireKey("ArrowDown")
    expect(useViewerStore.getState().speed).toBe(0.25)
  })

  it("does not decrease speed below min", () => {
    useViewerStore.getState().setSpeed(0.25)
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("ArrowDown")
    expect(useViewerStore.getState().speed).toBe(0.25)
  })

  it("toggles scoreboard on Tab", () => {
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("Tab")
    expect(onToggleScoreboard).toHaveBeenCalledTimes(1)
  })

  it("prevents default on Tab", () => {
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    const event = fireKey("Tab")
    expect(event.defaultPrevented).toBe(true)
  })

  it("deselects player on Escape", () => {
    useViewerStore.getState().setSelectedPlayer("STEAM_123")
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("Escape")
    expect(useViewerStore.getState().selectedPlayerSteamId).toBeNull()
  })

  it("resets viewport on R", () => {
    const before = useViewerStore.getState().resetViewportCounter
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("r")
    expect(useViewerStore.getState().resetViewportCounter).toBe(before + 1)
  })

  it("resets viewport on uppercase R", () => {
    const before = useViewerStore.getState().resetViewportCounter
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey("R")
    expect(useViewerStore.getState().resetViewportCounter).toBe(before + 1)
  })

  it("skips events when target is an input", () => {
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey(" ", { tagName: "INPUT" })
    expect(useViewerStore.getState().isPlaying).toBe(false)
  })

  it("skips events when target is a textarea", () => {
    renderHook(() => useViewerKeyboard({ onToggleScoreboard }))

    fireKey(" ", { tagName: "TEXTAREA" })
    expect(useViewerStore.getState().isPlaying).toBe(false)
  })

  it("cleans up listener on unmount", () => {
    const { unmount } = renderHook(() =>
      useViewerKeyboard({ onToggleScoreboard }),
    )

    unmount()
    fireKey(" ")
    expect(useViewerStore.getState().isPlaying).toBe(false)
  })
})
