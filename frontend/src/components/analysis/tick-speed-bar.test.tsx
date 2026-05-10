import { describe, it, expect, afterEach } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"

import { TickSpeedBar } from "@/components/analysis/tick-speed-bar"

describe("TickSpeedBar", () => {
  afterEach(() => {
    cleanup()
  })

  it("renders one segment per speed sample with status colors", () => {
    renderWithProviders(
      <TickSpeedBar
        speeds={[153, 25, 0, 0]}
        weaponSpeedCap={40}
        ticksWindow={[42114, 42115, 42116, 42117]}
      />,
    )

    expect(screen.getByTestId("tick-speed-bar")).toBeInTheDocument()
    expect(screen.getByTestId("tick-speed-segment-0")).toHaveAttribute(
      "data-status",
      "bad",
    )
    expect(screen.getByTestId("tick-speed-segment-1")).toHaveAttribute(
      "data-status",
      "warn",
    )
    expect(screen.getByTestId("tick-speed-segment-2")).toHaveAttribute(
      "data-status",
      "good",
    )
    expect(screen.getByTestId("tick-speed-segment-3")).toHaveAttribute(
      "data-status",
      "good",
    )
  })

  it("includes the cap and tick range in the labels", () => {
    renderWithProviders(
      <TickSpeedBar
        speeds={[150, 50, 5]}
        weaponSpeedCap={40}
        ticksWindow={[88, 92, 96]}
      />,
    )

    expect(screen.getByText(/cap 40 u\/s/i)).toBeInTheDocument()
    expect(screen.getByText(/ticks 88–96/)).toBeInTheDocument()
  })

  it("falls back to a sample-count footer when ticks_window is missing", () => {
    renderWithProviders(<TickSpeedBar speeds={[10, 5]} weaponSpeedCap={40} />)

    expect(screen.getByText(/2 sampled ticks/i)).toBeInTheDocument()
  })

  it("renders nothing when the speeds array is empty", () => {
    const { container } = renderWithProviders(
      <TickSpeedBar speeds={[]} weaponSpeedCap={40} />,
    )

    expect(container).toBeEmptyDOMElement()
  })
})
