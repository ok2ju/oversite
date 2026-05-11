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
  shot_while_moving: "movement",
  slow_reaction: "aim",
  missed_first_shot: "spray",
  spray_decay: "spray",
  no_counter_strafe: "movement",
  isolated_peek: "positioning",
  repeated_death_zone: "positioning",
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

// Per-kind "why this hurts" copy mirroring the canonical strings in the Go
// analysis package (`internal/demo/analysis/templates.go`). The plan
// (§4 / §6 of `analysis-overhaul.md`) deliberately keeps WhyItHurts as a
// static lookup on each side instead of widening the MistakeEntry wire
// payload — most surfaces (mistake-detail subtitle from P2-3, coaching
// errors-strip from P5-4, positive-highlight cards for `flash_assist` /
// `he_damage`) already have the kind in hand and only need the copy.
//
// Keep this map in lock-step with the Go side; the test in
// `mistakes.test.ts` enforces that every key in `KIND_CATEGORY` also has
// a `WHY_IT_HURTS` entry so the two stay aligned.
export const WHY_IT_HURTS: Record<string, string> = {
  shot_while_moving:
    "First-bullet accuracy collapses past ~25 u/s of drift, so the duel is decided before the spray.",
  slow_reaction:
    "If you fire 100 ms after the enemy, you've already eaten the bullet that decides the duel.",
  missed_first_shot:
    "The first bullet is your most accurate one — miss it and you're spraying into recoil to recover.",
  spray_decay:
    "Past shot 5 the cone is so wide most bullets miss — you're just feeding ammo into a wall.",
  no_counter_strafe:
    "Without a counter-strafe your rifle's first-bullet cone is closer to a deagle's than a tap kill.",
  isolated_peek:
    "Without a trade nearby, your death is a free pick — the enemy gets the kill and the position.",
  repeated_death_zone:
    "The enemy has read this position — every repeat peek is a duel you're starting at a disadvantage.",
  eco_misbuy:
    "Saving when the enemy is also poor concedes a round you could have stolen with pistols.",
  caught_reloading:
    "You can't shoot back. Whoever swung the angle gets a free kill.",
  flash_assist:
    "A good flash hands your teammate a free duel — losing the habit costs your team easy openers.",
  he_damage:
    "Skipped HE damage is HP your team has to take from rifles instead — every chip shot matters.",
}

// whyItHurts returns the per-kind cost sentence. Unknown kinds return an
// empty string so callers can render conditionally without a guard.
export function whyItHurts(kind: string): string {
  return WHY_IT_HURTS[kind] ?? ""
}
