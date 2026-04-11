import { describe, it, expect, beforeEach, afterEach } from "vitest"
import { screen, cleanup, act } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useViewerStore } from "@/stores/viewer"
import { renderWithProviders } from "@/test/render"
import { PlaybackControls } from "./playback-controls"

describe("PlaybackControls", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
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

  it("displays formatted tick counter", () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    useViewerStore.getState().setTick(12345)
    renderWithProviders(<PlaybackControls />)
    expect(screen.getByTestId("tick-counter")).toHaveTextContent(
      "12,345 / 128,000",
    )
  })

  it("tick counter updates when store changes", () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    renderWithProviders(<PlaybackControls />)

    expect(screen.getByTestId("tick-counter")).toHaveTextContent(
      "0 / 128,000",
    )

    act(() => useViewerStore.getState().setTick(64000))
    expect(screen.getByTestId("tick-counter")).toHaveTextContent(
      "64,000 / 128,000",
    )
  })

  it("renders timeline component", () => {
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setDemoId("demo-1")
    renderWithProviders(<PlaybackControls />)
    expect(screen.getByTestId("timeline-track")).toBeInTheDocument()
  })
})
