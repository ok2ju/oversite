import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import { useViewerStore } from "@/stores/viewer"
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
})
