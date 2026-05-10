import { describe, it, expect, beforeEach } from "vitest"
import { useUiStore } from "./ui"

describe("uiStore", () => {
  beforeEach(() => {
    useUiStore.getState().reset()
  })

  it("has correct initial state", () => {
    const state = useUiStore.getState()
    expect(state.sidebarOpen).toBe(true)
    expect(state.activeModal).toBeNull()
    expect(state.mistakeAdvancedOpen).toBe(false)
  })

  it("toggleSidebar toggles sidebarOpen", () => {
    useUiStore.getState().toggleSidebar()
    expect(useUiStore.getState().sidebarOpen).toBe(false)
    useUiStore.getState().toggleSidebar()
    expect(useUiStore.getState().sidebarOpen).toBe(true)
  })

  it("setSidebarOpen sets explicit value", () => {
    useUiStore.getState().setSidebarOpen(false)
    expect(useUiStore.getState().sidebarOpen).toBe(false)
  })

  it("openModal sets activeModal", () => {
    useUiStore.getState().openModal("upload")
    expect(useUiStore.getState().activeModal).toBe("upload")
  })

  it("closeModal clears activeModal", () => {
    useUiStore.getState().openModal("upload")
    useUiStore.getState().closeModal()
    expect(useUiStore.getState().activeModal).toBeNull()
  })

  it("setMistakeAdvancedOpen sets explicit value", () => {
    useUiStore.getState().setMistakeAdvancedOpen(true)
    expect(useUiStore.getState().mistakeAdvancedOpen).toBe(true)
    useUiStore.getState().setMistakeAdvancedOpen(false)
    expect(useUiStore.getState().mistakeAdvancedOpen).toBe(false)
  })

  it("toggleMistakeAdvancedOpen flips the flag", () => {
    expect(useUiStore.getState().mistakeAdvancedOpen).toBe(false)
    useUiStore.getState().toggleMistakeAdvancedOpen()
    expect(useUiStore.getState().mistakeAdvancedOpen).toBe(true)
    useUiStore.getState().toggleMistakeAdvancedOpen()
    expect(useUiStore.getState().mistakeAdvancedOpen).toBe(false)
  })
})
