import { describe, it, expect, beforeEach } from "vitest"
import { useDemoStore } from "./demo"
import type { Demo } from "@/types/demo"

const demoDust2: Demo = {
  id: 1,
  map_name: "de_dust2",
  file_path: "/Users/test/demos/match1.dem",
  file_size: 150_000_000,
  status: "ready",
  total_ticks: 128000,
  tick_rate: 64,
  duration_secs: 2000,
  match_date: "2026-03-01T18:00:00Z",
  created_at: "2026-03-01T19:00:00Z",
}

const demoMirage: Demo = {
  id: 2,
  map_name: "de_mirage",
  file_path: "/Users/test/demos/match2.dem",
  file_size: 120_000_000,
  status: "parsing",
  total_ticks: 0,
  tick_rate: 0,
  duration_secs: 0,
  match_date: "",
  created_at: "2026-03-02T10:00:00Z",
}

const demoInferno: Demo = {
  id: 3,
  map_name: "de_inferno",
  file_path: "/Users/test/demos/match3.dem",
  file_size: 140_000_000,
  status: "imported",
  total_ticks: 0,
  tick_rate: 0,
  duration_secs: 0,
  match_date: "",
  created_at: "2026-03-03T12:00:00Z",
}

describe("demoStore", () => {
  beforeEach(() => {
    useDemoStore.getState().reset()
  })

  it("has correct initial state", () => {
    const state = useDemoStore.getState()
    expect(state.demos).toEqual([])
    expect(state.selectedDemoId).toBeNull()
    expect(state.importProgress).toBeNull()
    expect(state.filters).toEqual({
      mapName: null,
      status: null,
      search: "",
    })
  })

  describe("setDemos", () => {
    it("sets the demos list", () => {
      useDemoStore.getState().setDemos([demoDust2, demoMirage])
      expect(useDemoStore.getState().demos).toEqual([demoDust2, demoMirage])
    })

    it("replaces existing demos", () => {
      useDemoStore.getState().setDemos([demoDust2])
      useDemoStore.getState().setDemos([demoMirage, demoInferno])
      expect(useDemoStore.getState().demos).toEqual([demoMirage, demoInferno])
    })
  })

  describe("selectDemo", () => {
    it("sets selectedDemoId", () => {
      useDemoStore.getState().selectDemo(1)
      expect(useDemoStore.getState().selectedDemoId).toBe(1)
    })

    it("clears selectedDemoId with null", () => {
      useDemoStore.getState().selectDemo(1)
      useDemoStore.getState().selectDemo(null)
      expect(useDemoStore.getState().selectedDemoId).toBeNull()
    })
  })

  describe("updateImportProgress", () => {
    it("sets import progress", () => {
      const progress = {
        demoId: 1,
        fileName: "match1.dem",
        percent: 50,
        stage: "parsing" as const,
      }
      useDemoStore.getState().updateImportProgress(progress)
      expect(useDemoStore.getState().importProgress).toEqual(progress)
    })

    it("updates progress incrementally", () => {
      useDemoStore.getState().updateImportProgress({
        demoId: 1,
        fileName: "match1.dem",
        percent: 25,
        stage: "importing",
      })
      useDemoStore.getState().updateImportProgress({
        demoId: 1,
        fileName: "match1.dem",
        percent: 100,
        stage: "complete",
      })
      expect(useDemoStore.getState().importProgress).toEqual({
        demoId: 1,
        fileName: "match1.dem",
        percent: 100,
        stage: "complete",
      })
    })

    it("clears progress with null", () => {
      useDemoStore.getState().updateImportProgress({
        demoId: 1,
        fileName: "match1.dem",
        percent: 100,
        stage: "complete",
      })
      useDemoStore.getState().updateImportProgress(null)
      expect(useDemoStore.getState().importProgress).toBeNull()
    })
  })

  describe("setFilters", () => {
    it("sets mapName filter", () => {
      useDemoStore.getState().setFilters({ mapName: "de_dust2" })
      expect(useDemoStore.getState().filters).toEqual({
        mapName: "de_dust2",
        status: null,
        search: "",
      })
    })

    it("sets status filter", () => {
      useDemoStore.getState().setFilters({ status: "ready" })
      expect(useDemoStore.getState().filters).toEqual({
        mapName: null,
        status: "ready",
        search: "",
      })
    })

    it("sets search filter", () => {
      useDemoStore.getState().setFilters({ search: "dust" })
      expect(useDemoStore.getState().filters).toEqual({
        mapName: null,
        status: null,
        search: "dust",
      })
    })

    it("merges partial filter updates", () => {
      useDemoStore.getState().setFilters({ mapName: "de_dust2" })
      useDemoStore.getState().setFilters({ status: "ready" })
      expect(useDemoStore.getState().filters).toEqual({
        mapName: "de_dust2",
        status: "ready",
        search: "",
      })
    })

    it("clears individual filters", () => {
      useDemoStore
        .getState()
        .setFilters({ mapName: "de_dust2", status: "ready" })
      useDemoStore.getState().setFilters({ mapName: null })
      expect(useDemoStore.getState().filters).toEqual({
        mapName: null,
        status: "ready",
        search: "",
      })
    })
  })

  describe("reset", () => {
    it("resets all state to initial values", () => {
      useDemoStore.getState().setDemos([demoDust2, demoMirage])
      useDemoStore.getState().selectDemo(1)
      useDemoStore.getState().updateImportProgress({
        demoId: 1,
        fileName: "match1.dem",
        percent: 50,
        stage: "parsing",
      })
      useDemoStore
        .getState()
        .setFilters({ mapName: "de_dust2", search: "dust" })

      useDemoStore.getState().reset()

      const state = useDemoStore.getState()
      expect(state.demos).toEqual([])
      expect(state.selectedDemoId).toBeNull()
      expect(state.importProgress).toBeNull()
      expect(state.filters).toEqual({
        mapName: null,
        status: null,
        search: "",
      })
    })
  })
})
