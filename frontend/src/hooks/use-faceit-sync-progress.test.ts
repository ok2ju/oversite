import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { mockRuntime, resetRuntimeMocks } from "@/test/mocks/bindings"
import { useFaceitSyncProgress } from "./use-faceit-sync-progress"

vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

describe("useFaceitSyncProgress", () => {
  beforeEach(() => {
    resetRuntimeMocks()
  })

  afterEach(() => {
    resetRuntimeMocks()
  })

  it("returns null progress initially", () => {
    const { result } = renderHook(() => useFaceitSyncProgress())
    expect(result.current.progress).toBeNull()
  })

  it("subscribes to faceit:sync:progress on mount", () => {
    renderHook(() => useFaceitSyncProgress())
    expect(mockRuntime.EventsOn).toHaveBeenCalledWith(
      "faceit:sync:progress",
      expect.any(Function),
    )
  })

  it("updates progress when event fires", () => {
    const { result } = renderHook(() => useFaceitSyncProgress())

    const callback = mockRuntime.EventsOn.mock.calls[0][1]
    act(() => {
      callback({ current: 5, total: 20 })
    })

    expect(result.current.progress).toEqual({ current: 5, total: 20 })
  })

  it("unsubscribes on unmount", () => {
    const { unmount } = renderHook(() => useFaceitSyncProgress())
    unmount()
    expect(mockRuntime.EventsOff).toHaveBeenCalledWith("faceit:sync:progress")
  })

  it("resets progress to null", () => {
    const { result } = renderHook(() => useFaceitSyncProgress())

    const callback = mockRuntime.EventsOn.mock.calls[0][1]
    act(() => {
      callback({ current: 10, total: 20 })
    })
    expect(result.current.progress).not.toBeNull()

    act(() => {
      result.current.reset()
    })
    expect(result.current.progress).toBeNull()
  })
})
