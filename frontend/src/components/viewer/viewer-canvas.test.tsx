import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, cleanup } from "@testing-library/react"
import { useViewerStore } from "@/stores/viewer"

const mockDestroy = vi.fn()
const mockTickerStart = vi.fn()
const mockTickerStop = vi.fn()
const mockTicker = { start: mockTickerStart, stop: mockTickerStop, speed: 1 }

const mockApp = {
  initialized: true,
  ticker: mockTicker,
  destroy: mockDestroy,
}

const mockCreateViewerApp = vi.fn().mockResolvedValue(mockApp)

vi.mock("@/lib/pixi/app", () => ({
  createViewerApp: (...args: unknown[]) => mockCreateViewerApp(...args),
}))

import { ViewerCanvas } from "./viewer-canvas"

describe("ViewerCanvas", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockTicker.speed = 1
    mockApp.initialized = true
    mockCreateViewerApp.mockResolvedValue(mockApp)
    useViewerStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders container div with data-testid", () => {
    render(<ViewerCanvas />)

    expect(screen.getByTestId("viewer-canvas-container")).toBeInTheDocument()
  })

  it("calls createViewerApp with container element on mount", async () => {
    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalledWith({
        container: screen.getByTestId("viewer-canvas-container"),
      })
    })
  })

  it("calls destroy on unmount", async () => {
    const { unmount } = render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    unmount()

    expect(mockDestroy).toHaveBeenCalled()
  })

  it("stops ticker when isPlaying is false (initial state)", async () => {
    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockTickerStop).toHaveBeenCalled()
    })
  })

  it("starts ticker when isPlaying changes to true", async () => {
    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    mockTickerStart.mockClear()
    useViewerStore.getState().togglePlay()

    await vi.waitFor(() => {
      expect(mockTickerStart).toHaveBeenCalled()
    })
  })

  it("updates ticker.speed when speed changes", async () => {
    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    useViewerStore.getState().setSpeed(2)

    await vi.waitFor(() => {
      expect(mockTicker.speed).toBe(2)
    })
  })

  it("cleans up Zustand subscriptions on unmount", async () => {
    const { unmount } = render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    unmount()

    // After unmount, changing store should not affect ticker
    mockTickerStart.mockClear()
    mockTickerStop.mockClear()
    useViewerStore.getState().togglePlay()

    // Give any potential async handlers time to fire
    await new Promise((r) => setTimeout(r, 50))

    expect(mockTickerStart).not.toHaveBeenCalled()
  })

  it("handles async init completing after cleanup (StrictMode guard)", async () => {
    // Simulate slow init that resolves after component unmounts
    let resolveInit: (value: typeof mockApp) => void
    mockCreateViewerApp.mockReturnValue(
      new Promise((resolve) => {
        resolveInit = resolve
      })
    )

    const { unmount } = render(<ViewerCanvas />)
    unmount()

    // Resolve init after unmount — should not throw
    resolveInit!(mockApp)
    await new Promise((r) => setTimeout(r, 50))

    // destroy should still be called for cleanup
    expect(mockDestroy).toHaveBeenCalled()
  })
})
