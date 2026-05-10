import { vi, describe, it, expect, beforeEach, afterEach } from "vitest"
import { screen, waitFor, cleanup } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import { useAnalysisStore } from "@/stores/analysis"
import { useUiStore } from "@/stores/ui"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

import { MistakeDetail } from "@/components/analysis/mistake-detail"

const baseContext = () => ({
  entry: {
    id: 1,
    kind: "missed_first_shot",
    category: "spray",
    severity: 2,
    title: "Missed first shot",
    suggestion: "Tap, don't spray, on the opener.",
    why_it_hurts:
      "The first bullet is your most accurate one — miss it and you're spraying into recoil to recover.",
    round_number: 4,
    tick: 100,
    steam_id: "STEAM_A",
    extras: {
      weapon: "ak47",
      cause_tag: "shot_before_stop",
      fire_tick: 100,
      speed_at_fire: 153.2,
      weapon_speed_cap: 40,
      ticks_window: [42114, 42115, 42116, 42117],
      speeds: [180, 153, 25, 0],
      yaw_path: [-12.4, -3.1, 0.4, 0.6],
      pitch_path: [-2, -0.5, 0.1, 0],
    } as Record<string, unknown>,
  },
  round_start_tick: 0,
  round_end_tick: 5000,
  freeze_end_tick: 100,
  co_occurring: [
    { id: 2, kind: "no_counter_strafe", title: "No counter-strafe", tick: 100 },
    { id: 3, kind: "shot_while_moving", title: "Shot while moving", tick: 102 },
  ],
})

describe("MistakeDetail", () => {
  beforeEach(() => {
    useAnalysisStore.setState({ selectedMistakeId: 1 })
    useUiStore.getState().reset()
  })
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    useAnalysisStore.setState({ selectedMistakeId: null })
    useUiStore.getState().reset()
    mockAppBindings.GetMistakeContext.mockReset()
    mockAppBindings.GetMistakeContext.mockResolvedValue(null)
  })

  it("renders cause headline, why-it-hurts, speed bar, and co-occurring chips", async () => {
    mockAppBindings.GetMistakeContext.mockResolvedValueOnce(baseContext())

    renderWithProviders(<MistakeDetail id={1} />)

    await waitFor(() =>
      expect(screen.getByTestId("mistake-detail")).toBeInTheDocument(),
    )
    // Cause headline is appended to the title.
    expect(screen.getByTestId("mistake-detail-cause")).toHaveTextContent(
      /shot before stop/i,
    )
    // Why-it-hurts caption renders under the title.
    expect(screen.getByTestId("mistake-detail-why")).toHaveTextContent(
      /first bullet is your most accurate one/i,
    )
    // Speed bar renders all four segments with the expected statuses.
    expect(screen.getByTestId("tick-speed-bar")).toBeInTheDocument()
    expect(screen.getByTestId("tick-speed-segment-0")).toHaveAttribute(
      "data-status",
      "bad",
    )
    expect(screen.getByTestId("tick-speed-segment-3")).toHaveAttribute(
      "data-status",
      "good",
    )
    // Co-occurring chips render one button per sibling.
    expect(
      screen.getByTestId("mistake-detail-co-occurring"),
    ).toBeInTheDocument()
    expect(screen.getByTestId("mistake-detail-co-chip-2")).toHaveTextContent(
      "No counter-strafe",
    )
    expect(screen.getByTestId("mistake-detail-co-chip-3")).toHaveTextContent(
      "Shot while moving",
    )
  })

  it("clicking a co-occurring chip swaps the pinned mistake id", async () => {
    mockAppBindings.GetMistakeContext.mockResolvedValueOnce(baseContext())

    renderWithProviders(<MistakeDetail id={1} />)
    await waitFor(() =>
      expect(screen.getByTestId("mistake-detail")).toBeInTheDocument(),
    )

    await userEvent.click(screen.getByTestId("mistake-detail-co-chip-2"))

    expect(useAnalysisStore.getState().selectedMistakeId).toBe(2)
  })

  it("hides the speed bar and chips when extras lack forensic fields", async () => {
    const ctx = baseContext()
    ctx.entry.extras = { weapon: "ak47" }
    ctx.co_occurring = []
    mockAppBindings.GetMistakeContext.mockResolvedValueOnce(ctx)

    renderWithProviders(<MistakeDetail id={1} />)
    await waitFor(() =>
      expect(screen.getByTestId("mistake-detail")).toBeInTheDocument(),
    )

    expect(screen.queryByTestId("tick-speed-bar")).not.toBeInTheDocument()
    expect(
      screen.queryByTestId("mistake-detail-co-occurring"),
    ).not.toBeInTheDocument()
    expect(screen.queryByTestId("mistake-detail-cause")).not.toBeInTheDocument()
    expect(
      screen.queryByTestId("mistake-detail-advanced"),
    ).not.toBeInTheDocument()
    // Untouched extras still render in the key/value list.
    expect(screen.getByTestId("mistake-detail-extras")).toHaveTextContent(
      /ak47/,
    )
  })

  it("renders the advanced mouse-path expander closed by default", async () => {
    mockAppBindings.GetMistakeContext.mockResolvedValueOnce(baseContext())

    renderWithProviders(<MistakeDetail id={1} />)
    await waitFor(() =>
      expect(screen.getByTestId("mistake-detail-advanced")).toBeInTheDocument(),
    )

    const toggle = screen.getByTestId("mistake-detail-advanced-toggle")
    expect(toggle).toHaveAttribute("aria-expanded", "false")
    expect(
      screen.queryByTestId("mistake-detail-advanced-panel"),
    ).not.toBeInTheDocument()
    expect(screen.queryByTestId("mouse-path")).not.toBeInTheDocument()
  })

  it("toggling the expander reveals the mouse path and persists state", async () => {
    mockAppBindings.GetMistakeContext.mockResolvedValueOnce(baseContext())

    const { unmount } = renderWithProviders(<MistakeDetail id={1} />)
    await waitFor(() =>
      expect(screen.getByTestId("mistake-detail-advanced")).toBeInTheDocument(),
    )

    await userEvent.click(screen.getByTestId("mistake-detail-advanced-toggle"))
    expect(useUiStore.getState().mistakeAdvancedOpen).toBe(true)
    expect(screen.getByTestId("mouse-path")).toBeInTheDocument()
    expect(
      screen.getByTestId("mistake-detail-advanced-toggle"),
    ).toHaveAttribute("aria-expanded", "true")

    // Re-mount and verify the open state is remembered (the store persists
    // across mounts; the next mistake the user clicks opens with the panel
    // already expanded).
    unmount()
    mockAppBindings.GetMistakeContext.mockResolvedValueOnce(baseContext())
    renderWithProviders(<MistakeDetail id={1} />)
    await waitFor(() =>
      expect(screen.getByTestId("mouse-path")).toBeInTheDocument(),
    )
    expect(
      screen.getByTestId("mistake-detail-advanced-toggle"),
    ).toHaveAttribute("aria-expanded", "true")
  })

  it("hides the advanced expander when extras lack yaw/pitch paths", async () => {
    const ctx = baseContext()
    ctx.entry.extras = {
      weapon: "ak47",
      cause_tag: "shot_before_stop",
      speeds: [180, 153, 25, 0],
      weapon_speed_cap: 40,
    } as Record<string, unknown>
    mockAppBindings.GetMistakeContext.mockResolvedValueOnce(ctx)

    renderWithProviders(<MistakeDetail id={1} />)
    await waitFor(() =>
      expect(screen.getByTestId("mistake-detail")).toBeInTheDocument(),
    )

    // Speed bar still renders because speeds array is present.
    expect(screen.getByTestId("tick-speed-bar")).toBeInTheDocument()
    expect(
      screen.queryByTestId("mistake-detail-advanced"),
    ).not.toBeInTheDocument()
  })
})
