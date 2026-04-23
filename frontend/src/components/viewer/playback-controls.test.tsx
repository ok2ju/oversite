import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup, act, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useViewerStore } from "@/stores/viewer"
import { mockRounds } from "@/test/fixtures/demos"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings } from "@/test/mocks/bindings"
import { PlaybackControls } from "./playback-controls"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

describe("PlaybackControls", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
    mockAppBindings.GetDemoRounds.mockResolvedValue(mockRounds)
  })

  afterEach(() => {
    cleanup()
  })

  it("does not render when totalTicks is 0", () => {
    renderWithProviders(<PlaybackControls />)
    expect(screen.queryByTestId("playback-controls")).not.toBeInTheDocument()
  })

  it("renders when demo is loaded", () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    renderWithProviders(<PlaybackControls />)
    expect(screen.getByTestId("playback-controls")).toBeInTheDocument()
  })

  it("shows play button when paused", () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    renderWithProviders(<PlaybackControls />)
    expect(screen.getByRole("button", { name: /play/i })).toBeInTheDocument()
  })

  it("shows pause button when playing", () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().togglePlay()
    renderWithProviders(<PlaybackControls />)
    expect(screen.getByRole("button", { name: /pause/i })).toBeInTheDocument()
  })

  it("click toggles isPlaying in store", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    renderWithProviders(<PlaybackControls />)

    const playBtn = screen.getByRole("button", { name: /play/i })
    await user.click(playBtn)
    expect(useViewerStore.getState().isPlaying).toBe(true)

    const pauseBtn = screen.getByRole("button", { name: /pause/i })
    await user.click(pauseBtn)
    expect(useViewerStore.getState().isPlaying).toBe(false)
  })

  it("shows current speed in speed trigger", () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    renderWithProviders(<PlaybackControls />)
    expect(screen.getByTestId("speed-trigger")).toHaveTextContent("1x")
  })

  it("round time falls back to whole-game elapsed when rounds are not yet loaded", () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId(null)
    useViewerStore.getState().setTick(64 * 113) // 1:53
    renderWithProviders(<PlaybackControls />)
    expect(screen.getByTestId("round-time")).toHaveTextContent("1:53")
  })

  // Timeline is scoped to the live portion of each round (post-freeze-time).
  // Round 1: freeze_end_tick = 960 (0:15 of 15s freeze), live window 960..3200.
  // Round 2: freeze_end_tick = 4160, live window 4160..6400.
  // Round 3: freeze_end_tick = 7360, live window 7360..9600.
  it("round time displays elapsed live seconds within the active round", async () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().setTick(4160 + 64 * 25) // 25s into live round 2
    renderWithProviders(<PlaybackControls />)

    await waitFor(() => {
      expect(screen.getByTestId("round-time")).toHaveTextContent("0:25")
    })
  })

  it("round time is 0:00 while still in freeze time", async () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().setTick(0)
    renderWithProviders(<PlaybackControls />)

    await waitFor(() => {
      expect(screen.getByTestId("round-time")).toHaveTextContent("0:00")
    })

    // Still inside round 1 freeze (freeze_end_tick = 960) — clamp to 0:00.
    act(() => useViewerStore.getState().setTick(64 * 10))
    expect(screen.getByTestId("round-time")).toHaveTextContent("0:00")

    // Past freeze into live round 1.
    act(() => useViewerStore.getState().setTick(960 + 64 * 10))
    expect(screen.getByTestId("round-time")).toHaveTextContent("0:10")
  })

  it("round time rescales when tick crosses round boundary", async () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().setTick(960 + 64 * 10) // 10s into live round 1
    renderWithProviders(<PlaybackControls />)

    await waitFor(() => {
      expect(screen.getByTestId("round-time")).toHaveTextContent("0:10")
    })

    // Jump into live round 3 (starts at 6400, freeze_end 7360); 5s into it.
    act(() => useViewerStore.getState().setTick(7360 + 64 * 5))
    expect(screen.getByTestId("round-time")).toHaveTextContent("0:05")
  })

  it("renders timeline component", () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    renderWithProviders(<PlaybackControls />)
    expect(screen.getByTestId("timeline-track")).toBeInTheDocument()
  })

  it("does not render round boundary markers on the timeline", async () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    renderWithProviders(<PlaybackControls />)

    await waitFor(() => {
      expect(screen.getByTestId("round-time")).toBeInTheDocument()
    })

    expect(screen.queryAllByTestId("round-marker")).toHaveLength(0)
  })
})
