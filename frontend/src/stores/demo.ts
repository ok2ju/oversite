import { create } from "zustand"
import type { Demo, DemoStatus } from "@/types/demo"

export interface ImportProgress {
  demoId: number
  fileName: string
  percent: number
  stage: "importing" | "parsing" | "complete" | "error"
}

export interface DemoFilters {
  mapName: string | null
  status: DemoStatus | null
  search: string
}

interface DemoState {
  demos: Demo[]
  selectedDemoId: number | null
  importProgress: ImportProgress | null
  filters: DemoFilters
  setDemos: (demos: Demo[]) => void
  selectDemo: (id: number | null) => void
  updateImportProgress: (progress: ImportProgress | null) => void
  setFilters: (filters: Partial<DemoFilters>) => void
  reset: () => void
}

const initialState = {
  demos: [] as Demo[],
  selectedDemoId: null as number | null,
  importProgress: null as ImportProgress | null,
  filters: {
    mapName: null,
    status: null,
    search: "",
  } as DemoFilters,
}

export const useDemoStore = create<DemoState>((set) => ({
  ...initialState,
  setDemos: (demos) => set({ demos }),
  selectDemo: (id) => set({ selectedDemoId: id }),
  updateImportProgress: (progress) => set({ importProgress: progress }),
  setFilters: (partial) =>
    set((state) => ({ filters: { ...state.filters, ...partial } })),
  reset: () => set({ ...initialState, filters: { ...initialState.filters } }),
}))
