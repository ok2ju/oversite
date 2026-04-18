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

export interface FaceitMatch {
  id: string
  faceit_match_id: string
  map_name: string
  score_team: number
  score_opponent: number
  result: "W" | "L"
  kills: number | null
  deaths: number | null
  assists: number | null
  demo_url: string | null
  demo_id: string | null
  has_demo: boolean
  played_at: string
}

export interface FaceitMatchListResponse {
  data: FaceitMatch[]
  meta: { total: number; page: number; per_page: number }
}
