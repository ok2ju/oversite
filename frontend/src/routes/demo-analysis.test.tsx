import { vi, describe, it, expect, beforeEach, afterEach } from "vitest"
import { screen, waitFor, cleanup } from "@testing-library/react"
import { Route, Routes } from "react-router-dom"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import { mockDemos } from "@/test/fixtures"
import { useViewerStore } from "@/stores/viewer"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

import DemoAnalysisPage from "@/routes/demo-analysis"

function renderAnalysisPage(demoId = "1") {
  return renderWithProviders(
    <Routes>
      <Route path="/demos/:id/analysis" element={<DemoAnalysisPage />} />
    </Routes>,
    { initialRoute: `/demos/${demoId}/analysis` },
  )
}

describe("DemoAnalysisPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useViewerStore.getState().reset()
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
    mockAppBindings.GetDemoByID.mockReturnValue(new Promise(() => {}))

    renderAnalysisPage()

    expect(screen.queryByTestId("demo-analysis")).not.toBeInTheDocument()
  })

  it("shows error state when demo not found", async () => {
    mockAppBindings.GetDemoByID.mockRejectedValueOnce(
      new Error("demo not found"),
    )

    renderAnalysisPage("999")

    await waitFor(() => {
      expect(
        screen.getByText(/demo not found or failed to load/i),
      ).toBeInTheDocument()
    })
  })

  it("shows not-ready state for non-ready demos", async () => {
    const parsingDemo = { ...mockDemos[1] } // status: "parsing"
    mockAppBindings.GetDemoByID.mockResolvedValueOnce(parsingDemo)

    renderAnalysisPage("2")

    await waitFor(() => {
      expect(screen.getByText(/not ready for viewing/i)).toBeInTheDocument()
    })
  })

  it("renders the gauge, Trades card, and round bars on the happy path", async () => {
    mockAppBindings.GetPlayerAnalysis.mockResolvedValue({
      steam_id: "STEAM_A",
      overall_score: 62,
      trade_pct: 0.62,
      avg_trade_ticks: 90,
      extras: { aim_pct: 0.74, standing_shot_pct: 0.62 },
    })
    mockAppBindings.GetPlayerRoundAnalysis.mockResolvedValue([
      { steam_id: "STEAM_A", round_number: 1, trade_pct: 1, extras: null },
    ])

    renderAnalysisPage("1")

    // The page mounts useViewerStore via initDemo, which clears
    // selectedPlayerSteamId. Once the page lands, set the selected player so
    // the shared gauge / Trades card / round-bars components light up.
    await waitFor(() => {
      expect(screen.getByTestId("demo-analysis")).toBeInTheDocument()
    })
    useViewerStore.getState().setSelectedPlayer("STEAM_A")

    await waitFor(() => {
      expect(screen.getByTestId("analysis-overall-gauge")).toBeInTheDocument()
    })
    expect(screen.getByTestId("category-card-trade")).toBeInTheDocument()
    expect(screen.getByTestId("category-card-aim")).toBeInTheDocument()
    expect(screen.getByTestId("category-card-movement")).toBeInTheDocument()
    expect(screen.getByTestId("round-trade-bars")).toBeInTheDocument()
  })

  it("marks the Analysis tab active in the route tab strip", async () => {
    renderAnalysisPage("1")

    await waitFor(() => {
      expect(screen.getByTestId("demo-route-tab-analysis")).toBeInTheDocument()
    })

    expect(screen.getByTestId("demo-route-tab-analysis")).toHaveAttribute(
      "data-state",
      "active",
    )
    expect(screen.getByTestId("demo-route-tab-viewer")).toHaveAttribute(
      "data-state",
      "inactive",
    )
  })
})
