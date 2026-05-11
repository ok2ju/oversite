import { describe, it, expect } from "vitest"
import { KIND_CATEGORY, WHY_IT_HURTS, whyItHurts } from "./mistakes"

describe("WHY_IT_HURTS", () => {
  it("has a non-empty sentence for every entry", () => {
    for (const [kind, copy] of Object.entries(WHY_IT_HURTS)) {
      expect(copy, `${kind} should have non-empty copy`).not.toBe("")
      expect(copy.trim(), `${kind} should not be whitespace-only`).not.toBe("")
    }
  })

  it("covers exactly the same kinds as KIND_CATEGORY", () => {
    const whyKeys = Object.keys(WHY_IT_HURTS).sort()
    const categoryKeys = Object.keys(KIND_CATEGORY).sort()
    expect(whyKeys).toEqual(categoryKeys)
  })
})

describe("whyItHurts", () => {
  it("returns the canonical copy for known kinds", () => {
    expect(whyItHurts("caught_reloading")).toBe(WHY_IT_HURTS.caught_reloading)
    expect(whyItHurts("slow_reaction")).toBe(WHY_IT_HURTS.slow_reaction)
  })

  it("returns an empty string for unknown kinds", () => {
    expect(whyItHurts("__unknown__")).toBe("")
  })
})
