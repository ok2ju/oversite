// Kind → category mapping for analyzer mistakes. Extracted here (not in
// mistake-list.tsx) because slice 5's <CategoryCard /> will reuse the same
// grouping. Unknown kinds bucket into "other" so a future Go-only rule still
// shows up in the count strip instead of vanishing.
//
// Slice 10 promoted the canonical mapping to the backend (templates.go) — the
// frontend table is the legacy fallback used when MistakeEntry.category is
// empty (pre-slice-10 rows). New rules should not be added here; they appear
// automatically once the backend `category` field is populated.
export const KIND_CATEGORY: Record<string, string> = {
  no_trade_death: "trade",
  died_with_util_unused: "utility",
  survived_with_util: "utility",
  crosshair_too_low: "aim",
  shot_while_moving: "movement",
  slow_reaction: "aim",
  missed_flick: "aim",
  missed_first_shot: "spray",
  spray_decay: "spray",
  no_counter_strafe: "movement",
  unused_smoke: "utility",
  isolated_peek: "positioning",
  repeated_death_zone: "positioning",
  walked_into_molotov: "utility",
  eco_misbuy: "economy",
  caught_reloading: "aim",
  flash_assist: "utility",
  he_damage: "utility",
}

export const OTHER_CATEGORY = "other"

export const CATEGORY_LABEL: Record<string, string> = {
  trade: "Trade",
  utility: "Utility",
  aim: "Aim",
  spray: "Spray",
  movement: "Movement",
  positioning: "Positioning",
  economy: "Economy",
  round: "Round",
  other: "Other",
}

// One-line, player-facing coaching nudge per category. Used by the legacy
// CategoryCard surface that mounts in the side panel header. Slice 10 added
// per-kind suggestions on the backend (templates.go); per-category nudges
// stay on the frontend because the card is keyed by category, not kind.
export const SUGGESTIONS: Record<string, string> = {
  trade:
    "Trade your teammates' deaths sooner — even one extra trade per half lifts T-side win rate.",
  aim: "Pre-aim head level on every common angle — flicking down loses more time than checking high.",
  spray: "Burst-fire past shot 5 and tap, don't spray, on openers.",
  movement:
    "Counter-strafe before firing — even small drift past 30 u/s drops your first-bullet accuracy.",
  utility:
    "Throw util on the way to your hold — dying with grenades is dropping free damage.",
  positioning:
    "Wait for a teammate within 600 u — peeking alone trades your life for almost nothing.",
  economy:
    "Force-buy when the enemy is also broke — both sides eco'ing is a free round you're refusing to take.",
}

// categoryForKind returns the bucket for a mistake kind. Callers that already
// have a MistakeEntry should prefer entry.category (filled server-side) and
// only fall back to this when the field is empty — slice 10 made the field
// authoritative.
export function categoryForKind(kind: string): string {
  return KIND_CATEGORY[kind] ?? OTHER_CATEGORY
}
