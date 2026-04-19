import { describe, it, expect } from "vitest"
import { roundClass } from "@/components/match/round-timeline"

describe("roundClass", () => {
  it("tags a win on my side (CT) as win-ct", () => {
    expect(roundClass({ winner_side: "CT" }, "CT")).toBe("win-ct")
  })

  it("tags a win on my side (T) as win-t", () => {
    expect(roundClass({ winner_side: "T" }, "T")).toBe("win-t")
  })

  it("tags a loss where the other side was T as loss-t", () => {
    expect(roundClass({ winner_side: "T" }, "CT")).toBe("loss-t")
  })

  it("tags a loss where the other side was CT as loss-ct", () => {
    expect(roundClass({ winner_side: "CT" }, "T")).toBe("loss-ct")
  })
})
