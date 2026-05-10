import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useViewerStore } from "@/stores/viewer"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, resetAppBindings } from "@/test/mocks/bindings"
import { CategoryCard } from "./category-card"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

const TICK_RATE = 64

describe("CategoryCard", () => {
  beforeEach(() => {
    resetAppBindings()
    useViewerStore.getState().reset()
    useViewerStore.getState().initDemo({
      id: "1",
      mapName: "de_dust2",
      totalTicks: 100000,
      tickRate: TICK_RATE,
    })
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
  })

  afterEach(() => {
    cleanup()
  })

  it("renders the trade metrics formatted as XX% and Xs", async () => {
    // 0.62 trade_pct → 62%; 90 ticks at tick rate 64 → 1.4s.
    mockAppBindings.GetPlayerAnalysis.mockResolvedValueOnce({
      steam_id: "STEAM_A",
      overall_score: 62,
      trade_pct: 0.62,
      avg_trade_ticks: 90,
      extras: null,
    })

    renderWithProviders(<CategoryCard category="trade" />)

    // Card itself is always rendered (header is the toggle); assert metrics
    // appear once it expands. Default state is closed → click to open.
    const header = await screen.findByTestId("category-card-header-trade")
    await userEvent.setup().click(header)

    expect(
      screen.getByTestId("category-card-trade-pct-trade"),
    ).toHaveTextContent("62%")
    expect(
      screen.getByTestId("category-card-avg-trade-trade"),
    ).toHaveTextContent("1.4s")
  })

  it("opens and closes when the header is clicked", async () => {
    const user = userEvent.setup()
    mockAppBindings.GetPlayerAnalysis.mockResolvedValueOnce({
      steam_id: "STEAM_A",
      overall_score: 50,
      trade_pct: 0.5,
      avg_trade_ticks: 32,
      extras: null,
    })

    renderWithProviders(<CategoryCard category="trade" />)

    const card = await screen.findByTestId("category-card-trade")
    expect(card).toHaveAttribute("data-state", "closed")
    expect(
      screen.queryByTestId("category-card-body-trade"),
    ).not.toBeInTheDocument()

    await user.click(screen.getByTestId("category-card-header-trade"))
    expect(card).toHaveAttribute("data-state", "open")
    expect(screen.getByTestId("category-card-body-trade")).toBeInTheDocument()

    await user.click(screen.getByTestId("category-card-header-trade"))
    expect(card).toHaveAttribute("data-state", "closed")
    expect(
      screen.queryByTestId("category-card-body-trade"),
    ).not.toBeInTheDocument()
  })

  it("shows the suggestion line only when open", async () => {
    const user = userEvent.setup()
    mockAppBindings.GetPlayerAnalysis.mockResolvedValueOnce({
      steam_id: "STEAM_A",
      overall_score: 50,
      trade_pct: 0.5,
      avg_trade_ticks: 32,
      extras: null,
    })

    renderWithProviders(<CategoryCard category="trade" />)

    const header = await screen.findByTestId("category-card-header-trade")
    expect(
      screen.queryByTestId("category-card-suggestion-trade"),
    ).not.toBeInTheDocument()

    await user.click(header)

    expect(
      screen.getByTestId("category-card-suggestion-trade"),
    ).toHaveTextContent(/Trade your teammates/i)
  })
})
