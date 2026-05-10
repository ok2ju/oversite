// PlayerAnalysis mirrors main.PlayerAnalysis. Slice 8 rides aim_pct,
// standing_shot_pct, engagements, and avg_fire_speed in the extras blob (no
// schema migration); consumers cast at the call site (the analysis surface
// is read-only and small enough that a discriminated union isn't worth it).
export interface PlayerAnalysis {
  steam_id: string
  overall_score: number
  trade_pct: number
  avg_trade_ticks: number
  extras: Record<string, unknown> | null
}

// Mirrors main.PlayerRoundEntry — one row from the player_round_analysis
// table, returned in round_number ASC order from GetPlayerRoundAnalysis.
export interface PlayerRoundEntry {
  steam_id: string
  round_number: number
  trade_pct: number
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
