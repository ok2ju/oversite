import { describe, it, expect, beforeEach } from "vitest"
import { useAnalysisStore } from "./analysis"

describe("analysisStore", () => {
  beforeEach(() => {
    useAnalysisStore.getState().reset()
  })

  it("has correct initial state", () => {
    expect(useAnalysisStore.getState().selectedCategory).toBeNull()
  })

  describe("setSelectedCategory", () => {
    it("sets the selected category", () => {
      useAnalysisStore.getState().setSelectedCategory("trade")
      expect(useAnalysisStore.getState().selectedCategory).toBe("trade")
    })

    it("clears the selected category with null", () => {
      useAnalysisStore.getState().setSelectedCategory("utility")
      useAnalysisStore.getState().setSelectedCategory(null)
      expect(useAnalysisStore.getState().selectedCategory).toBeNull()
    })
  })

  describe("reset", () => {
    it("resets state to initial values", () => {
      useAnalysisStore.getState().setSelectedCategory("trade")
      useAnalysisStore.getState().reset()
      expect(useAnalysisStore.getState().selectedCategory).toBeNull()
    })
  })
})
