import { vi, describe, it, expect, beforeEach, afterEach } from "vitest"
import { screen, waitFor, cleanup } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Route, Routes } from "react-router-dom"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import { mockDemos, mockScoreboardEntries } from "@/test/fixtures"
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

  it("auto-selects the first scoreboard player so the debrief lights up", async () => {
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
    mockAppBindings.GetHabitReport.mockResolvedValueOnce({
      demo_id: "1",
      steam_id: "STEAM_A",
      as_of: "2026-05-10T00:00:00Z",
      habits: [
        {
          key: "counter_strafe",
          label: "Counter-strafe",
          description: "Stop before firing",
          unit: "ms",
          direction: "lower",
          value: 240,
          status: "warn",
          good_threshold: 100,
          warn_threshold: 200,
          good_min: 0,
          good_max: 0,
          warn_min: 0,
          warn_max: 0,
          previous_value: null,
          delta: null,
        },
      ],
    })

    renderAnalysisPage("1")

    // No manual setSelectedPlayer — the page reads the scoreboard and
    // auto-selects the first player so the redesigned surfaces render.
    await waitFor(() => {
      expect(screen.getByTestId("verdict-hero")).toBeInTheDocument()
    })
    expect(useViewerStore.getState().selectedPlayerSteamId).toBe(
      mockScoreboardEntries[0].steam_id,
    )
    await waitFor(() => {
      expect(screen.getByTestId("habit-checklist")).toBeInTheDocument()
    })
    expect(screen.getByTestId("match-timeline")).toBeInTheDocument()
    expect(screen.getByTestId("mistakes-feed")).toBeInTheDocument()
  })

  it("switches the active player when the picker changes", async () => {
    mockAppBindings.GetPlayerAnalysis.mockResolvedValue({
      steam_id: "STEAM_A",
      overall_score: 50,
      trade_pct: 0.5,
      avg_trade_ticks: 64,
      extras: null,
    })

    renderAnalysisPage("1")

    await waitFor(() => {
      expect(screen.getByTestId("analysis-player-picker")).toBeInTheDocument()
    })

    const user = userEvent.setup()
    await user.click(screen.getByTestId("analysis-player-picker"))
    const target = mockScoreboardEntries.find(
      (e) => e.steam_id !== mockScoreboardEntries[0].steam_id,
    )!
    await user.click(await screen.findByText(target.player_name))

    await waitFor(() => {
      expect(useViewerStore.getState().selectedPlayerSteamId).toBe(
        target.steam_id,
      )
    })
  })

  it("shows an empty-state message when the scoreboard has no players", async () => {
    mockAppBindings.GetScoreboard.mockResolvedValueOnce([])

    renderAnalysisPage("1")

    await waitFor(() => {
      expect(screen.getByTestId("analysis-no-players")).toBeInTheDocument()
    })
    expect(
      screen.queryByTestId("analysis-player-picker"),
    ).not.toBeInTheDocument()
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
