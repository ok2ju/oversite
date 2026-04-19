import { describe, it, expect } from "vitest"
import { ratingClass, tierColor } from "@/components/match/scoreboard-table"

describe("ratingClass", () => {
  it("returns hi when rating >= 1.10", () => {
    expect(ratingClass(1.1)).toBe("hi")
    expect(ratingClass(1.45)).toBe("hi")
  })

  it("returns lo when rating < 0.90", () => {
    expect(ratingClass(0.89)).toBe("lo")
    expect(ratingClass(0.5)).toBe("lo")
  })

  it("returns mid in the [0.90, 1.10) band", () => {
    expect(ratingClass(0.9)).toBe("mid")
    expect(ratingClass(1.0)).toBe("mid")
    expect(ratingClass(1.09)).toBe("mid")
  })
})

describe("tierColor", () => {
  it("picks tier 10 for level 10", () => {
    expect(tierColor(10)).toBe("var(--tier-10)")
  })

  it("picks tier 8 for levels 8-9", () => {
    expect(tierColor(8)).toBe("var(--tier-8)")
    expect(tierColor(9)).toBe("var(--tier-8)")
  })

  it("picks tier 6 for levels 6-7", () => {
    expect(tierColor(6)).toBe("var(--tier-6)")
    expect(tierColor(7)).toBe("var(--tier-6)")
  })

  it("picks tier 5 for levels 1-5", () => {
    expect(tierColor(1)).toBe("var(--tier-5)")
    expect(tierColor(5)).toBe("var(--tier-5)")
  })
})
