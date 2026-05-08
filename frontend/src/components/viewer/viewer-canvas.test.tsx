import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import { renderWithProviders as render } from "@/test/render"
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

const mockAddLayer = vi
  .fn()
  .mockReturnValue({ addChild: vi.fn(), removeChild: vi.fn() })
const mockCanvas = document.createElement("canvas")

const mockApp = {
  initialized: true,
  ticker: mockTicker,
  destroy: mockDestroy,
  addLayer: mockAddLayer,
  stage: { addChild: vi.fn() },
  canvas: mockCanvas,
}

const mockCreateViewerApp = vi.fn().mockResolvedValue(mockApp)

vi.mock("@/lib/pixi/app", () => ({
  createViewerApp: (...args: unknown[]) => mockCreateViewerApp(...args),
}))

const mockSetMap = vi.fn().mockResolvedValue(undefined)
const mockClear = vi.fn()
const mockMapLayerDestroy = vi.fn()
let mockMapLayerCalibration: unknown = null

vi.mock("@/lib/pixi/layers/map-layer", () => {
  return {
    MapLayer: class MockMapLayer {
      get calibration() {
        return mockMapLayerCalibration
      }
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
  useAllRosters: vi.fn().mockReturnValue({ data: undefined }),
}))

const mockTickBufferDispose = vi.fn()
const mockGetTickData = vi.fn().mockReturnValue([])
const mockGetFramePair = vi.fn().mockReturnValue({ current: null, next: null })

vi.mock("@/lib/pixi/tick-buffer", () => ({
  TickBuffer: class MockTickBuffer {
    getTickData = mockGetTickData
    getFramePair = mockGetFramePair
    dispose = mockTickBufferDispose
    seek = vi.fn()
  },
}))

const mockCameraDestroy = vi.fn()
const mockCameraSetScreenSize = vi.fn()
const mockCameraSetMapSize = vi.fn()
const mockCameraResetView = vi.fn()
const mockCameraContainer = {
  addChild: vi.fn(),
  removeChild: vi.fn(),
  position: { set: vi.fn() },
  scale: { set: vi.fn() },
  label: "camera-viewport",
}

vi.mock("@/lib/pixi/camera", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/pixi/camera")>()
  return {
    ...actual,
    Camera: class MockCamera {
      container = mockCameraContainer
      setScreenSize = mockCameraSetScreenSize
      setMapSize = mockCameraSetMapSize
      resetView = mockCameraResetView
      destroy = mockCameraDestroy
    },
  }
})

vi.mock("@/hooks/use-game-events", () => ({
  useGameEvents: vi.fn().mockReturnValue({ data: undefined }),
}))

vi.mock("@/hooks/use-rounds", () => ({
  useRounds: vi.fn().mockReturnValue({ data: undefined }),
}))

import { ViewerCanvas } from "./viewer-canvas"

describe("ViewerCanvas", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockTicker.speed = 1
    mockApp.initialized = true
    mockApp.stage = { addChild: vi.fn() }
    mockAddLayer.mockReturnValue({ addChild: vi.fn(), removeChild: vi.fn() })
    mockCreateViewerApp.mockResolvedValue(mockApp)
    mockSetMap.mockResolvedValue(undefined)
    mockOnPlayerClick.mockReset()
    mockPlayerLayerDestroy.mockReset()
    mockTickBufferDispose.mockReset()
    mockCameraDestroy.mockReset()
    mockCameraSetScreenSize.mockReset()
    mockMapLayerCalibration = null
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

  it("registers a ticker callback that renders every frame", async () => {
    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockTickerAdd).toHaveBeenCalledWith(expect.any(Function))
    })
  })

  it("removes the ticker callback on unmount", async () => {
    const { unmount } = render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockTickerAdd).toHaveBeenCalled()
    })

    const registeredFn = mockTickerAdd.mock.calls[0][0]
    unmount()

    expect(mockTickerRemove).toHaveBeenCalledWith(registeredFn)
  })

  it("creates layers as children of camera container", async () => {
    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    // addLayer should be called with camera container as parent
    expect(mockAddLayer).toHaveBeenCalledWith("map", mockCameraContainer)
    expect(mockAddLayer).toHaveBeenCalledWith("players", mockCameraContainer)
    expect(mockAddLayer).toHaveBeenCalledWith("events", mockCameraContainer)
    const calls = mockAddLayer.mock.calls.map((c: unknown[]) => c[0])
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

  it("destroys camera on unmount", async () => {
    const { unmount } = render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    unmount()
    expect(mockCameraDestroy).toHaveBeenCalled()
  })

  it("resets camera view on demo change", async () => {
    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    mockCameraResetView.mockClear()
    useViewerStore.getState().setDemoId("demo-123")

    await vi.waitFor(() => {
      expect(mockCameraResetView).toHaveBeenCalled()
    })
  })

  it("resets camera view when resetViewport is triggered", async () => {
    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    mockCameraResetView.mockClear()
    useViewerStore.getState().resetViewport()

    expect(mockCameraResetView).toHaveBeenCalled()
  })

  it("calls setMapSize after map loads", async () => {
    const mockCalibration = {
      originX: -2476,
      originY: 3239,
      scale: 4.4,
      width: 1024,
      height: 1024,
    }

    // Make setMap set calibration via the module-level getter when it resolves
    mockSetMap.mockImplementation(() => {
      mockMapLayerCalibration = mockCalibration
      return Promise.resolve()
    })

    render(<ViewerCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    mockCameraSetMapSize.mockClear()
    useViewerStore.getState().setMapName("de_dust2")

    await vi.waitFor(() => {
      expect(mockSetMap).toHaveBeenCalledWith("de_dust2")
    })

    await vi.waitFor(() => {
      expect(mockCameraSetMapSize).toHaveBeenCalledWith(1024, 1024)
    })

    mockMapLayerCalibration = null
  })

  it("handles async init completing after cleanup (StrictMode guard)", async () => {
    // Simulate slow init that resolves after component unmounts
    let resolveInit: (value: typeof mockApp) => void
    mockCreateViewerApp.mockReturnValue(
      new Promise((resolve) => {
        resolveInit = resolve
      }),
    )

    const { unmount } = render(<ViewerCanvas />)
    unmount()

    // Resolve init after unmount — should not throw
    resolveInit!(mockApp)

    // Flush microtask queue for the .then() callback
    await vi.waitFor(() => {
      expect(mockDestroy).toHaveBeenCalled()
    })

    // Ticker callback should not be registered after late init
    expect(mockTickerAdd).not.toHaveBeenCalled()
  })
})
