import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, cleanup } from "@testing-library/react"
import { useViewerStore } from "@/stores/viewer"

const mockDestroy = vi.fn()
const mockTickerStart = vi.fn()
const mockTickerStop = vi.fn()
const mockTickerAdd = vi.fn()
const mockTickerRemove = vi.fn()
const mockTicker = {
  start: mockTickerStart,
  stop: mockTickerStop,
  add: mockTickerAdd,
  remove: mockTickerRemove,
  speed: 1,
}

const mockAddLayer = vi.fn().mockReturnValue({ addChild: vi.fn(), removeChild: vi.fn() })

const mockApp = {
  initialized: true,
  ticker: mockTicker,
  destroy: mockDestroy,
  addLayer: mockAddLayer,
}

const mockCreateViewerApp = vi.fn().mockResolvedValue(mockApp)

vi.mock("@/lib/pixi/app", () => ({
  createViewerApp: (...args: unknown[]) => mockCreateViewerApp(...args),
}))

const mockSetMap = vi.fn().mockResolvedValue(undefined)
const mockClear = vi.fn()
const mockMapLayerDestroy = vi.fn()

vi.mock("@/lib/pixi/layers/map-layer", () => {
  return {
    MapLayer: class MockMapLayer {
      calibration = null
      setMap = mockSetMap
      clear = mockClear
      destroy = mockMapLayerDestroy
    },
  }
})

const mockSetRoster = vi.fn()
const mockOnPlayerClick = vi.fn()
const mockPlayerLayerDestroy = vi.fn()

vi.mock("@/lib/pixi/layers/player-layer", () => {
  return {
    PlayerLayer: class MockPlayerLayer {
      setRoster = mockSetRoster
      onPlayerClick = mockOnPlayerClick
      update = vi.fn()
      destroy = mockPlayerLayerDestroy
    },
  }
})

const mockEventLayerSetEvents = vi.fn()
const mockEventLayerUpdate = vi.fn()
const mockEventLayerDestroy = vi.fn()

vi.mock("@/lib/pixi/layers/event-layer", () => {
  return {
    EventLayer: class MockEventLayer {
      setEvents = mockEventLayerSetEvents
      update = mockEventLayerUpdate
      destroy = mockEventLayerDestroy
    },
  }
})

vi.mock("@/hooks/use-roster", () => ({
  fetchRoster: vi.fn().mockResolvedValue([]),
}))

const mockTickBufferDispose = vi.fn()
const mockGetTickData = vi.fn().mockReturnValue([])

vi.mock("@/lib/pixi/tick-buffer", () => ({
  TickBuffer: class MockTickBuffer {
    getTickData = mockGetTickData
    dispose = mockTickBufferDispose
    seek = vi.fn()
  },
}))

vi.mock("@/hooks/use-game-events", () => ({
  useGameEvents: vi.fn().mockReturnValue({ data: undefined }),
}))

import { ViewerCanvas } from "./viewer-canvas"

describe("ViewerCanvas", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockTicker.speed = 1
    mockApp.initialized = true
    mockAddLayer.mockReturnValue({ addChild: vi.fn(), removeChild: vi.fn() })
    mockCreateViewerApp.mockResolvedValue(mockApp)
    mockSetMap.mockResolvedValue(undefined)
    mockOnPlayerClick.mockReset()
    mockPlayerLayerDestroy.mockReset()
    mockTickBufferDispose.mockReset()
    useViewerStore.getState().reset()
    // Re-set return values cleared by clearAllMocks
    mockSetMap.mockResolvedValue(undefined)
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

    // Zustand subscribers fire synchronously — no async wait needed
    expect(mockTickerStart).not.toHaveBeenCalled()
  })

  it("creates player layer after map layer", async () => {
    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    // addLayer should be called twice: "map" then "players"
    expect(mockAddLayer).toHaveBeenCalledWith("map")
    expect(mockAddLayer).toHaveBeenCalledWith("players")
    const calls = mockAddLayer.mock.calls.map((c) => c[0])
    expect(calls.indexOf("map")).toBeLessThan(calls.indexOf("players"))
  })

  it("destroys player layer on unmount", async () => {
    const { unmount } = render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    unmount()
    expect(mockPlayerLayerDestroy).toHaveBeenCalled()
  })

  it("registers click handler on player layer", async () => {
    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockOnPlayerClick).toHaveBeenCalledWith(expect.any(Function))
    })
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

    // Flush microtask queue for the .then() callback
    await vi.waitFor(() => {
      expect(mockDestroy).toHaveBeenCalled()
    })

    // Subscriptions should not be set up after late init
    mockTickerStart.mockClear()
    mockTickerStop.mockClear()
    useViewerStore.getState().togglePlay()
    expect(mockTickerStart).not.toHaveBeenCalled()
  })
})
