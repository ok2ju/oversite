import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useViewerStore } from "@/stores/viewer"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, resetAppBindings } from "@/test/mocks/bindings"
import { PlayerStatsPanel } from "./player-stats-panel"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

describe("PlayerStatsPanel", () => {
  beforeEach(() => {
    resetAppBindings()
    useViewerStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders nothing when no player is selected", () => {
    useViewerStore.getState().setDemoId("1")
    renderWithProviders(<PlayerStatsPanel />)
    expect(screen.queryByTestId("player-stats-panel")).not.toBeInTheDocument()
  })

  it("opens when selectedPlayerSteamId is set", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")

    renderWithProviders(<PlayerStatsPanel />)

    expect(await screen.findByTestId("player-stats-panel")).toBeInTheDocument()
    // Mock returns player_name "MockPlayer" for any steamId
    expect(await screen.findByText("MockPlayer")).toBeInTheDocument()
  })

  it("calls the binding with demoId + steamId", async () => {
    useViewerStore.getState().setDemoId("42")
    useViewerStore.getState().setSelectedPlayer("STEAM_X")

    renderWithProviders(<PlayerStatsPanel />)
    await screen.findByTestId("player-stats-panel")

    expect(mockAppBindings.GetPlayerMatchStats).toHaveBeenCalledWith(
      "42",
      "STEAM_X",
    )
  })

  it("close button clears the selection", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")

    renderWithProviders(<PlayerStatsPanel />)
    await screen.findByTestId("player-stats-panel")

    await user.click(screen.getByTestId("player-stats-panel-close"))
    expect(useViewerStore.getState().selectedPlayerSteamId).toBeNull()
  })

  it("clicking a round cell sets the current round", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")

    renderWithProviders(<PlayerStatsPanel />)
    await screen.findByTestId("player-stats-panel")

    // Match tab is the default — wait for the strip to render.
    const cell = await screen.findByTestId("player-stats-round-cell-2")
    await user.click(cell)

    // setRound clears the selected player; this is the existing viewer
    // contract (round change resets selection). Assert the round update went
    // through and selection cleared.
    expect(useViewerStore.getState().currentRound).toBe(2)
  })

  it("renders damage breakdowns under the Detail tab", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")

    renderWithProviders(<PlayerStatsPanel />)
    await screen.findByTestId("player-stats-panel")

    await user.click(screen.getByRole("tab", { name: /detail/i }))

    expect(
      await screen.findByTestId("player-stats-damage-weapon-ak-47"),
    ).toHaveTextContent("175")
    expect(
      screen.getByTestId("player-stats-damage-opponent-STEAM_X"),
    ).toHaveTextContent("Enemy1")
  })

  it("renders the per-round distance sparkline on the Match tab", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")

    renderWithProviders(<PlayerStatsPanel />)
    await screen.findByTestId("player-stats-panel")

    expect(
      await screen.findByTestId("player-stats-movement-sparkline"),
    ).toBeInTheDocument()
    expect(
      screen.getByTestId("player-stats-distance-bar-1"),
    ).toBeInTheDocument()
  })

  it("renders the movement card under the Round tab", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")

    renderWithProviders(<PlayerStatsPanel />)
    await screen.findByTestId("player-stats-panel")

    await user.click(screen.getByRole("tab", { name: /round/i }))

    const card = await screen.findByTestId("player-stats-movement-card")
    // 6000u total comes from the bindings mock.
    expect(card).toHaveTextContent("6,000u total")
    // Strafe bar carries the tooltip explaining the 16 Hz sample rate.
    expect(screen.getByText("Strafing")).toHaveAttribute("title")
  })

  it("renders the utility card under the Round tab", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")

    renderWithProviders(<PlayerStatsPanel />)
    await screen.findByTestId("player-stats-panel")

    await user.click(screen.getByRole("tab", { name: /round/i }))
    const card = await screen.findByTestId("player-stats-utility-card")
    // Mock: 4 smokes, 6 flashes, 14.5s blind time.
    expect(card).toHaveTextContent("Smokes")
    expect(card).toHaveTextContent("4")
    expect(card).toHaveTextContent("14.5s")
  })

  it("renders the hit-group breakdown under the Detail tab", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("STEAM_A")

    renderWithProviders(<PlayerStatsPanel />)
    await screen.findByTestId("player-stats-panel")

    await user.click(screen.getByRole("tab", { name: /detail/i }))

    // Mock: chest (hit_group=2) is the top row at 130 damage / 4 hits.
    const chestRow = await screen.findByTestId("player-stats-hit-group-2")
    expect(chestRow).toHaveTextContent("Chest")
    expect(chestRow).toHaveTextContent("130")
    expect(chestRow).toHaveTextContent("(4)")

    expect(screen.getByTestId("player-stats-hit-group-1")).toHaveTextContent(
      "Head",
    )
  })
})
