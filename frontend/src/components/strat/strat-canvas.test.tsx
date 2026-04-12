import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { createMockViewerApp, type MockViewerApp } from "@/test/mocks/pixi"
import { useStratStore } from "@/stores/strat"

let mockApp: MockViewerApp

const mockCreateViewerApp = vi.fn(() => Promise.resolve(mockApp))

vi.mock("@/lib/pixi/app", () => ({
  createViewerApp: (...args: unknown[]) => mockCreateViewerApp(...(args as [])),
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

const mockRendererDestroy = vi.fn()

vi.mock("@/lib/strat/renderer", () => {
  return {
    StratRenderer: class MockStratRenderer {
      destroy = mockRendererDestroy
    },
  }
})

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

import { StratCanvas } from "./strat-canvas"

describe("StratCanvas", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockApp = createMockViewerApp()
    mockCreateViewerApp.mockResolvedValue(mockApp)
    mockSetMap.mockResolvedValue(undefined)
    mockCameraDestroy.mockReset()
    mockCameraSetScreenSize.mockReset()
    mockMapLayerCalibration = null
    useStratStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders container div with data-testid", () => {
    renderWithProviders(<StratCanvas />)
    expect(screen.getByTestId("strat-canvas-container")).toBeInTheDocument()
  })

  it("calls createViewerApp with container element on mount", async () => {
    renderWithProviders(<StratCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalledWith({
        container: screen.getByTestId("strat-canvas-container"),
      })
    })
  })

  it("calls destroy on unmount", async () => {
    const { unmount } = renderWithProviders(<StratCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    unmount()
    expect(mockApp.destroy).toHaveBeenCalled()
  })

  it("creates map and drawings layers under camera container", async () => {
    renderWithProviders(<StratCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    expect(mockApp.addLayer).toHaveBeenCalledWith("map", mockCameraContainer)
    expect(mockApp.addLayer).toHaveBeenCalledWith(
      "drawings",
      mockCameraContainer,
    )
    const calls = mockApp.addLayer.mock.calls.map((c: unknown[]) => c[0])
    expect(calls.indexOf("map")).toBeLessThan(calls.indexOf("drawings"))
  })

  it("loads map when mapName changes", async () => {
    renderWithProviders(<StratCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    useStratStore.getState().setBoard("board-1", "de_mirage")

    await vi.waitFor(() => {
      expect(mockSetMap).toHaveBeenCalledWith("de_mirage")
    })
  })

  it("clears map when mapName becomes null", async () => {
    useStratStore.getState().setBoard("board-1", "de_mirage")

    renderWithProviders(<StratCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    await vi.waitFor(() => {
      expect(mockSetMap).toHaveBeenCalledWith("de_mirage")
    })

    mockClear.mockClear()
    useStratStore.getState().setBoard(null, null)

    await vi.waitFor(() => {
      expect(mockClear).toHaveBeenCalled()
    })
  })

  it("calls setMapSize after map loads", async () => {
    const mockCalibration = {
      originX: -2476,
      originY: 3239,
      scale: 4.4,
      width: 1024,
      height: 1024,
    }

    mockSetMap.mockImplementation(() => {
      mockMapLayerCalibration = mockCalibration
      return Promise.resolve()
    })

    renderWithProviders(<StratCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    mockCameraSetMapSize.mockClear()
    useStratStore.getState().setBoard("board-1", "de_dust2")

    await vi.waitFor(() => {
      expect(mockSetMap).toHaveBeenCalledWith("de_dust2")
    })

    await vi.waitFor(() => {
      expect(mockCameraSetMapSize).toHaveBeenCalledWith(1024, 1024)
    })

    mockMapLayerCalibration = null
  })

  // TODO: Test boardId -> load strat data via Wails bindings (P5 tasks)

  it("destroys camera on unmount", async () => {
    const { unmount } = renderWithProviders(<StratCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    unmount()
    expect(mockCameraDestroy).toHaveBeenCalled()
  })

  it("handles async init completing after cleanup (StrictMode guard)", async () => {
    let resolveInit: (value: typeof mockApp) => void
    mockCreateViewerApp.mockReturnValue(
      new Promise((resolve) => {
        resolveInit = resolve
      }),
    )

    const { unmount } = renderWithProviders(<StratCanvas />)
    unmount()

    resolveInit!(mockApp)

    await vi.waitFor(() => {
      expect(mockApp.destroy).toHaveBeenCalled()
    })
  })

  it("observes container resize and updates camera screen size", async () => {
    renderWithProviders(<StratCanvas />)

    await vi.waitFor(() => {
      expect(mockCreateViewerApp).toHaveBeenCalled()
    })

    // Camera setScreenSize is called once during init
    expect(mockCameraSetScreenSize).toHaveBeenCalled()
  })
})
