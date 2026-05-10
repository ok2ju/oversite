// MistakeEntry mirrors main.MistakeEntry. Slice 10 promoted category /
// severity / title / suggestion to first-class fields filled server-side
// from analysis.TemplateForKind, so the frontend no longer carries a
// kind-keyed presentation map. Unknown kinds still flow through with empty
// title / suggestion — render the kind string as a fallback.
export interface MistakeEntry {
  id: number
  kind: string
  category: string
  severity: number
  title: string
  suggestion: string
  round_number: number
  tick: number
  steam_id: string
  extras: Record<string, unknown> | null
}

// MistakeContext is the deep-detail variant returned by GetMistakeContext.
// Carries the surrounding round window so the analysis-detail card can render
// the play without a second round-trip.
export interface MistakeContext {
  entry: MistakeEntry
  round_start_tick: number
  round_end_tick: number
  freeze_end_tick: number
}
