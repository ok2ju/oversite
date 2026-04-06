import { describe, it, expect, beforeEach, afterEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { useViewerStore } from "@/stores/viewer"
import { useTickData } from "./use-tick-data"

describe("useTickData", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
  })

  afterEach(() => {
    useViewerStore.getState().reset()
  })

  it("returns null getTickData and seek when no demoId", () => {
    const { result } = renderHook(() => useTickData())
    expect(result.current.getTickData(0)).toBeNull()
    // seek should not throw
    result.current.seek(0)
  })

  it("creates buffer when demoId is set", () => {
    act(() => {
      useViewerStore.getState().setDemoId("demo-1")
    })

    const { result } = renderHook(() => useTickData())
    // Buffer exists — getTickData returns null (no data fetched yet) but doesn't throw
    expect(result.current.getTickData(0)).toBeNull()
  })

  it("disposes buffer on demoId change", () => {
    act(() => {
      useViewerStore.getState().setDemoId("demo-1")
    })

    const { result, rerender } = renderHook(() => useTickData())

    // Get reference — next call shouldn't throw
    result.current.getTickData(0)

    act(() => {
      useViewerStore.getState().setDemoId("demo-2")
    })

    rerender()

    // Should not throw — new buffer created for demo-2
    expect(result.current.getTickData(0)).toBeNull()
  })

  it("disposes buffer on unmount", () => {
    act(() => {
      useViewerStore.getState().setDemoId("demo-1")
    })

    const { result, unmount } = renderHook(() => useTickData())
    result.current.getTickData(0)

    // Should not throw
    unmount()
  })

  it("exposes seek function", () => {
    act(() => {
      useViewerStore.getState().setDemoId("demo-1")
    })

    const { result } = renderHook(() => useTickData())
    // seek should not throw
    expect(() => result.current.seek(500)).not.toThrow()
  })
})
