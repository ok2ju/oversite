import { create } from "zustand"
import { subscribeWithSelector } from "zustand/middleware"
import { DEFAULT_VIEWPORT, type Viewport } from "@/lib/pixi/camera"

// Multi-select state for the round timeline's filter chips. All on by default;
// the slice survives round + player switches so a user-tuned filter set
// persists across context changes (per plan: "remembered when switching
// rounds").
export interface TimelineFilters {
  kills: boolean
  utility: boolean
  bomb: boolean
  myEvents: boolean
}

// Discrete zoom ladder for the round timeline track. 1× fills the dock width;
// higher steps stretch the inner track and make the container horizontally
// scrollable so dense rounds can be inspected without re-clustering at the
// dock's pixel budget.
export const TIMELINE_ZOOM_LEVELS = [1, 1.5, 2, 3, 4, 6, 8] as const
export const MIN_TIMELINE_ZOOM = TIMELINE_ZOOM_LEVELS[0]
export const MAX_TIMELINE_ZOOM =
  TIMELINE_ZOOM_LEVELS[TIMELINE_ZOOM_LEVELS.length - 1]

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
  timelineFilters: TimelineFilters
  timelineZoom: number
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
  setTimelineFilter: (key: keyof TimelineFilters, value: boolean) => void
  setTimelineZoom: (zoom: number) => void
  zoomTimelineIn: () => void
  zoomTimelineOut: () => void
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
  timelineFilters: {
    kills: true,
    utility: true,
    bomb: true,
    myEvents: false,
  } as TimelineFilters,
  timelineZoom: 1,
}

function nextZoomLevel(current: number, direction: 1 | -1): number {
  if (direction === 1) {
    for (const level of TIMELINE_ZOOM_LEVELS) {
      if (level > current + 1e-6) return level
    }
    return MAX_TIMELINE_ZOOM
  }
  for (let i = TIMELINE_ZOOM_LEVELS.length - 1; i >= 0; i--) {
    if (TIMELINE_ZOOM_LEVELS[i] < current - 1e-6) return TIMELINE_ZOOM_LEVELS[i]
  }
  return MIN_TIMELINE_ZOOM
}

function clampZoom(zoom: number): number {
  if (!Number.isFinite(zoom)) return MIN_TIMELINE_ZOOM
  return Math.min(MAX_TIMELINE_ZOOM, Math.max(MIN_TIMELINE_ZOOM, zoom))
}

export const useViewerStore = create<ViewerState>()(
  subscribeWithSelector((set) => ({
    ...initialState,
    setTick: (tick) => set({ currentTick: tick }),
    setTotalTicks: (total) => set({ totalTicks: total }),
    togglePlay: () => set((state) => ({ isPlaying: !state.isPlaying })),
    pause: () => set({ isPlaying: false }),
    setSpeed: (speed) => set({ speed }),
    setRound: (round) => set({ currentRound: round }),
    setDemoId: (id) =>
      set({
        demoId: id,
        currentTick: 0,
        mapName: null,
        selectedPlayerSteamId: null,
        viewport: { ...DEFAULT_VIEWPORT },
        timelineZoom: 1,
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
        timelineZoom: 1,
      }),
    setSelectedPlayer: (steamId) => set({ selectedPlayerSteamId: steamId }),
    setViewport: (v) => set({ viewport: v }),
    setScreenSize: (w, h) => set({ screenWidth: w, screenHeight: h }),
    resetViewport: () =>
      set((state) => ({
        viewport: { ...DEFAULT_VIEWPORT },
        resetViewportCounter: state.resetViewportCounter + 1,
      })),
    setTimelineFilter: (key, value) =>
      set((state) => ({
        timelineFilters: { ...state.timelineFilters, [key]: value },
      })),
    setTimelineZoom: (zoom) => set({ timelineZoom: clampZoom(zoom) }),
    zoomTimelineIn: () =>
      set((state) => ({ timelineZoom: nextZoomLevel(state.timelineZoom, 1) })),
    zoomTimelineOut: () =>
      set((state) => ({ timelineZoom: nextZoomLevel(state.timelineZoom, -1) })),
    reset: () => set(initialState),
  })),
)

// DEV-only: expose the store on window so the Playwright e2e specs can
// read currentTick / isPlaying / setTick directly. Guarded by
// import.meta.env.DEV so production bundles ship without the global.
const __viteEnv = (import.meta as unknown as { env?: { DEV?: boolean } }).env
if (__viteEnv?.DEV && typeof window !== "undefined") {
  ;(
    window as unknown as { __useViewerStore: typeof useViewerStore }
  ).__useViewerStore = useViewerStore
}
