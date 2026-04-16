import { create } from "zustand"
import { subscribeWithSelector } from "zustand/middleware"
import type { CS2MapName } from "@/lib/maps/calibration"

interface HeatmapState {
  selectedMap: CS2MapName | null
  selectedDemoIds: number[]
  selectedWeapons: string[]
  selectedPlayer: string
  selectedSide: string
  bandwidth: number
  opacity: number
  setMap: (map: CS2MapName | null) => void
  setDemoIds: (ids: number[]) => void
  setWeapons: (weapons: string[]) => void
  setPlayer: (steamId: string) => void
  setSide: (side: string) => void
  setBandwidth: (bandwidth: number) => void
  setOpacity: (opacity: number) => void
  reset: () => void
}

const initialState = {
  selectedMap: null as CS2MapName | null,
  selectedDemoIds: [] as number[],
  selectedWeapons: [] as string[],
  selectedPlayer: "",
  selectedSide: "",
  bandwidth: 18,
  opacity: 0.7,
}

export const useHeatmapStore = create<HeatmapState>()(
  subscribeWithSelector((set) => ({
    ...initialState,
    setMap: (map) =>
      set({
        selectedMap: map,
        selectedDemoIds: [],
        selectedWeapons: [],
        selectedPlayer: "",
        selectedSide: "",
      }),
    setDemoIds: (ids) =>
      set({
        selectedDemoIds: ids,
        selectedWeapons: [],
        selectedPlayer: "",
        selectedSide: "",
      }),
    setWeapons: (weapons) => set({ selectedWeapons: weapons }),
    setPlayer: (steamId) => set({ selectedPlayer: steamId }),
    setSide: (side) => set({ selectedSide: side }),
    setBandwidth: (bandwidth) => set({ bandwidth }),
    setOpacity: (opacity) => set({ opacity }),
    reset: () => set(initialState),
  })),
)
