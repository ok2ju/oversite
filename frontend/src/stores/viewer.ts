import { create } from "zustand"
import { subscribeWithSelector } from "zustand/middleware"

interface ViewerState {
  currentTick: number
  totalTicks: number
  isPlaying: boolean
  speed: number
  currentRound: number
  demoId: string | null
  mapName: string | null
  selectedPlayerSteamId: string | null
  setTick: (tick: number) => void
  setTotalTicks: (total: number) => void
  togglePlay: () => void
  setSpeed: (speed: number) => void
  setRound: (round: number) => void
  setDemoId: (id: string | null) => void
  setMapName: (name: string | null) => void
  pause: () => void
  setSelectedPlayer: (steamId: string | null) => void
  reset: () => void
}

const initialState = {
  currentTick: 0,
  totalTicks: 0,
  isPlaying: false,
  speed: 1,
  currentRound: 1,
  demoId: null as string | null,
  mapName: null as string | null,
  selectedPlayerSteamId: null as string | null,
}

export const useViewerStore = create<ViewerState>()(
  subscribeWithSelector((set) => ({
    ...initialState,
    setTick: (tick) => set({ currentTick: tick }),
    setTotalTicks: (total) => set({ totalTicks: total }),
    togglePlay: () => set((state) => ({ isPlaying: !state.isPlaying })),
    pause: () => set({ isPlaying: false }),
    setSpeed: (speed) => set({ speed }),
    setRound: (round) => set({ currentRound: round, selectedPlayerSteamId: null }),
    setDemoId: (id) => set({ demoId: id, currentTick: 0, mapName: null, selectedPlayerSteamId: null }),
    setMapName: (name) => set({ mapName: name }),
    setSelectedPlayer: (steamId) => set({ selectedPlayerSteamId: steamId }),
    reset: () => set(initialState),
  }))
)
