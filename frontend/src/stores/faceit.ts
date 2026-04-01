import { create } from "zustand"

interface FaceitProfile {
  id: string
  nickname: string
  avatarUrl: string | null
  elo: number
  level: number
  country: string
}

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
