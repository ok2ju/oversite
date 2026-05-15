import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useViewerStore } from "@/stores/viewer"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings } from "@/test/mocks/bindings"
import type { PlayerRosterEntry } from "@/types/roster"
import { TeamBars } from "./team-bars"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

const ROSTER: PlayerRosterEntry[] = [
  {
    steam_id: "76561198000000001",
    player_name: "PlayerOne",
    team_side: "CT",
  },
  {
    steam_id: "76561198000000002",
    player_name: "PlayerTwo",
    team_side: "CT",
  },
  {
    steam_id: "76561198000000003",
    player_name: "PlayerThree",
    team_side: "T",
  },
  {
    steam_id: "76561198000000004",
    player_name: "PlayerFour",
    team_side: "T",
  },
]

describe("TeamBars player card selection", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
    useViewerStore.getState().initDemo({
      id: "1",
      mapName: "de_dust2",
      totalTicks: 128000,
      tickRate: 64,
    })
    mockAppBindings.GetRoundRoster.mockResolvedValue(
      ROSTER as unknown as never[],
    )
  })

  afterEach(() => {
    cleanup()
    mockAppBindings.GetRoundRoster.mockReset()
  })

  it("selects a player when their card is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<TeamBars />)

    const card = await screen.findByTestId("team-bar-row-76561198000000003")
    await user.click(card)

    expect(useViewerStore.getState().selectedPlayerSteamId).toBe(
      "76561198000000003",
    )
  })

  it("deselects when the selected card is clicked again", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setSelectedPlayer("76561198000000001")
    renderWithProviders(<TeamBars />)

    const card = await screen.findByTestId("team-bar-row-76561198000000001")
    await user.click(card)

    expect(useViewerStore.getState().selectedPlayerSteamId).toBeNull()
  })

  it("switches selection when a different card is clicked", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setSelectedPlayer("76561198000000001")
    renderWithProviders(<TeamBars />)

    const card = await screen.findByTestId("team-bar-row-76561198000000002")
    await user.click(card)

    expect(useViewerStore.getState().selectedPlayerSteamId).toBe(
      "76561198000000002",
    )
  })

  it("marks the selected card with aria-pressed and the team-colored ring", async () => {
    useViewerStore.getState().setSelectedPlayer("76561198000000003")
    renderWithProviders(<TeamBars />)

    const card = await screen.findByTestId("team-bar-row-76561198000000003")
    expect(card).toHaveAttribute("aria-pressed", "true")
    expect(card.className).toContain("ring-amber-400/70")

    const other = await screen.findByTestId("team-bar-row-76561198000000001")
    expect(other).toHaveAttribute("aria-pressed", "false")
  })
})
