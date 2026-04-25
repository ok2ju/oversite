import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { Logo, ReticleGlyph } from "@/components/brand/logo"

describe("Logo", () => {
  it("renders the Oversite wordmark", () => {
    render(<Logo />)
    expect(screen.getByText("Oversite")).toBeInTheDocument()
  })

  it("renders the reticle glyph beside the wordmark", () => {
    const { container } = render(<Logo />)
    expect(container.querySelector("svg")).toBeInTheDocument()
  })

  it("forwards a className to the wrapper", () => {
    const { container } = render(<Logo className="brand-test" />)
    expect(container.firstChild).toHaveClass("brand-test")
  })

  it("respects the iconSize prop", () => {
    const { container } = render(<Logo iconSize={64} />)
    const svg = container.querySelector("svg")
    expect(svg).toHaveAttribute("width", "64")
    expect(svg).toHaveAttribute("height", "64")
  })
})

describe("ReticleGlyph", () => {
  it("exposes an accessible label when given a title", () => {
    render(<ReticleGlyph title="Oversite" />)
    expect(screen.getByRole("img", { name: "Oversite" })).toBeInTheDocument()
  })

  it("is presentational by default", () => {
    const { container } = render(<ReticleGlyph />)
    const svg = container.querySelector("svg")
    expect(svg).toHaveAttribute("aria-hidden", "true")
  })
})
