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
