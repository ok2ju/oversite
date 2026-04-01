import { create } from "zustand"

interface UiState {
  sidebarOpen: boolean
  activeModal: string | null
  toggleSidebar: () => void
  setSidebarOpen: (open: boolean) => void
  openModal: (modal: string) => void
  closeModal: () => void
  reset: () => void
}

const initialState = {
  sidebarOpen: true,
  activeModal: null as string | null,
}

export const useUiStore = create<UiState>((set) => ({
  ...initialState,
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
  openModal: (modal) => set({ activeModal: modal }),
  closeModal: () => set({ activeModal: null }),
  reset: () => set(initialState),
}))
