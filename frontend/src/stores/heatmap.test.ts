import { describe, it, expect, beforeEach } from "vitest"
import { useHeatmapStore } from "./heatmap"

describe("heatmapStore", () => {
  beforeEach(() => {
    useHeatmapStore.getState().reset()
  })

  it("has correct initial state", () => {
    const state = useHeatmapStore.getState()
    expect(state.selectedMap).toBeNull()
    expect(state.selectedDemoIds).toEqual([])
    expect(state.selectedWeapons).toEqual([])
    expect(state.selectedPlayer).toBe("")
    expect(state.selectedSide).toBe("")
    expect(state.bandwidth).toBe(18)
    expect(state.opacity).toBe(0.7)
  })

  describe("setMap", () => {
    it("sets map and resets dependent filters", () => {
      useHeatmapStore.getState().setDemoIds([1, 2])
      useHeatmapStore.getState().setWeapons(["AK-47"])
      useHeatmapStore.getState().setPlayer("STEAM_A")
      useHeatmapStore.getState().setSide("CT")

      useHeatmapStore.getState().setMap("de_dust2")

      const state = useHeatmapStore.getState()
      expect(state.selectedMap).toBe("de_dust2")
      expect(state.selectedDemoIds).toEqual([])
      expect(state.selectedWeapons).toEqual([])
      expect(state.selectedPlayer).toBe("")
      expect(state.selectedSide).toBe("")
    })

    it("clears map with null", () => {
      useHeatmapStore.getState().setMap("de_mirage")
      useHeatmapStore.getState().setMap(null)
      expect(useHeatmapStore.getState().selectedMap).toBeNull()
    })
  })

  describe("setDemoIds", () => {
    it("sets demo ids and resets downstream filters", () => {
      useHeatmapStore.getState().setWeapons(["AWP"])
      useHeatmapStore.getState().setPlayer("STEAM_B")

      useHeatmapStore.getState().setDemoIds([1, 2, 3])

      const state = useHeatmapStore.getState()
      expect(state.selectedDemoIds).toEqual([1, 2, 3])
      expect(state.selectedWeapons).toEqual([])
      expect(state.selectedPlayer).toBe("")
    })
  })

  describe("setWeapons", () => {
    it("sets weapons", () => {
      useHeatmapStore.getState().setWeapons(["AK-47", "M4A1"])
      expect(useHeatmapStore.getState().selectedWeapons).toEqual([
        "AK-47",
        "M4A1",
      ])
    })
  })

  describe("setPlayer", () => {
    it("sets player steam id", () => {
      useHeatmapStore.getState().setPlayer("STEAM_A")
      expect(useHeatmapStore.getState().selectedPlayer).toBe("STEAM_A")
    })
  })

  describe("setSide", () => {
    it("sets side filter", () => {
      useHeatmapStore.getState().setSide("T")
      expect(useHeatmapStore.getState().selectedSide).toBe("T")
    })
  })

  describe("setBandwidth", () => {
    it("sets bandwidth", () => {
      useHeatmapStore.getState().setBandwidth(25)
      expect(useHeatmapStore.getState().bandwidth).toBe(25)
    })
  })

  describe("setOpacity", () => {
    it("sets opacity", () => {
      useHeatmapStore.getState().setOpacity(0.5)
      expect(useHeatmapStore.getState().opacity).toBe(0.5)
    })
  })

  describe("reset", () => {
    it("resets all state to initial values", () => {
      useHeatmapStore.getState().setMap("de_inferno")
      useHeatmapStore.getState().setDemoIds([1])
      useHeatmapStore.getState().setWeapons(["AWP"])
      useHeatmapStore.getState().setPlayer("STEAM_X")
      useHeatmapStore.getState().setSide("CT")
      useHeatmapStore.getState().setBandwidth(30)
      useHeatmapStore.getState().setOpacity(0.3)

      useHeatmapStore.getState().reset()

      const state = useHeatmapStore.getState()
      expect(state.selectedMap).toBeNull()
      expect(state.selectedDemoIds).toEqual([])
      expect(state.selectedWeapons).toEqual([])
      expect(state.selectedPlayer).toBe("")
      expect(state.selectedSide).toBe("")
      expect(state.bandwidth).toBe(18)
      expect(state.opacity).toBe(0.7)
    })
  })
})
