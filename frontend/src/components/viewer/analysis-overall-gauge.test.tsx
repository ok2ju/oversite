import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import { useViewerStore } from "@/stores/viewer"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, resetAppBindings } from "@/test/mocks/bindings"
import { AnalysisOverallGauge } from "./analysis-overall-gauge"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

describe("AnalysisOverallGauge", () => {
  beforeEach(() => {
    resetAppBindings()
    useViewerStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders Overall: 62/100 for an analysis row with overall_score 62", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetPlayerAnalysis.mockResolvedValueOnce({
      steam_id: "STEAM_A",
      overall_score: 62,
      trade_pct: 0.62,
      avg_trade_ticks: 90,
      extras: null,
    })

    renderWithProviders(<AnalysisOverallGauge />)

    expect(
      await screen.findByTestId("analysis-overall-gauge"),
    ).toHaveTextContent("Overall: 62/100")
  })

  it("renders nothing when the binding returns the zero value", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetPlayerAnalysis.mockResolvedValueOnce({
      steam_id: "",
      overall_score: 0,
      trade_pct: 0,
      avg_trade_ticks: 0,
      extras: null,
    })

    renderWithProviders(<AnalysisOverallGauge />)

    // Wait one microtask so the resolved query settles, then assert absence.
    await Promise.resolve()
    expect(
      screen.queryByTestId("analysis-overall-gauge"),
    ).not.toBeInTheDocument()
  })

  it("renders nothing while the query is pending", () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    // Never-resolving promise keeps the query in a loading state for the
    // duration of the assertion.
    mockAppBindings.GetPlayerAnalysis.mockImplementationOnce(
      () => new Promise(() => {}),
    )

    renderWithProviders(<AnalysisOverallGauge />)

    expect(
      screen.queryByTestId("analysis-overall-gauge"),
    ).not.toBeInTheDocument()
  })
})
