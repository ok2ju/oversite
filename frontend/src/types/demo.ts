export interface Demo {
  id: string
  map_name: string | null
  file_size: number
  status: "uploaded" | "parsing" | "ready" | "failed"
  total_ticks: number | null
  tick_rate: number | null
  duration_secs: number | null
  match_date: string | null
  created_at: string
}

export interface DemoListResponse {
  data: Demo[]
  meta: { total: number; page: number; per_page: number }
}

export interface DemoResponse {
  data: Demo
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
}

export interface TickDataResponse {
  data: TickData[]
}

export type GameEventType =
  | "kill"
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
