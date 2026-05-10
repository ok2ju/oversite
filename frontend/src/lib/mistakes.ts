// Kind → category mapping for analyzer mistakes. Extracted here (not in
// mistake-list.tsx) because slice 5's <CategoryCard /> will reuse the same
// grouping. Unknown kinds bucket into "other" so a future Go-only rule still
// shows up in the count strip instead of vanishing.
export const KIND_CATEGORY: Record<string, string> = {
  no_trade_death: "trade",
  died_with_util_unused: "utility",
}

export const OTHER_CATEGORY = "other"

export const CATEGORY_LABEL: Record<string, string> = {
  trade: "Trade",
  utility: "Utility",
  other: "Other",
}

export function categoryForKind(kind: string): string {
  return KIND_CATEGORY[kind] ?? OTHER_CATEGORY
}
