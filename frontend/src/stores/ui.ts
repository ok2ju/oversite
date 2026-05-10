import { create } from "zustand"

interface UiState {
  sidebarOpen: boolean
  activeModal: string | null
  // mistakeAdvancedOpen — persists the open/closed state of the
  // "Advanced" expander in MistakeDetail (mouse-path / forensic
  // visualizations). Default closed per plan §3.2 density discipline;
  // toggle persists across re-mounts of the panel as the user pages
  // through different mistakes.
  mistakeAdvancedOpen: boolean
  toggleSidebar: () => void
  setSidebarOpen: (open: boolean) => void
  openModal: (modal: string) => void
  closeModal: () => void
  setMistakeAdvancedOpen: (open: boolean) => void
  toggleMistakeAdvancedOpen: () => void
  reset: () => void
}

const initialState = {
  sidebarOpen: true,
  activeModal: null as string | null,
  mistakeAdvancedOpen: false,
}

export const useUiStore = create<UiState>((set) => ({
  ...initialState,
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
  openModal: (modal) => set({ activeModal: modal }),
  closeModal: () => set({ activeModal: null }),
  setMistakeAdvancedOpen: (open) => set({ mistakeAdvancedOpen: open }),
  toggleMistakeAdvancedOpen: () =>
    set((state) => ({ mistakeAdvancedOpen: !state.mistakeAdvancedOpen })),
  reset: () => set(initialState),
}))
