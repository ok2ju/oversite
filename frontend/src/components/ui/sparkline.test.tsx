import { describe, it, expect } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { Sparkline } from "@/components/ui/sparkline"

describe("Sparkline", () => {
  it("renders an SVG path with at least 2 points", () => {
    renderWithProviders(
      <Sparkline
        points={[{ value: 1 }, { value: 5 }, { value: 3 }]}
        ariaLabel="test trend"
      />,
    )
    const svg = screen.getByTestId("sparkline")
    expect(svg).toBeInTheDocument()
    expect(svg).toHaveAttribute("aria-label", "test trend")
  })

  it("renders the empty placeholder when fewer than 2 points", () => {
    renderWithProviders(<Sparkline points={[{ value: 1 }]} />)
    expect(screen.getByTestId("sparkline-empty")).toBeInTheDocument()
    expect(screen.queryByTestId("sparkline")).not.toBeInTheDocument()
  })

  it("renders nothing when given no points", () => {
    renderWithProviders(<Sparkline points={[]} />)
    expect(screen.getByTestId("sparkline-empty")).toBeInTheDocument()
  })
})
