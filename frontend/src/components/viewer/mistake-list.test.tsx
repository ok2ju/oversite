import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useViewerStore } from "@/stores/viewer"
import { useAnalysisStore } from "@/stores/analysis"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, resetAppBindings } from "@/test/mocks/bindings"
import { MistakeList } from "./mistake-list"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

// Round 7 starts at start_tick 30000, freeze ends at 30100. tickRate 64. Pick
// a mistake tick that lands at exactly 1:23 of round time:
//   30100 + 83 * 64 = 35412
const ROUND_7_FREEZE_END = 30100
const TICK_RATE = 64
const SECONDS_INTO_ROUND = 83 // 1:23
const ROUND_7_TICK = ROUND_7_FREEZE_END + SECONDS_INTO_ROUND * TICK_RATE

describe("MistakeList", () => {
  beforeEach(() => {
    resetAppBindings()
    useViewerStore.getState().reset()
    useAnalysisStore.getState().reset()

    // Tests below need a single round-7 entry to exercise the M:SS formatting.
    mockAppBindings.GetDemoRounds.mockResolvedValue([
      {
        id: "round-7",
        round_number: 7,
        start_tick: 30000,
        freeze_end_tick: ROUND_7_FREEZE_END,
        end_tick: 60000,
        winner_side: "CT",
        win_reason: "TargetBombed",
        ct_score: 4,
        t_score: 3,
        is_overtime: false,
        ct_team_name: "",
        t_team_name: "",
      },
    ])
  })

  afterEach(() => {
    cleanup()
  })

  it("renders nothing when no player is selected", () => {
    useViewerStore.getState().setDemoId("1")
    renderWithProviders(<MistakeList />)
    expect(screen.queryByTestId("mistake-list")).not.toBeInTheDocument()
  })

  it("renders the empty state when the player has no mistakes", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetMistakeTimeline.mockResolvedValueOnce([])

    renderWithProviders(<MistakeList />)

    expect(await screen.findByTestId("mistake-list-empty")).toHaveTextContent(
      "No mistakes",
    )
  })

  it("mounts the analysis overall gauge inside the panel header when the binding returns a non-zero score", async () => {
    useViewerStore.getState().initDemo({
      id: "1",
      mapName: "de_dust2",
      totalTicks: 100000,
      tickRate: TICK_RATE,
    })
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetMistakeTimeline.mockResolvedValueOnce([])
    mockAppBindings.GetPlayerAnalysis.mockResolvedValueOnce({
      steam_id: "STEAM_A",
      overall_score: 62,
      trade_pct: 0.62,
      avg_trade_ticks: 90,
      extras: null,
    })

    renderWithProviders(<MistakeList />)

    expect(
      await screen.findByTestId("analysis-overall-gauge"),
    ).toHaveTextContent("Overall: 62/100")
  })

  it("renders one row per mistake with the canonical text format", async () => {
    useViewerStore.getState().initDemo({
      id: "1",
      mapName: "de_dust2",
      totalTicks: 100000,
      tickRate: TICK_RATE,
    })
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetMistakeTimeline.mockResolvedValueOnce([
      {
        kind: "no_trade_death",
        round_number: 7,
        tick: ROUND_7_TICK,
        steam_id: "STEAM_A",
        extras: { killer_steam_id: "STEAM_B" },
      },
    ])

    renderWithProviders(<MistakeList />)

    const row = await screen.findByTestId("mistake-list-row-0")
    expect(row).toHaveTextContent("Untraded death — round 7, 1:23")
  })

  it("calls the binding with demoId + steamId", async () => {
    useViewerStore.getState().initDemo({
      id: "42",
      mapName: "de_inferno",
      totalTicks: 1000,
      tickRate: TICK_RATE,
    })
    useViewerStore.getState().setSelectedPlayer("STEAM_X")

    renderWithProviders(<MistakeList />)
    await screen.findByTestId("mistake-list")

    expect(mockAppBindings.GetMistakeTimeline).toHaveBeenCalledWith(
      "42",
      "STEAM_X",
    )
  })

  it("clicking a row seeks the viewer to the mistake tick", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().initDemo({
      id: "1",
      mapName: "de_dust2",
      totalTicks: 100000,
      tickRate: TICK_RATE,
    })
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetMistakeTimeline.mockResolvedValueOnce([
      {
        kind: "no_trade_death",
        round_number: 7,
        tick: ROUND_7_TICK,
        steam_id: "STEAM_A",
        extras: { killer_steam_id: "STEAM_B" },
      },
    ])

    renderWithProviders(<MistakeList />)
    const row = await screen.findByTestId("mistake-list-row-0")

    await user.click(row)

    const state = useViewerStore.getState()
    expect(state.currentTick).toBe(ROUND_7_TICK)
    expect(state.selectedPlayerSteamId).toBe("STEAM_A")
    expect(state.isPlaying).toBe(false)
  })

  it("activating a focused row via Enter seeks to the mistake tick", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().initDemo({
      id: "1",
      mapName: "de_dust2",
      totalTicks: 100000,
      tickRate: TICK_RATE,
    })
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetMistakeTimeline.mockResolvedValueOnce([
      {
        kind: "no_trade_death",
        round_number: 7,
        tick: ROUND_7_TICK,
        steam_id: "STEAM_A",
        extras: { killer_steam_id: "STEAM_B" },
      },
    ])

    renderWithProviders(<MistakeList />)
    const row = await screen.findByTestId("mistake-list-row-0")

    row.focus()
    await user.keyboard("{Enter}")

    expect(useViewerStore.getState().currentTick).toBe(ROUND_7_TICK)
  })

  it("renders both kinds in tick order with distinguishable severity badges", async () => {
    // Round 7 entry from the existing fixture is at ROUND_7_TICK; place a
    // util-unused entry earlier in the same round so we can assert the
    // chronological-by-tick render order across kinds.
    const UTIL_UNUSED_TICK = ROUND_7_FREEZE_END + 30 * TICK_RATE // 0:30

    useViewerStore.getState().initDemo({
      id: "1",
      mapName: "de_dust2",
      totalTicks: 100000,
      tickRate: TICK_RATE,
    })
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetMistakeTimeline.mockResolvedValueOnce([
      {
        kind: "died_with_util_unused",
        round_number: 7,
        tick: UTIL_UNUSED_TICK,
        steam_id: "STEAM_A",
        extras: { unused: ["smokegrenade"] },
      },
      {
        kind: "no_trade_death",
        round_number: 7,
        tick: ROUND_7_TICK,
        steam_id: "STEAM_A",
        extras: { killer_steam_id: "STEAM_B" },
      },
    ])

    renderWithProviders(<MistakeList />)
    const firstRow = await screen.findByTestId("mistake-list-row-0")
    const secondRow = await screen.findByTestId("mistake-list-row-1")

    expect(firstRow).toHaveTextContent(
      "Died with utility unused — round 7, 0:30",
    )
    expect(secondRow).toHaveTextContent("Untraded death — round 7, 1:23")

    expect(
      screen.getByTestId("mistake-row-severity-died_with_util_unused"),
    ).toBeInTheDocument()
    expect(
      screen.getByTestId("mistake-row-severity-no_trade_death"),
    ).toBeInTheDocument()
  })

  describe("category filter", () => {
    const UTIL_UNUSED_TICK = ROUND_7_FREEZE_END + 30 * TICK_RATE

    function setupTwoKindFixture() {
      useViewerStore.getState().initDemo({
        id: "1",
        mapName: "de_dust2",
        totalTicks: 100000,
        tickRate: TICK_RATE,
      })
      useViewerStore.getState().setSelectedPlayer("STEAM_A")
      mockAppBindings.GetMistakeTimeline.mockResolvedValue([
        {
          kind: "died_with_util_unused",
          round_number: 7,
          tick: UTIL_UNUSED_TICK,
          steam_id: "STEAM_A",
          extras: { unused: ["smokegrenade"] },
        },
        {
          kind: "no_trade_death",
          round_number: 7,
          tick: ROUND_7_TICK,
          steam_id: "STEAM_A",
          extras: { killer_steam_id: "STEAM_B" },
        },
      ])
    }

    it("renders one badge per non-empty category with the correct count", async () => {
      setupTwoKindFixture()

      renderWithProviders(<MistakeList />)
      await screen.findByTestId("mistake-list-row-0")

      const tradeBadge = screen.getByTestId("mistake-category-badge-trade")
      const utilityBadge = screen.getByTestId("mistake-category-badge-utility")
      expect(tradeBadge).toHaveTextContent("Trade 1")
      expect(utilityBadge).toHaveTextContent("Utility 1")
      expect(
        screen.queryByTestId("mistake-category-badge-other"),
      ).not.toBeInTheDocument()
    })

    it("clicking a badge filters the list to that category", async () => {
      const user = userEvent.setup()
      setupTwoKindFixture()

      renderWithProviders(<MistakeList />)
      await screen.findByTestId("mistake-list-row-1")

      await user.click(screen.getByTestId("mistake-category-badge-trade"))

      expect(useAnalysisStore.getState().selectedCategory).toBe("trade")
      const rows = screen.getAllByTestId(/^mistake-list-row-/)
      expect(rows).toHaveLength(1)
      expect(rows[0]).toHaveTextContent("Untraded death — round 7, 1:23")
      expect(
        screen.queryByText(/Died with utility unused/),
      ).not.toBeInTheDocument()
    })

    it("clicking the active badge clears the filter", async () => {
      const user = userEvent.setup()
      setupTwoKindFixture()

      renderWithProviders(<MistakeList />)
      await screen.findByTestId("mistake-list-row-1")

      const tradeBadge = screen.getByTestId("mistake-category-badge-trade")
      await user.click(tradeBadge)
      expect(screen.getAllByTestId(/^mistake-list-row-/)).toHaveLength(1)

      await user.click(screen.getByTestId("mistake-category-badge-trade"))

      expect(useAnalysisStore.getState().selectedCategory).toBeNull()
      expect(screen.getAllByTestId(/^mistake-list-row-/)).toHaveLength(2)
    })

    it("clears the filter when the selected player changes", async () => {
      const user = userEvent.setup()
      setupTwoKindFixture()

      renderWithProviders(<MistakeList />)
      await screen.findByTestId("mistake-list-row-1")

      await user.click(screen.getByTestId("mistake-category-badge-trade"))
      expect(useAnalysisStore.getState().selectedCategory).toBe("trade")

      useViewerStore.getState().setSelectedPlayer("STEAM_B")

      await screen.findByTestId("mistake-list")
      expect(useAnalysisStore.getState().selectedCategory).toBeNull()
    })
  })
})
