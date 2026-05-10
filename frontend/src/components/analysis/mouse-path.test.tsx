import { describe, it, expect, afterEach } from "vitest"
import { screen, cleanup } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"

import { MousePath } from "@/components/analysis/mouse-path"

describe("MousePath", () => {
  afterEach(() => {
    cleanup()
  })

  it("renders an accessible SVG with one dot per tick and per-speed status", () => {
    renderWithProviders(
      <MousePath
        yaws={[-12.4, -3.1, 0.4, 0.6]}
        pitches={[-2, -0.5, 0.1, 0]}
        speeds={[180, 25, 0, 0]}
        weaponSpeedCap={40}
      />,
    )

    const root = screen.getByTestId("mouse-path")
    expect(root).toBeInTheDocument()
    expect(root).toHaveAttribute("data-pitch", "available")

    const svg = screen.getByRole("img", { name: /mouse path/i })
    expect(svg.tagName.toLowerCase()).toBe("svg")
    expect(svg.getAttribute("aria-label")).toMatch(/yaw and pitch tracked/i)

    expect(screen.getByTestId("mouse-path-dot-0")).toHaveAttribute(
      "data-status",
      "bad",
    )
    expect(screen.getByTestId("mouse-path-dot-1")).toHaveAttribute(
      "data-status",
      "warn",
    )
    expect(screen.getByTestId("mouse-path-dot-2")).toHaveAttribute(
      "data-status",
      "good",
    )
    expect(screen.getByTestId("mouse-path-dot-3")).toHaveAttribute(
      "data-status",
      "good",
    )
  })

  it("renders default concentric speed rings with labels", () => {
    renderWithProviders(
      <MousePath yaws={[0, 0, 0]} speeds={[100, 50, 0]} weaponSpeedCap={40} />,
    )

    expect(screen.getByTestId("mouse-path-ring-10")).toBeInTheDocument()
    expect(screen.getByTestId("mouse-path-ring-50")).toBeInTheDocument()
    expect(screen.getByTestId("mouse-path-ring-100")).toBeInTheDocument()
    expect(screen.getByTestId("mouse-path-ring-150")).toBeInTheDocument()
  })

  it("renders the legend with good / warn / bad swatches and the caption", () => {
    renderWithProviders(
      <MousePath yaws={[0, 0]} speeds={[0, 0]} weaponSpeedCap={40} />,
    )

    expect(screen.getByText(/stopped/i)).toBeInTheDocument()
    expect(screen.getByText(/slow/i)).toBeInTheDocument()
    expect(screen.getByText(/moving/i)).toBeInTheDocument()
    expect(
      screen.getByText(/first bullet at center.*mouse path leading/i),
    ).toBeInTheDocument()
  })

  it("falls back to yaw-only when pitches are missing and tags the section", () => {
    renderWithProviders(
      <MousePath
        yaws={[-12.4, -3.1, 0.4, 0.6]}
        speeds={[180, 25, 0, 0]}
        weaponSpeedCap={40}
      />,
    )

    expect(screen.getByTestId("mouse-path")).toHaveAttribute(
      "data-pitch",
      "missing",
    )
    expect(
      screen.getByRole("img", { name: /yaw tracked; pitch unavailable/i }),
    ).toBeInTheDocument()
    // Path still renders with all four dots.
    expect(screen.getByTestId("mouse-path-dot-3")).toBeInTheDocument()
  })

  it("renders nothing when speeds is empty", () => {
    const { container } = renderWithProviders(
      <MousePath yaws={[]} speeds={[]} weaponSpeedCap={40} />,
    )
    expect(container).toBeEmptyDOMElement()
  })
})
