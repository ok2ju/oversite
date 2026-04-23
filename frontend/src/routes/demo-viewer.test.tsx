import { vi, describe, it, expect, beforeEach, afterEach } from "vitest"
import { screen, waitFor, cleanup } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import { mockDemos } from "@/test/fixtures"
import { useViewerStore } from "@/stores/viewer"
import { Route, Routes } from "react-router-dom"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

import DemoViewerPage from "@/routes/demo-viewer"

// Mock PixiJS viewer components to avoid canvas initialization
vi.mock("@/components/viewer/viewer-canvas", () => ({
  ViewerCanvas: () => <div data-testid="viewer-canvas" />,
}))
vi.mock("@/components/viewer/playback-controls", () => ({
  PlaybackControls: () => <div data-testid="playback-controls" />,
}))
vi.mock("@/components/viewer/round-selector", () => ({
  RoundSelector: () => <div data-testid="round-selector" />,
}))
vi.mock("@/components/viewer/scoreboard", () => ({
  Scoreboard: () => <div data-testid="scoreboard" />,
}))

function renderViewerPage(demoId = "1") {
  return renderWithProviders(
    <Routes>
      <Route path="/demos/:id" element={<DemoViewerPage />} />
    </Routes>,
    { initialRoute: `/demos/${demoId}` },
  )
}

describe("DemoViewerPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useViewerStore.getState().reset()
    // Restore default mock after clearAllMocks resets implementations
    mockAppBindings.GetDemoByID.mockImplementation((id: string) => {
      const demo = mockDemos.find((d) => String(d.id) === id)
      if (!demo) return Promise.reject(new Error("demo not found"))
      return Promise.resolve(demo)
    })
  })

  afterEach(() => {
    cleanup()
  })

  it("shows loading state while fetching demo", () => {
    // Make the mock hang to keep loading state visible
    mockAppBindings.GetDemoByID.mockReturnValue(new Promise(() => {}))

    renderViewerPage()

    expect(screen.queryByTestId("demo-viewer")).not.toBeInTheDocument()
    expect(screen.queryByTestId("viewer-canvas")).not.toBeInTheDocument()
  })

  it("renders viewer canvas when demo is ready", async () => {
    renderViewerPage("1")

    await waitFor(() => {
      expect(screen.getByTestId("demo-viewer")).toBeInTheDocument()
    })

    expect(screen.getByTestId("viewer-canvas")).toBeInTheDocument()
    expect(screen.getByTestId("playback-controls")).toBeInTheDocument()
  })

  it("shows error state when demo not found", async () => {
    mockAppBindings.GetDemoByID.mockRejectedValueOnce(
      new Error("demo not found"),
    )

    renderViewerPage("999")

    await waitFor(() => {
      expect(
        screen.getByText(/demo not found or failed to load/i),
      ).toBeInTheDocument()
    })
  })

  it("shows not-ready state for non-ready demos", async () => {
    const parsingDemo = { ...mockDemos[1] } // status: "parsing"
    mockAppBindings.GetDemoByID.mockResolvedValueOnce(parsingDemo)

    renderViewerPage("2")

    await waitFor(() => {
      expect(screen.getByText(/not ready for viewing/i)).toBeInTheDocument()
    })
  })

  it("sets viewer store state when demo loads", async () => {
    renderViewerPage("1")

    await waitFor(() => {
      expect(screen.getByTestId("demo-viewer")).toBeInTheDocument()
    })

    const state = useViewerStore.getState()
    expect(state.demoId).toBe("1")
    expect(state.mapName).toBe("de_dust2")
    expect(state.totalTicks).toBe(128000)
  })

  it("resets viewer store on unmount", async () => {
    const { unmount } = renderViewerPage("1")

    await waitFor(() => {
      expect(screen.getByTestId("demo-viewer")).toBeInTheDocument()
    })

    expect(useViewerStore.getState().demoId).toBe("1")

    unmount()

    expect(useViewerStore.getState().demoId).toBeNull()
    expect(useViewerStore.getState().mapName).toBeNull()
    expect(useViewerStore.getState().totalTicks).toBe(0)
  })
})
