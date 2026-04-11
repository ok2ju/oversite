import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { Timeline } from "./timeline"

const defaultProps = {
  currentTick: 6400,
  totalTicks: 128000,
  roundBoundaries: [
    { roundNumber: 2, startTick: 3200, endTick: 6400 },
    { roundNumber: 3, startTick: 6400, endTick: 9600 },
  ],
  onSeek: vi.fn(),
}

describe("Timeline", () => {
  it("renders track, progress, and thumb elements", () => {
    render(<Timeline {...defaultProps} />)
    expect(screen.getByTestId("timeline-track")).toBeInTheDocument()
    expect(screen.getByTestId("timeline-progress")).toBeInTheDocument()
    expect(screen.getByTestId("timeline-thumb")).toBeInTheDocument()
  })

  it("progress width matches tick percent", () => {
    render(<Timeline {...defaultProps} />)
    const progress = screen.getByTestId("timeline-progress")
    // 6400 / 128000 = 5%
    expect(progress.style.width).toBe("5%")
  })

  it("renders round markers at correct positions", () => {
    render(<Timeline {...defaultProps} />)
    const markers = screen.getAllByTestId("round-marker")
    expect(markers).toHaveLength(2)
    // Round 2: 3200/128000 = 2.5%
    expect(markers[0].style.left).toBe("2.5%")
    // Round 3: 6400/128000 = 5%
    expect(markers[1].style.left).toBe("5%")
  })

  it("renders no markers when boundaries empty", () => {
    render(<Timeline {...defaultProps} roundBoundaries={[]} />)
    expect(screen.queryAllByTestId("round-marker")).toHaveLength(0)
  })

  it("calls onSeek when track is clicked", () => {
    const onSeek = vi.fn()
    render(<Timeline {...defaultProps} onSeek={onSeek} />)

    const track = screen.getByTestId("timeline-track")
    // Mock getBoundingClientRect
    vi.spyOn(track, "getBoundingClientRect").mockReturnValue({
      left: 0,
      width: 1000,
      top: 0,
      right: 1000,
      bottom: 10,
      height: 10,
      x: 0,
      y: 0,
      toJSON: () => {},
    })

    track.dispatchEvent(
      new MouseEvent("mousedown", { bubbles: true, clientX: 500 }),
    )

    expect(onSeek).toHaveBeenCalled()
    // 500/1000 = 50% → percentToTick(50, 128000) = 64000
    expect(onSeek).toHaveBeenCalledWith(64000)
  })

  it("has correct ARIA attributes", () => {
    render(<Timeline {...defaultProps} />)
    const slider = screen.getByRole("slider")
    expect(slider).toHaveAttribute("aria-valuemin", "0")
    expect(slider).toHaveAttribute("aria-valuemax", "128000")
    expect(slider).toHaveAttribute("aria-valuenow", "6400")
  })
})
