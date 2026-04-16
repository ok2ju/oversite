export interface ScoreboardEntry {
  steam_id: string
  player_name: string
  team_side: "CT" | "T"
  kills: number
  deaths: number
  assists: number
  damage: number
  hs_kills: number
  rounds_played: number
  hs_percent: number
  adr: number
}
