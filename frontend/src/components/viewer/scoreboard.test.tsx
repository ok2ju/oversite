import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useViewerStore } from "@/stores/viewer"
import { mockScoreboardEntries } from "@/test/fixtures/demos"
import { renderWithProviders } from "@/test/render"
import { Scoreboard } from "./scoreboard"

vi.mock("@wailsjs/go/main/App", () => ({
  GetScoreboard: vi.fn().mockResolvedValue(mockScoreboardEntries),
}))

describe("Scoreboard", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("does not render when visible is false", () => {
    useViewerStore.getState().setDemoId("1")
    renderWithProviders(<Scoreboard visible={false} />)
    expect(screen.queryByTestId("scoreboard-overlay")).not.toBeInTheDocument()
  })

  it("does not render when no data is available", () => {
    renderWithProviders(<Scoreboard visible={true} />)
    expect(screen.queryByTestId("scoreboard-overlay")).not.toBeInTheDocument()
  })

  it("renders both team sections when visible with data", async () => {
    useViewerStore.getState().setDemoId("1")
    renderWithProviders(<Scoreboard visible={true} />)

    expect(await screen.findByTestId("scoreboard-overlay")).toBeInTheDocument()
    expect(screen.getByTestId("scoreboard-team-ct")).toBeInTheDocument()
    expect(screen.getByTestId("scoreboard-team-t")).toBeInTheDocument()
  })

  it("displays correct player stats", async () => {
    useViewerStore.getState().setDemoId("1")
    renderWithProviders(<Scoreboard visible={true} />)

    await screen.findByTestId("scoreboard-overlay")

    // CT player
    const p1Row = screen.getByTestId("scoreboard-player-76561198000000001")
    expect(p1Row).toHaveTextContent("PlayerOne")
    expect(p1Row).toHaveTextContent("20") // kills
    expect(p1Row).toHaveTextContent("12") // deaths
    expect(p1Row).toHaveTextContent("5") // assists
    expect(p1Row).toHaveTextContent("160") // ADR
    expect(p1Row).toHaveTextContent("50%") // HS%

    // T player
    const p3Row = screen.getByTestId("scoreboard-player-76561198000000003")
    expect(p3Row).toHaveTextContent("PlayerThree")
    expect(p3Row).toHaveTextContent("18") // kills
  })

  it("renders correct number of players per team", async () => {
    useViewerStore.getState().setDemoId("1")
    renderWithProviders(<Scoreboard visible={true} />)

    await screen.findByTestId("scoreboard-overlay")

    const ctSection = screen.getByTestId("scoreboard-team-ct")
    const tSection = screen.getByTestId("scoreboard-team-t")

    // 2 CT players + 1 header row
    const ctRows = ctSection.querySelectorAll("tbody tr")
    expect(ctRows).toHaveLength(2)

    // 2 T players + 1 header row
    const tRows = tSection.querySelectorAll("tbody tr")
    expect(tRows).toHaveLength(2)
  })

  it("selects a player when row is clicked", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    renderWithProviders(<Scoreboard visible={true} />)

    await screen.findByTestId("scoreboard-overlay")

    const row = screen.getByTestId("scoreboard-player-76561198000000001")
    await user.click(row)

    expect(useViewerStore.getState().selectedPlayerSteamId).toBe(
      "76561198000000001",
    )
  })

  it("deselects player when clicking the same row again", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("76561198000000001")
    renderWithProviders(<Scoreboard visible={true} />)

    await screen.findByTestId("scoreboard-overlay")

    const row = screen.getByTestId("scoreboard-player-76561198000000001")
    await user.click(row)

    expect(useViewerStore.getState().selectedPlayerSteamId).toBeNull()
  })

  it("highlights the selected player row", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setSelectedPlayer("76561198000000001")
    renderWithProviders(<Scoreboard visible={true} />)

    await screen.findByTestId("scoreboard-overlay")

    const row = screen.getByTestId("scoreboard-player-76561198000000001")
    expect(row.className).toContain("bg-white/20")
  })
})
