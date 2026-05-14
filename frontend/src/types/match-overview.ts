import type { Demo } from "@/types/demo"

export interface MatchFormat {
  regulation_rounds: number
  halftime_round: number
  overtime_half_len: number
  has_overtime: boolean
  total_rounds: number
  pistol_round_numbers: number[]
}

export interface PlayerOverview {
  steam_id: string
  player_name: string
  kills: number
  deaths: number
  assists: number
  hs_percent: number
  adr: number
  kast: number
  rating_2: number
  rounds_played: number
}

export interface TeamTotals {
  kills: number
  deaths: number
  assists: number
  adr: number
  hs_percent: number
  kast: number
  rating: number
}

export interface TeamOverview {
  name: string
  side: "A" | "B"
  score: number
  players: PlayerOverview[]
  totals: TeamTotals
  top_performer: PlayerOverview | null
  pistol_wins: number
}

export interface RoundOverview {
  round_number: number
  winner_side: "T" | "CT" | string
  win_reason: string
  winner: "a" | "b"
  is_pistol: boolean
  is_overtime: boolean
  team_a_damage: number
  team_b_damage: number
  team_a_equip_value: number
  team_b_equip_value: number
}

export interface HalfOverview {
  label: string
  team_a_wins: number
  team_b_wins: number
  team_a_side: "T" | "CT" | string
  team_b_side: "T" | "CT" | string
}

export interface MatchKPIs {
  total_rounds: number
  pistol_a: number
  pistol_b: number
  longest_streak: number
  streak_team: "a" | "b" | ""
  max_lead: number
}

export interface MatchOverview {
  demo: Demo
  format: MatchFormat
  team_a: TeamOverview
  team_b: TeamOverview
  rounds: RoundOverview[]
  halves: HalfOverview[]
  kpis: MatchKPIs
}
