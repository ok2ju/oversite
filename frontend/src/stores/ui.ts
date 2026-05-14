import { create } from "zustand"

const SIDEBAR_COLLAPSED_KEY = "oversite:sidebar-collapsed"

function readInitialSidebarCollapsed(): boolean {
  if (typeof window === "undefined") return false
  return window.localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "1"
}

function persistSidebarCollapsed(collapsed: boolean) {
  if (typeof window === "undefined") return
  window.localStorage.setItem(SIDEBAR_COLLAPSED_KEY, collapsed ? "1" : "0")
}

interface UiState {
  sidebarOpen: boolean
  sidebarCollapsed: boolean
  activeModal: string | null
  // mistakeAdvancedOpen — persists the open/closed state of the
  // "Advanced" expander in MistakeDetail (mouse-path / forensic
  // visualizations). Default closed per plan §3.2 density discipline;
  // toggle persists across re-mounts of the panel as the user pages
  // through different mistakes.
  mistakeAdvancedOpen: boolean
  toggleSidebar: () => void
  setSidebarOpen: (open: boolean) => void
  toggleSidebarCollapsed: () => void
  setSidebarCollapsed: (collapsed: boolean) => void
  openModal: (modal: string) => void
  closeModal: () => void
  setMistakeAdvancedOpen: (open: boolean) => void
  toggleMistakeAdvancedOpen: () => void
  reset: () => void
}

const initialState = {
  sidebarOpen: true,
  sidebarCollapsed: readInitialSidebarCollapsed(),
  activeModal: null as string | null,
  mistakeAdvancedOpen: false,
}

export const useUiStore = create<UiState>((set) => ({
  ...initialState,
  toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
  toggleSidebarCollapsed: () =>
    set((state) => {
      const next = !state.sidebarCollapsed
      persistSidebarCollapsed(next)
      return { sidebarCollapsed: next }
    }),
  setSidebarCollapsed: (collapsed) => {
    persistSidebarCollapsed(collapsed)
    set({ sidebarCollapsed: collapsed })
  },
  openModal: (modal) => set({ activeModal: modal }),
  closeModal: () => set({ activeModal: null }),
  setMistakeAdvancedOpen: (open) => set({ mistakeAdvancedOpen: open }),
  toggleMistakeAdvancedOpen: () =>
    set((state) => ({ mistakeAdvancedOpen: !state.mistakeAdvancedOpen })),
  reset: () => {
    persistSidebarCollapsed(false)
    set({ ...initialState, sidebarCollapsed: false })
  },
}))
