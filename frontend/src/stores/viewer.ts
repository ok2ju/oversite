import { create } from "zustand"
import { subscribeWithSelector } from "zustand/middleware"
import { DEFAULT_VIEWPORT, type Viewport } from "@/lib/pixi/camera"

interface ViewerState {
  currentTick: number
  totalTicks: number
  isPlaying: boolean
  speed: number
  currentRound: number
  demoId: string | null
  mapName: string | null
  tickRate: number
  selectedPlayerSteamId: string | null
  viewport: Viewport
  screenWidth: number
  screenHeight: number
  resetViewportCounter: number
  setTick: (tick: number) => void
  setTotalTicks: (total: number) => void
  togglePlay: () => void
  setSpeed: (speed: number) => void
  setRound: (round: number) => void
  setDemoId: (id: string | null) => void
  setMapName: (name: string | null) => void
  initDemo: (opts: {
    id: string
    mapName: string
    totalTicks: number
    tickRate: number
  }) => void
  pause: () => void
  setSelectedPlayer: (steamId: string | null) => void
  setViewport: (v: Viewport) => void
  setScreenSize: (w: number, h: number) => void
  resetViewport: () => void
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
  tickRate: 64,
  selectedPlayerSteamId: null as string | null,
  viewport: { ...DEFAULT_VIEWPORT },
  screenWidth: 0,
  screenHeight: 0,
  resetViewportCounter: 0,
}

export const useViewerStore = create<ViewerState>()(
  subscribeWithSelector((set) => ({
    ...initialState,
    setTick: (tick) => set({ currentTick: tick }),
    setTotalTicks: (total) => set({ totalTicks: total }),
    togglePlay: () => set((state) => ({ isPlaying: !state.isPlaying })),
    pause: () => set({ isPlaying: false }),
    setSpeed: (speed) => set({ speed }),
    setRound: (round) =>
      set({ currentRound: round, selectedPlayerSteamId: null }),
    setDemoId: (id) =>
      set({
        demoId: id,
        currentTick: 0,
        mapName: null,
        selectedPlayerSteamId: null,
        viewport: { ...DEFAULT_VIEWPORT },
      }),
    setMapName: (name) => set({ mapName: name }),
    initDemo: (opts) =>
      set({
        demoId: opts.id,
        mapName: opts.mapName,
        totalTicks: opts.totalTicks,
        tickRate: opts.tickRate,
        currentTick: 0,
        selectedPlayerSteamId: null,
        viewport: { ...DEFAULT_VIEWPORT },
      }),
    setSelectedPlayer: (steamId) => set({ selectedPlayerSteamId: steamId }),
    setViewport: (v) => set({ viewport: v }),
    setScreenSize: (w, h) => set({ screenWidth: w, screenHeight: h }),
    resetViewport: () =>
      set((state) => ({
        viewport: { ...DEFAULT_VIEWPORT },
        resetViewportCounter: state.resetViewportCounter + 1,
      })),
    reset: () => set(initialState),
  })),
)
