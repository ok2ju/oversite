export type TeamSide = "CT" | "T"

export interface PlayerRosterEntry {
  steam_id: string
  player_name: string
  team_side: TeamSide
}

export interface PlayerRosterResponse {
  data: PlayerRosterEntry[]
}
