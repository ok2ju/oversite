// Kind → category mapping for analyzer mistakes. Extracted here (not in
// mistake-list.tsx) because slice 5's <CategoryCard /> will reuse the same
// grouping. Unknown kinds bucket into "other" so a future Go-only rule still
// shows up in the count strip instead of vanishing.
export const KIND_CATEGORY: Record<string, string> = {
  no_trade_death: "trade",
  died_with_util_unused: "utility",
  crosshair_too_low: "aim",
  shot_while_moving: "movement",
}

export const OTHER_CATEGORY = "other"

export const CATEGORY_LABEL: Record<string, string> = {
  trade: "Trade",
  utility: "Utility",
  aim: "Aim",
  movement: "Movement",
  other: "Other",
}

// One-line, player-facing coaching nudge per category. Slice 5 only ships the
// "trade" entry — the remaining cards (utility, aim, …) populate their key as
// they land. Keep the strings short: they slot into a single line below the
// category card's metrics.
export const SUGGESTIONS: Record<string, string> = {
  trade:
    "Trade your teammates' deaths sooner — even one extra trade per half lifts T-side win rate.",
  aim: "Pre-aim head level on every common angle — flicking down loses more time than checking high.",
  movement:
    "Counter-strafe before firing — even small drift past 30 u/s drops your first-bullet accuracy.",
}

export function categoryForKind(kind: string): string {
  return KIND_CATEGORY[kind] ?? OTHER_CATEGORY
}
