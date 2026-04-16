import { describe, it, expect, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import HeatmapsPage from "@/routes/heatmaps"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

// Mock PixiJS — heatmap canvas creates a PixiJS app
vi.mock("@/lib/pixi/app", () => ({
  createViewerApp: vi.fn().mockResolvedValue({
    stage: { addChild: vi.fn() },
    addLayer: vi
      .fn()
      .mockReturnValue({ addChild: vi.fn(), removeChild: vi.fn() }),
    ticker: { add: vi.fn(), remove: vi.fn(), start: vi.fn(), stop: vi.fn() },
    canvas: document.createElement("canvas"),
    destroy: vi.fn(),
  }),
}))

describe("HeatmapsPage", () => {
  it("renders page with filter panel and canvas", () => {
    renderWithProviders(<HeatmapsPage />)

    expect(screen.getByTestId("heatmaps-page")).toBeInTheDocument()
    expect(screen.getByTestId("heatmap-filter-panel")).toBeInTheDocument()
    expect(screen.getByTestId("heatmap-canvas-container")).toBeInTheDocument()
  })

  it("renders filter controls", () => {
    renderWithProviders(<HeatmapsPage />)

    expect(screen.getByText("Filters")).toBeInTheDocument()
    expect(screen.getByText("Map")).toBeInTheDocument()
    expect(screen.getByText(/Bandwidth/)).toBeInTheDocument()
    expect(screen.getByText(/Opacity/)).toBeInTheDocument()
  })
})
