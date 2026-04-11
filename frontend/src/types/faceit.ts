export interface FaceitProfile {
  nickname: string
  avatar_url: string | null
  elo: number | null
  level: number | null
  country: string | null
  matches_played: number
  current_streak: { type: "win" | "loss" | "draw" | "none"; count: number }
}

export interface FaceitProfileResponse {
  data: FaceitProfile
}

export interface EloHistoryPoint {
  elo: number | null
  map_name: string
  played_at: string
}

export interface EloHistoryResponse {
  data: EloHistoryPoint[]
}
