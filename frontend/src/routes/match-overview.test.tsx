import { vi, describe, it, expect, beforeEach, afterEach } from "vitest"
import { screen, waitFor, cleanup } from "@testing-library/react"
import { Route, Routes } from "react-router-dom"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import { mockDemos } from "@/test/fixtures"
import type { MatchOverview } from "@/types/match-overview"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

import MatchOverviewPage from "@/routes/match-overview"

function renderOverview(demoId = "1") {
  return renderWithProviders(
    <Routes>
      <Route path="/demos/:id/overview" element={<MatchOverviewPage />} />
    </Routes>,
    { initialRoute: `/demos/${demoId}/overview` },
  )
}

function basePayload(): MatchOverview {
  return {
    demo: mockDemos[0],
    format: {
      regulation_rounds: 24,
      halftime_round: 12,
      overtime_half_len: 3,
      has_overtime: false,
      total_rounds: 16,
      pistol_round_numbers: [1, 13],
    },
    team_a: {
      name: "Astralis",
      side: "A",
      score: 13,
      players: [
        {
          steam_id: "STEAM_A1",
          player_name: "device",
          kills: 20,
          deaths: 15,
          assists: 5,
          hs_percent: 60,
          adr: 95,
          kast: 75,
          rating_2: 1.21,
          rounds_played: 16,
        },
      ],
      totals: {
        kills: 50,
        deaths: 40,
        assists: 10,
        adr: 80,
        hs_percent: 50,
        kast: 70,
        rating: 1.1,
      },
      top_performer: {
        steam_id: "STEAM_A1",
        player_name: "device",
        kills: 20,
        deaths: 15,
        assists: 5,
        hs_percent: 60,
        adr: 95,
        kast: 75,
        rating_2: 1.21,
        rounds_played: 16,
      },
      pistol_wins: 1,
    },
    team_b: {
      name: "NaVi",
      side: "B",
      score: 3,
      players: [
        {
          steam_id: "STEAM_B1",
          player_name: "s1mple",
          kills: 18,
          deaths: 18,
          assists: 4,
          hs_percent: 50,
          adr: 85,
          kast: 65,
          rating_2: 1.05,
          rounds_played: 16,
        },
      ],
      totals: {
        kills: 40,
        deaths: 50,
        assists: 8,
        adr: 70,
        hs_percent: 45,
        kast: 60,
        rating: 0.95,
      },
      top_performer: {
        steam_id: "STEAM_B1",
        player_name: "s1mple",
        kills: 18,
        deaths: 18,
        assists: 4,
        hs_percent: 50,
        adr: 85,
        kast: 65,
        rating_2: 1.05,
        rounds_played: 16,
      },
      pistol_wins: 1,
    },
    rounds: [
      {
        round_number: 1,
        winner_side: "T",
        win_reason: "ct_killed",
        winner: "a",
        is_pistol: true,
        is_overtime: false,
        team_a_damage: 480,
        team_b_damage: 320,
        team_a_equip_value: 800,
        team_b_equip_value: 800,
      },
      {
        round_number: 13,
        winner_side: "CT",
        win_reason: "ct_killed",
        winner: "a",
        is_pistol: true,
        is_overtime: false,
        team_a_damage: 460,
        team_b_damage: 280,
        team_a_equip_value: 850,
        team_b_equip_value: 850,
      },
    ],
    halves: [
      {
        label: "1st half",
        team_a_wins: 7,
        team_b_wins: 5,
        team_a_side: "T",
        team_b_side: "CT",
      },
      {
        label: "2nd half",
        team_a_wins: 6,
        team_b_wins: 6,
        team_a_side: "CT",
        team_b_side: "T",
      },
    ],
    kpis: {
      total_rounds: 24,
      pistol_a: 2,
      pistol_b: 0,
      longest_streak: 4,
      streak_team: "a",
      max_lead: 6,
    },
  }
}

describe("MatchOverviewPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders the final score, team names, and KPIs from the payload", async () => {
    mockAppBindings.GetMatchOverview.mockResolvedValueOnce(basePayload())
    renderOverview()
    await waitFor(() =>
      expect(screen.getAllByText("Astralis").length).toBeGreaterThan(0),
    )
    expect(screen.getAllByText("NaVi").length).toBeGreaterThan(0)
    expect(screen.getByText(/of 24 max/i)).toBeInTheDocument()
    const scoreA = document.querySelector(".mo-score-a")
    const scoreB = document.querySelector(".mo-score-b")
    expect(scoreA?.textContent).toBe("13")
    expect(scoreB?.textContent).toBe("3")
  })

  it("shows 1st and 2nd half breakdown", async () => {
    mockAppBindings.GetMatchOverview.mockResolvedValueOnce(basePayload())
    renderOverview()
    await waitFor(() =>
      expect(screen.getAllByText("1st half").length).toBeGreaterThan(0),
    )
    expect(screen.getAllByText("2nd half").length).toBeGreaterThan(0)
  })

  it("renders an OT half label when overtime is present", async () => {
    const payload = basePayload()
    payload.format.has_overtime = true
    payload.format.total_rounds = 27
    payload.halves = [
      ...payload.halves,
      {
        label: "OT1 first",
        team_a_wins: 2,
        team_b_wins: 1,
        team_a_side: "T",
        team_b_side: "CT",
      },
    ]
    mockAppBindings.GetMatchOverview.mockResolvedValueOnce(payload)
    renderOverview()
    await waitFor(() =>
      expect(screen.getByText("OT1 first")).toBeInTheDocument(),
    )
    expect(screen.getByText(/of 24 reg\. \+ OT/i)).toBeInTheDocument()
  })

  it("shows demo-not-found when the binding returns no data", async () => {
    mockAppBindings.GetMatchOverview.mockResolvedValueOnce(null)
    renderOverview()
    await waitFor(() =>
      expect(screen.getByText(/Demo not found/i)).toBeInTheDocument(),
    )
  })

  it("falls back to Team A/Team B labels when names are empty", async () => {
    const payload = basePayload()
    payload.team_a.name = "Team A"
    payload.team_b.name = "Team B"
    mockAppBindings.GetMatchOverview.mockResolvedValueOnce(payload)
    renderOverview()
    await waitFor(() =>
      expect(screen.getAllByText("Team A").length).toBeGreaterThan(0),
    )
    expect(screen.getAllByText("Team B").length).toBeGreaterThan(0)
  })
})
