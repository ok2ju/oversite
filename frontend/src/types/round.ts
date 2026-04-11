export interface Round {
  id: string
  round_number: number
  start_tick: number
  end_tick: number
  winner_side: string
  ct_score: number
  t_score: number
}

export interface RoundsResponse {
  data: Round[]
}
