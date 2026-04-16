export interface HeatmapPoint {
  x: number
  y: number
  kill_count: number
}

export interface PlayerInfo {
  steam_id: string
  player_name: string
}

export interface WeaponStat {
  weapon: string
  kill_count: number
  hs_count: number
}
