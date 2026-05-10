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
