import { create } from "zustand"
import { subscribeWithSelector } from "zustand/middleware"

interface AnalysisState {
  selectedCategory: string | null
  selectedMistakeId: number | null
  hoveredMistakeId: number | null
  setSelectedCategory: (category: string | null) => void
  setSelectedMistakeId: (id: number | null) => void
  setHoveredMistakeId: (id: number | null) => void
  reset: () => void
}

const initialState = {
  selectedCategory: null as string | null,
  selectedMistakeId: null as number | null,
  hoveredMistakeId: null as number | null,
}

export const useAnalysisStore = create<AnalysisState>()(
  subscribeWithSelector((set) => ({
    ...initialState,
    setSelectedCategory: (category) => set({ selectedCategory: category }),
    setSelectedMistakeId: (id) => set({ selectedMistakeId: id }),
    setHoveredMistakeId: (id) => set({ hoveredMistakeId: id }),
    reset: () => set(initialState),
  })),
)
