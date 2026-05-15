export type DemoStatus = "imported" | "parsing" | "ready" | "failed"

// Demo is the full detail-row variant returned by GetDemoByID and used by the
// viewer page. Includes the absolute file_path because the backend may need
// it on detail-only paths (e.g. logs, future "reveal in Finder" actions).
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

// DemoSummary is the lighter list-row variant returned by ListDemos. The full
// path is replaced with the basename (file_name) — the table never used
// anything but the basename, and a 100-row page saves ~10–20 KB on the wire.
// ct_score / t_score are the final per-side totals from the rounds table;
// both zero means the demo hasn't been parsed yet.
export interface DemoSummary {
  id: number
  map_name: string
  file_name: string
  file_size: number
  status: DemoStatus
  total_ticks: number
  tick_rate: number
  duration_secs: number
  match_date: string
  created_at: string
  ct_score: number
  t_score: number
}

export interface DemoListResponse {
  data: DemoSummary[]
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
  ammo_clip: number
  ammo_reserve: number
  // Per-tick inventory (migration 023). Empty array on pre-023 demos —
  // team-bars falls back to the round-scoped freeze-end loadout in that case.
  inventory: string[]
}

// One player's freeze-end loadout for a single round. Inventory is a parsed
// array of weapon/equipment names; the wire format is a comma-separated string
// (see use-round-loadouts.ts for the split). Migration 011 moved this out of
// per-tick storage so the team bars don't have to re-fetch inventory at
// 64 Hz.
export interface RoundLoadoutEntry {
  steam_id: string
  inventory: string[]
}

export interface TickDataResponse {
  data: TickData[]
}

export type GameEventType =
  | "kill"
  | "weapon_fire"
  | "player_hurt"
  | "player_flashed"
  | "grenade_throw"
  | "grenade_bounce"
  | "grenade_detonate"
  | "smoke_start"
  | "smoke_expired"
  | "decoy_start"
  | "fire_start"
  | "bomb_plant"
  | "bomb_defuse"
  | "bomb_explode"

// Hot fields that used to live inside `extra_data` were promoted to top-level
// columns on game_events (migration 010). The frontend reads them directly
// instead of decoding the JSON blob — kill-log rendering is the dominant
// caller and was paying for a per-row `Record<string, unknown>` parse.
//
// Cold fields (penetrated, flash_assist, no_scope, hit_x/hit_y, entity_id, …)
// still travel inside `extra_data` because they fan out across many event
// types and aren't worth a column-per-field schema explosion.
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
  headshot: boolean
  assister_steam_id: string | null
  health_damage: number
  attacker_name: string
  victim_name: string
  attacker_team: string
  victim_team: string
  extra_data: Record<string, unknown> | null
}

export interface GameEventsResponse {
  data: GameEvent[]
}
