// DuelEntry mirrors main.DuelEntry — a directed attacker→victim
// engagement reconstructed from the merged weapon_fire + player_hurt +
// kill stream. The viewer renders one band per entry on the duels lane
// of RoundTimeline; mistakes inside the duel hang off duel_id on each
// MistakeEntry row.
export interface DuelEntry {
  id: number
  round_number: number
  attacker_steam: string
  victim_steam: string
  start_tick: number
  end_tick: number
  outcome: DuelOutcome
  end_reason: DuelEndReason
  hit_confirmed: boolean
  hurt_count: number
  shot_count: number
  // Peer V→A duel id when both players fired at each other in
  // overlapping windows. null otherwise.
  mutual_duel_id: number | null
}

export type DuelOutcome =
  | "won"
  | "lost"
  | "inconclusive"
  | "won_then_traded"
  | "lost_but_traded"

export type DuelEndReason =
  | "kill"
  | "trade"
  | "gap"
  | "cone_switch"
  | "round_end"
  | "clean_kill"

// DuelContext mirrors main.DuelContext — deep-detail variant returned by
// GetDuelContext. Powers the tooltip / popover content on the duels lane.
export interface DuelContext {
  duel: DuelEntry
  mistakes: import("./mistake").MistakeEntry[]
}
