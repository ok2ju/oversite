import { create } from "zustand"
import { subscribeWithSelector } from "zustand/middleware"

interface AnalysisState {
  selectedCategory: string | null
  setSelectedCategory: (category: string | null) => void
  reset: () => void
}

const initialState = {
  selectedCategory: null as string | null,
}

export const useAnalysisStore = create<AnalysisState>()(
  subscribeWithSelector((set) => ({
    ...initialState,
    setSelectedCategory: (category) => set({ selectedCategory: category }),
    reset: () => set(initialState),
  })),
)
