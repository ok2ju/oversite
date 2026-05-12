import { describe, it, expect } from "vitest"
import { findActiveContact } from "./contacts"
import type { ContactMarker } from "@/lib/timeline/types"
import type { main } from "@wailsjs/go/models"

function mk(
  id: number,
  tPre: number,
  tPost: number,
  tFirst = tPre + 32,
): ContactMarker {
  return {
    id,
    subjectSteam: "x",
    tFirst,
    tPre,
    tLast: tPost - 32,
    tPost,
    outcome: "won_clean" as main.ContactOutcome,
    enemies: [],
    mistakes: [],
    worstSeverity: 0,
  }
}

describe("findActiveContact", () => {
  it("returns null for empty contacts", () => {
    expect(findActiveContact([], 1000)).toBeNull()
  })

  it("returns null when no contact contains the tick", () => {
    expect(findActiveContact([mk(1, 1000, 1200)], 500)).toBeNull()
    expect(findActiveContact([mk(1, 1000, 1200)], 1500)).toBeNull()
  })

  it("returns the matching contact when the tick is inside [tPre, tPost]", () => {
    const c = mk(1, 1000, 1200)
    expect(findActiveContact([c], 1000)?.id).toBe(1) // boundary low
    expect(findActiveContact([c], 1100)?.id).toBe(1) // middle
    expect(findActiveContact([c], 1200)?.id).toBe(1) // boundary high
  })

  it("picks the earlier tFirst when two contacts overlap", () => {
    const a = mk(1, 1000, 1300, 1050)
    const b = mk(2, 1100, 1400, 1150)
    expect(findActiveContact([a, b], 1200)?.id).toBe(1)
  })

  it("ignores contacts whose entire window is past the tick", () => {
    const past = mk(1, 1000, 1200)
    const cur = mk(2, 1300, 1500)
    expect(findActiveContact([past, cur], 1400)?.id).toBe(2)
  })
})
