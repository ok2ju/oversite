import { create } from "zustand"
import type { FaceitProfile } from "@/types/faceit"

export type { FaceitProfile } from "@/types/faceit"

interface FaceitState {
  profile: FaceitProfile | null
  isLoading: boolean
  setProfile: (profile: FaceitProfile | null) => void
  setLoading: (loading: boolean) => void
  clearProfile: () => void
  reset: () => void
}

const initialState = {
  profile: null as FaceitProfile | null,
  isLoading: false,
}

export const useFaceitStore = create<FaceitState>((set) => ({
  ...initialState,
  setProfile: (profile) => set({ profile }),
  setLoading: (loading) => set({ isLoading: loading }),
  clearProfile: () => set({ profile: null }),
  reset: () => set(initialState),
}))
