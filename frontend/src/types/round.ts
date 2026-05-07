export interface Round {
  id: string
  round_number: number
  start_tick: number
  freeze_end_tick: number
  end_tick: number
  winner_side: string
  win_reason: string
  ct_score: number
  t_score: number
  is_overtime: boolean
  ct_team_name: string
  t_team_name: string
}

export interface RoundsResponse {
  data: Round[]
}
