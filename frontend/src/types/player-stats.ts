export interface DamageByWeapon {
  weapon: string
  damage: number
}

export interface DamageByOpponent {
  steam_id: string
  player_name: string
  team_side: string
  damage: number
}

export interface PlayerRoundDetail {
  round_number: number
  team_side: string
  kills: number
  deaths: number
  assists: number
  damage: number
  hs_kills: number
  clutch_kills: number
  first_kill: boolean
  first_death: boolean
  trade_kill: boolean
  loadout_value: number
  distance_units: number
  alive_duration_secs: number
  time_to_first_contact_sec: number | null
}

// MovementStats summarizes how the player moved through the match. Strafe
// percent is approximate (16 Hz sample rate); see the panel tooltip.
export interface MovementStats {
  distance_units: number
  avg_speed_ups: number
  max_speed_ups: number
  strafe_percent: number
  stationary_ratio: number
  walking_ratio: number
  running_ratio: number
}

// TimingStats summarizes round-shape behavior: average time before first
// contact, average alive duration, and a coarse time-on-bombsite proxy.
export interface TimingStats {
  avg_time_to_first_contact_secs: number
  avg_alive_duration_secs: number
  time_on_site_a_secs: number
  time_on_site_b_secs: number
}

// UtilityStats summarizes grenade throws, flash assists, and total blind
// time inflicted across a match.
export interface UtilityStats {
  flashes_thrown: number
  smokes_thrown: number
  hes_thrown: number
  molotovs_thrown: number
  decoys_thrown: number
  flash_assists: number
  blind_time_inflicted_secs: number
  enemies_flashed: number
}

// HitGroupBreakdown is one row in the damage-by-hit-group breakdown rendered
// under the Detail tab.
export interface HitGroupBreakdown {
  hit_group: number
  label: string
  damage: number
  hits: number
}

export interface PlayerMatchStats {
  steam_id: string
  player_name: string
  team_side: string
  rounds_played: number
  kills: number
  deaths: number
  assists: number
  damage: number
  hs_kills: number
  clutch_kills: number
  first_kills: number
  first_deaths: number
  opening_wins: number
  opening_losses: number
  trade_kills: number
  hs_percent: number
  adr: number
  damage_by_weapon: DamageByWeapon[]
  damage_by_opponent: DamageByOpponent[]
  rounds: PlayerRoundDetail[]
  movement: MovementStats
  timings: TimingStats
  utility: UtilityStats
  hit_groups: HitGroupBreakdown[]
}
