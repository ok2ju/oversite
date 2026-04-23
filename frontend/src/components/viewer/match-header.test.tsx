import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup, act, waitFor } from "@testing-library/react"
import { useViewerStore } from "@/stores/viewer"
import { mockRounds, mockScoreboardEntries } from "@/test/fixtures/demos"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings } from "@/test/mocks/bindings"
import type { PlayerRosterEntry } from "@/types/roster"
import { MatchHeader } from "./match-header"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

const initialRoster: PlayerRosterEntry[] = [
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

const swappedRoster: PlayerRosterEntry[] = initialRoster.map((e) => ({
  ...e,
  team_side: e.team_side === "CT" ? "T" : "CT",
}))

describe("MatchHeader", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
    mockAppBindings.GetDemoRounds.mockResolvedValue(mockRounds)
    mockAppBindings.GetScoreboard.mockResolvedValue(mockScoreboardEntries)
    mockAppBindings.GetRoundRoster.mockImplementation(
      async () => initialRoster as unknown as never[],
    )
  })

  afterEach(() => {
    cleanup()
  })

  it("does not render when demoId is null", () => {
    renderWithProviders(<MatchHeader />)
    expect(screen.queryByTestId("match-header")).not.toBeInTheDocument()
  })

  it("renders team labels derived from the active round's roster", async () => {
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<MatchHeader />)

    expect(await screen.findByTestId("match-header")).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByTestId("match-header-team-ct")).toHaveTextContent(
        "team_PlayerOne",
      )
    })
    expect(screen.getByTestId("match-header-team-t")).toHaveTextContent(
      "team_PlayerThree",
    )
  })

  it("swaps team labels when sides have switched in the active round", async () => {
    mockAppBindings.GetRoundRoster.mockImplementation(
      async () => swappedRoster as unknown as never[],
    )
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setTick(4000) // inside round 2 (stand-in for post-swap round)
    renderWithProviders(<MatchHeader />)

    expect(await screen.findByTestId("match-header")).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByTestId("match-header-team-ct")).toHaveTextContent(
        "team_PlayerThree",
      )
    })
    expect(screen.getByTestId("match-header-team-t")).toHaveTextContent(
      "team_PlayerOne",
    )
  })

  it("shows 0-0 during round 1", async () => {
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setTick(1000) // round 1
    renderWithProviders(<MatchHeader />)

    await waitFor(() => {
      expect(screen.getByTestId("match-header-ct-score")).toHaveTextContent("0")
    })
    expect(screen.getByTestId("match-header-t-score")).toHaveTextContent("0")
  })

  it("shows previous round score during the active round", async () => {
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setTick(4000) // inside round 2
    renderWithProviders(<MatchHeader />)

    // Round 2 is active; score going in is round 1's end: CT=1, T=0
    await waitFor(() => {
      expect(screen.getByTestId("match-header-ct-score")).toHaveTextContent("1")
    })
    expect(screen.getByTestId("match-header-t-score")).toHaveTextContent("0")
  })

  it("shows round countdown 1:25 when 45s into the active round (15s freeze + 30s of round elapsed)", async () => {
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setTick(3200 + 64 * 45) // 45s into round 2
    renderWithProviders(<MatchHeader />)

    await waitFor(() => {
      expect(screen.getByTestId("match-header-round-time")).toHaveTextContent(
        "1:25",
      )
    })
  })

  it("starts each round with the freeze countdown at 0:15 and counts down as ticks advance", async () => {
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setTick(0)
    renderWithProviders(<MatchHeader />)

    await waitFor(() => {
      expect(screen.getByTestId("match-header-round-time")).toHaveTextContent(
        "0:15",
      )
    })

    act(() => useViewerStore.getState().setTick(64 * 10))
    expect(screen.getByTestId("match-header-round-time")).toHaveTextContent(
      "0:05",
    )
  })
})
