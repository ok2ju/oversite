import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, waitFor, cleanup } from "@testing-library/react"
import { useViewerStore } from "@/stores/viewer"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, resetAppBindings } from "@/test/mocks/bindings"
import { mockRounds } from "@/test/fixtures"
import { RoundTradeBars } from "./round-trade-bars"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

describe("RoundTradeBars", () => {
  beforeEach(() => {
    resetAppBindings()
    useViewerStore.getState().reset()
    useViewerStore.getState().initDemo({
      id: "1",
      mapName: "de_dust2",
      totalTicks: 100000,
      tickRate: 64,
    })
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetDemoRounds.mockResolvedValue(mockRounds)
  })

  afterEach(() => {
    cleanup()
  })

  it("renders the empty state when no rounds are available", async () => {
    mockAppBindings.GetDemoRounds.mockResolvedValueOnce([])
    mockAppBindings.GetPlayerRoundAnalysis.mockResolvedValueOnce([])

    renderWithProviders(<RoundTradeBars />)

    expect(
      await screen.findByTestId("round-trade-bars-empty"),
    ).toBeInTheDocument()
  })

  it("renders one bar per round even when analysis rows are absent", async () => {
    // Empty analysis rows — every round should still render as a flat bar so
    // the chart still shows the full match cadence.
    mockAppBindings.GetPlayerRoundAnalysis.mockResolvedValueOnce([])

    renderWithProviders(<RoundTradeBars />)

    await waitFor(() => {
      expect(screen.getByTestId("round-trade-bar-1")).toBeInTheDocument()
    })

    for (const round of mockRounds) {
      const bar = screen.getByTestId(`round-trade-bar-${round.round_number}`)
      expect(bar).toBeInTheDocument()
      expect(bar).toHaveAttribute("data-trade-pct", "0")
    }
  })

  it("bar height matches trade_pct via the data-trade-pct attr", async () => {
    mockAppBindings.GetPlayerRoundAnalysis.mockResolvedValueOnce([
      { steam_id: "STEAM_A", round_number: 1, trade_pct: 0.5, extras: null },
      { steam_id: "STEAM_A", round_number: 3, trade_pct: 1, extras: null },
    ])

    renderWithProviders(<RoundTradeBars />)

    await waitFor(() => {
      expect(screen.getByTestId("round-trade-bar-1")).toHaveAttribute(
        "data-trade-pct",
        "0.5",
      )
    })
    expect(screen.getByTestId("round-trade-bar-2")).toHaveAttribute(
      "data-trade-pct",
      "0",
    )
    expect(screen.getByTestId("round-trade-bar-3")).toHaveAttribute(
      "data-trade-pct",
      "1",
    )

    // Heights map proportionally onto 0–100% so the visual cadence matches.
    expect(screen.getByTestId("round-trade-bar-1")).toHaveStyle({
      height: "50%",
    })
    expect(screen.getByTestId("round-trade-bar-3")).toHaveStyle({
      height: "100%",
    })
  })

  it("renders bars in round_number ASC order", async () => {
    // Rows out of order on purpose; bars must still come out in match order.
    mockAppBindings.GetPlayerRoundAnalysis.mockResolvedValueOnce([
      { steam_id: "STEAM_A", round_number: 3, trade_pct: 0.3, extras: null },
      { steam_id: "STEAM_A", round_number: 1, trade_pct: 0.1, extras: null },
      { steam_id: "STEAM_A", round_number: 2, trade_pct: 0.2, extras: null },
    ])

    renderWithProviders(<RoundTradeBars />)

    await waitFor(() => {
      expect(screen.getByTestId("round-trade-bar-1")).toBeInTheDocument()
    })

    const container = screen.getByTestId("round-trade-bars")
    const bars = Array.from(
      container.querySelectorAll("[data-testid^='round-trade-bar-']"),
    )
    const renderedRounds = bars.map((b) =>
      Number(b.getAttribute("data-testid")?.replace("round-trade-bar-", "")),
    )
    expect(renderedRounds).toEqual([1, 2, 3])
  })
})
