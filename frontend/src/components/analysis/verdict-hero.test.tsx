import { vi, describe, it, expect, beforeEach, afterEach } from "vitest"
import { screen, waitFor, cleanup } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, resetAppBindings } from "@/test/mocks/bindings"
import { useViewerStore } from "@/stores/viewer"
import { useAnalysisStore } from "@/stores/analysis"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

import { VerdictHero } from "./verdict-hero"

const baseAnalysis = {
  steam_id: "STEAM_A",
  overall_score: 72,
  version: 5,
  trade_pct: 0.78,
  avg_trade_ticks: 90,
  crosshair_height_avg_off: 0,
  time_to_fire_ms_avg: 240,
  flick_count: 0,
  flick_hit_pct: 0,
  first_shot_acc_pct: 0.54,
  spray_decay_slope: 0,
  standing_shot_pct: 0.62,
  counter_strafe_pct: 0,
  smokes_thrown: 0,
  smokes_kill_assist: 1,
  flash_assists: 1,
  he_damage: 0,
  nades_unused: 0,
  isolated_peek_deaths: 0,
  repeated_death_zones: 0,
  full_buy_adr: 0,
  eco_kills: 0,
  extras: { aim_pct: 0.74, standing_shot_pct: 0.62 },
}

describe("VerdictHero", () => {
  beforeEach(() => {
    resetAppBindings()
    useViewerStore.getState().reset()
    useAnalysisStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders the loading state while the analysis query is pending", () => {
    // A never-resolving promise keeps the query in the loading state.
    mockAppBindings.GetPlayerAnalysis.mockReturnValue(new Promise(() => {}))
    useViewerStore.setState({
      demoId: "1",
      selectedPlayerSteamId: "STEAM_A",
    })

    renderWithProviders(<VerdictHero />)

    expect(screen.getByTestId("verdict-hero-loading")).toBeInTheDocument()
    expect(screen.queryByTestId("verdict-hero")).not.toBeInTheDocument()
  })

  it("renders the empty state when the analysis row has no steam_id", async () => {
    mockAppBindings.GetPlayerAnalysis.mockResolvedValue({
      ...baseAnalysis,
      steam_id: "",
      overall_score: 0,
    })
    useViewerStore.setState({
      demoId: "1",
      selectedPlayerSteamId: "STEAM_A",
    })

    renderWithProviders(<VerdictHero />)

    await waitFor(() => {
      expect(screen.getByTestId("verdict-hero-empty")).toBeInTheDocument()
    })
    expect(screen.queryByTestId("verdict-hero")).not.toBeInTheDocument()
  })

  it("renders score, tier, and verdict when analysis is loaded", async () => {
    mockAppBindings.GetPlayerAnalysis.mockResolvedValue(baseAnalysis)
    useViewerStore.setState({
      demoId: "1",
      selectedPlayerSteamId: "STEAM_A",
    })

    renderWithProviders(<VerdictHero />)

    await waitFor(() => {
      expect(screen.getByTestId("verdict-hero")).toBeInTheDocument()
    })

    // Score = 72 → tier B ("Solid"). Verdict mentions trade %.
    expect(screen.getByTestId("verdict-hero-score")).toHaveTextContent("72")
    expect(screen.getByTestId("verdict-hero-tier")).toHaveTextContent("B")
    expect(screen.getByTestId("verdict-hero-verdict")).toHaveTextContent(
      /78% trade rate/i,
    )
  })

  it("does not render any category bars (right column removed)", async () => {
    mockAppBindings.GetPlayerAnalysis.mockResolvedValue(baseAnalysis)
    useViewerStore.setState({
      demoId: "1",
      selectedPlayerSteamId: "STEAM_A",
    })

    const { container } = renderWithProviders(<VerdictHero />)

    await waitFor(() => {
      expect(screen.getByTestId("verdict-hero")).toBeInTheDocument()
    })

    // No more A2 category bars — those move to HabitChecklist (P1-1).
    expect(
      container.querySelector('[data-testid^="verdict-hero-cat-"]'),
    ).toBeNull()
    expect(screen.queryByTestId("verdict-hero-cat-trade")).toBeNull()
    expect(screen.queryByTestId("verdict-hero-cat-aim")).toBeNull()
    expect(screen.queryByTestId("verdict-hero-cat-movement")).toBeNull()
    expect(screen.queryByTestId("verdict-hero-cat-utility")).toBeNull()
  })
})
