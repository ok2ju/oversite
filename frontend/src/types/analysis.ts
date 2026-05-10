export interface PlayerAnalysis {
  steam_id: string
  overall_score: number
  trade_pct: number
  avg_trade_ticks: number
  extras: Record<string, unknown> | null
}
