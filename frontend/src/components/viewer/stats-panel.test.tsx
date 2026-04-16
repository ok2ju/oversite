import { describe, it, expect, vi } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import { StatsPanel } from "./stats-panel"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

describe("StatsPanel", () => {
  it("renders nothing when not visible", () => {
    const { container } = renderWithProviders(
      <StatsPanel demoId="1" visible={false} />,
    )
    expect(container.querySelector("[data-testid='stats-panel']")).toBeNull()
  })

  it("renders weapon stats when visible", async () => {
    renderWithProviders(<StatsPanel demoId="1" visible={true} />)

    expect(screen.getByText("Weapon Stats")).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText("AK-47")).toBeInTheDocument()
    })

    expect(screen.getByText("M4A1")).toBeInTheDocument()
    expect(screen.getByText("AWP")).toBeInTheDocument()
    expect(screen.getByText(/10 kills/)).toBeInTheDocument()
    expect(screen.getByText(/50% HS/)).toBeInTheDocument()
  })

  it("shows no data message when demoId is null", () => {
    renderWithProviders(<StatsPanel demoId={null} visible={true} />)

    expect(screen.getByText("No kill data available")).toBeInTheDocument()
  })
})
