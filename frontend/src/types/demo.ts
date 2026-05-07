export type DemoStatus = "imported" | "parsing" | "ready" | "failed"

export interface Demo {
  id: number
  map_name: string
  file_path: string
  file_size: number
  status: DemoStatus
  total_ticks: number
  tick_rate: number
  duration_secs: number
  match_date: string
  created_at: string
}

export interface DemoListResponse {
  data: Demo[]
  meta: { total: number; page: number; per_page: number }
}

export interface TickData {
  tick: number
  steam_id: string
  x: number
  y: number
  z: number
  yaw: number
  health: number
  armor: number
  is_alive: boolean
  weapon: string | null
  money: number
  has_helmet: boolean
  has_defuser: boolean
  inventory: string[]
}

export interface TickDataResponse {
  data: TickData[]
}

export type GameEventType =
  | "kill"
  | "weapon_fire"
  | "player_hurt"
  | "grenade_throw"
  | "grenade_detonate"
  | "smoke_start"
  | "smoke_expired"
  | "decoy_start"
  | "bomb_plant"
  | "bomb_defuse"
  | "bomb_explode"

export interface GameEvent {
  id: string
  demo_id: string
  round_id: string | null
  tick: number
  event_type: GameEventType
  attacker_steam_id: string | null
  victim_steam_id: string | null
  weapon: string | null
  x: number | null
  y: number | null
  z: number | null
  extra_data: Record<string, unknown> | null
}

export interface GameEventsResponse {
  data: GameEvent[]
}
