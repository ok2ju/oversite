import type { ContactMarker } from "@/lib/timeline/types"

// Returns the contact whose [tPre, tPost] window includes the given tick,
// or null if none does. When multiple contacts overlap (rare — the Phase 2
// builder coalesces them), returns the one with the earlier tFirst.
//
// The active highlight derives from this on every currentTick change; no
// state lives in the store.
export function findActiveContact(
  contacts: readonly ContactMarker[],
  currentTick: number,
): ContactMarker | null {
  let active: ContactMarker | null = null
  for (const c of contacts) {
    if (currentTick < c.tPre || currentTick > c.tPost) continue
    if (active === null || c.tFirst < active.tFirst) {
      active = c
    }
  }
  return active
}
