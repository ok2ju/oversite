import { create } from "zustand"
import type { FaceitProfile } from "@/types/faceit"

export type { FaceitProfile } from "@/types/faceit"

interface FaceitState {
  profile: FaceitProfile | null
  isLoading: boolean
  lastSyncedAt: number | null
  setProfile: (profile: FaceitProfile | null) => void
  setLoading: (loading: boolean) => void
  setLastSyncedAt: (ts: number | null) => void
  clearProfile: () => void
  reset: () => void
}

const initialState = {
  profile: null as FaceitProfile | null,
  isLoading: false,
  lastSyncedAt: null as number | null,
}

export const useFaceitStore = create<FaceitState>((set) => ({
  ...initialState,
  setProfile: (profile) => set({ profile }),
  setLoading: (loading) => set({ isLoading: loading }),
  setLastSyncedAt: (lastSyncedAt) => set({ lastSyncedAt }),
  clearProfile: () => set({ profile: null }),
  reset: () => set(initialState),
}))
