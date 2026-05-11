// MistakeEntry mirrors main.MistakeEntry. Slice 10 promoted category /
// severity / title / suggestion to first-class fields filled server-side
// from analysis.TemplateForKind, so the frontend no longer carries a
// kind-keyed presentation map. Slice 12 (P2-3) adds why_it_hurts — the
// 1-sentence damage caption rendered as the mistake-detail subtitle.
// Unknown kinds still flow through with empty title / suggestion — render
// the kind string as a fallback.
export interface MistakeEntry {
  id: number
  kind: string
  category: string
  severity: number
  title: string
  suggestion: string
  why_it_hurts: string
  round_number: number
  tick: number
  steam_id: string
  extras: Record<string, unknown> | null
  // duel_id is the analysis_duels row this mistake attaches to. Slice 13
  // adds duel-scoped attribution: fire- and kill-anchored mistakes carry
  // a duel_id; cross-duel patterns (eco_misbuy, he_damage) carry null.
  duel_id: number | null
}

// MistakeCoOccurrence mirrors main.MistakeCoOccurrence — the lightweight
// reference for one of the player's *other* mistakes inside the same fire
// window. The detail panel renders one chip per entry so the user can pivot
// to the related play without leaving the card.
export interface MistakeCoOccurrence {
  id: number
  kind: string
  title: string
  tick: number
}

// MistakeContext is the deep-detail variant returned by GetMistakeContext.
// Carries the surrounding round window plus co-occurring siblings so the
// analysis-detail card can render the play with no extra round-trip.
export interface MistakeContext {
  entry: MistakeEntry
  round_start_tick: number
  round_end_tick: number
  freeze_end_tick: number
  co_occurring: MistakeCoOccurrence[] | null
}
