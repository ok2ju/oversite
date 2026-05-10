// PlayerAnalysis mirrors main.PlayerAnalysis. Slice 10 promoted the per-
// category aggregate columns to top-level fields; the legacy `extras` blob
// keeps slice-8 fields (aim_pct, standing_shot_pct, …) for backward compat.
export interface PlayerAnalysis {
  steam_id: string
  overall_score: number
  version: number
  trade_pct: number
  avg_trade_ticks: number

  // Aim
  crosshair_height_avg_off: number
  time_to_fire_ms_avg: number
  flick_count: number
  flick_hit_pct: number

  // Spray
  first_shot_acc_pct: number
  spray_decay_slope: number

  // Movement
  standing_shot_pct: number
  counter_strafe_pct: number

  // Utility
  smokes_thrown: number
  smokes_kill_assist: number
  flash_assists: number
  he_damage: number
  nades_unused: number

  // Positioning
  isolated_peek_deaths: number
  repeated_death_zones: number

  // Economy
  full_buy_adr: number
  eco_kills: number

  extras: Record<string, unknown> | null
}

// Mirrors main.PlayerRoundEntry — one row from the player_round_analysis
// table, returned in round_number ASC order from GetPlayerRoundAnalysis.
export interface PlayerRoundEntry {
  steam_id: string
  round_number: number
  trade_pct: number
  buy_type: string
  money_spent: number
  nades_used: number
  nades_unused: number
  shots_fired: number
  shots_hit: number
  extras: Record<string, unknown> | null
}

// Mirrors main.AnalysisStatus in types.go. status is one of:
// "imported" | "parsing" | "failed" | "missing" | "ready". The viewer panel
// auto-triggers a recompute when status is "missing".
export interface AnalysisStatus {
  demo_id: string
  status: AnalysisStatusValue
}

export type AnalysisStatusValue =
  | "imported"
  | "parsing"
  | "failed"
  | "missing"
  | "ready"

// Mirrors main.MatchInsights — the team-level summary for the analysis page.
export interface MatchInsights {
  demo_id: string
  ct_summary: TeamSummary
  t_summary: TeamSummary
  standouts: PlayerHighlight[]
}

export interface TeamSummary {
  side: "CT" | "T"
  players: number
  avg_overall_score: number
  avg_trade_pct: number
  avg_standing_shot_pct: number
  avg_counter_strafe_pct: number
  avg_first_shot_acc_pct: number
  total_flash_assists: number
  total_smokes_kill_assist: number
  total_he_damage: number
  total_isolated_peek_deaths: number
  total_eco_kills: number
  avg_full_buy_adr: number
}

export interface PlayerHighlight {
  steam_id: string
  category: string
  metric_name: string
  metric_value: number
}

// Habit checklist surface — see plans/analysis-overhaul.md §4.1 / §6.1.
// Status / direction live as discriminated-union strings so the frontend can
// switch on them without reinventing the threshold table; thresholds ride
// alongside so each row can render its own norm line ("≤ 100 ms").
export type HabitStatus = "good" | "warn" | "bad"
export type HabitDirection = "lower" | "higher" | "balanced"

export type HabitKey =
  | "counter_strafe"
  | "reaction"
  | "first_shot_acc"
  | "shooting_in_motion"
  | "crouch_before_shot"
  | "flick_balance"
  | "trade_timing"
  | "untraded_deaths"
  | "utility_used"
  | "isolated_peek_deaths"
  | "repeated_death_zone"

export interface HabitRow {
  key: HabitKey
  label: string
  description: string
  unit: string
  direction: HabitDirection
  value: number
  status: HabitStatus
  good_threshold: number
  warn_threshold: number
  good_min: number
  good_max: number
  warn_min: number
  warn_max: number
  // P0-3 — null when there's no previous demo to compare against.
  previous_value: number | null
  delta: number | null
}

export interface HabitReport {
  demo_id: string
  steam_id: string
  as_of: string
  habits: HabitRow[]
}
