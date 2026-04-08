import { describe, it, expect, beforeEach, afterEach } from "vitest"
import { render, screen, cleanup } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useViewerStore } from "@/stores/viewer"
import { MiniMap } from "./mini-map"

describe("MiniMap", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("does not render when mapName is null", () => {
    render(<MiniMap />)
    expect(screen.queryByTestId("mini-map")).not.toBeInTheDocument()
  })

  it("renders when map is loaded", () => {
    useViewerStore.getState().setMapName("de_dust2")
    render(<MiniMap />)
    expect(screen.getByTestId("mini-map")).toBeInTheDocument()
  })

  it("shows radar image for current map", () => {
    useViewerStore.getState().setMapName("de_mirage")
    render(<MiniMap />)
    const img = screen.getByRole("img", { name: /de_mirage radar/i })
    expect(img.getAttribute("src")).toContain("de_mirage.png")
  })

  it("renders viewport rectangle", () => {
    useViewerStore.getState().setMapName("de_dust2")
    useViewerStore.getState().setScreenSize(1024, 1024)
    render(<MiniMap />)
    expect(screen.getByTestId("mini-map-viewport-rect")).toBeInTheDocument()
  })

  it("viewport rect covers full area at zoom 1", () => {
    useViewerStore.getState().setMapName("de_dust2")
    useViewerStore.getState().setScreenSize(1024, 1024)
    useViewerStore.getState().setViewport({ x: 0, y: 0, zoom: 1 })
    render(<MiniMap />)

    const rect = screen.getByTestId("mini-map-viewport-rect")
    // At zoom 1 with 1024x1024 screen on 1024x1024 map, rect should cover full minimap
    expect(rect.style.width).toBe("100%")
    expect(rect.style.height).toBe("100%")
  })

  it("viewport rect is smaller when zoomed in", () => {
    useViewerStore.getState().setMapName("de_dust2")
    useViewerStore.getState().setScreenSize(1024, 1024)
    useViewerStore.getState().setViewport({ x: 0, y: 0, zoom: 2 })
    render(<MiniMap />)

    const rect = screen.getByTestId("mini-map-viewport-rect")
    // At zoom 2, viewport shows half the map => 50% of minimap
    expect(rect.style.width).toBe("50%")
    expect(rect.style.height).toBe("50%")
  })

  it("reset button calls resetViewport", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setMapName("de_dust2")
    useViewerStore.getState().setViewport({ x: -100, y: -50, zoom: 2 })
    render(<MiniMap />)

    const resetBtn = screen.getByRole("button", { name: /reset view/i })
    await user.click(resetBtn)

    const state = useViewerStore.getState()
    expect(state.viewport).toEqual({ x: 0, y: 0, zoom: 1 })
    expect(state.resetViewportCounter).toBe(1)
  })
})
