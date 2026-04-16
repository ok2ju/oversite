import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup, act } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useViewerStore } from "@/stores/viewer"
import { mockRounds } from "@/test/fixtures/demos"
import { renderWithProviders } from "@/test/render"
import { RoundSelector } from "./round-selector"

vi.mock("@wailsjs/go/main/App", () => ({
  GetDemoRounds: vi.fn().mockResolvedValue(mockRounds),
}))

describe("RoundSelector", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("does not render when demoId is null", () => {
    renderWithProviders(<RoundSelector />)
    expect(screen.queryByTestId("round-selector")).not.toBeInTheDocument()
  })

  it("renders when demo is loaded with rounds", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    expect(await screen.findByTestId("round-selector")).toBeInTheDocument()
  })

  it("displays all round options when opened", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    const trigger = await screen.findByRole("combobox")
    await user.click(trigger)

    // Radix Select renders options with role="option"
    const options = screen.getAllByRole("option")
    expect(options).toHaveLength(mockRounds.length)
  })

  it("shows correct score text for rounds", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    const trigger = await screen.findByRole("combobox")
    await user.click(trigger)

    const options = screen.getAllByRole("option")
    expect(options[0]).toHaveTextContent("Round 1: 1-0")
    expect(options[1]).toHaveTextContent("Round 2: 1-1")
    expect(options[2]).toHaveTextContent("Round 3: 2-1")
  })

  it("selects round and updates store with round number and start tick", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    const trigger = await screen.findByRole("combobox")
    await user.click(trigger)

    const options = screen.getAllByRole("option")
    await user.click(options[1]) // Round 2

    const state = useViewerStore.getState()
    expect(state.currentRound).toBe(2)
    expect(state.currentTick).toBe(mockRounds[1].start_tick)
  })

  it("highlights current round based on tick position", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    // Set tick in round 2 range (3200-6400)
    useViewerStore.getState().setTick(4000)
    renderWithProviders(<RoundSelector />)

    const trigger = await screen.findByRole("combobox")
    expect(trigger).toHaveTextContent("Round 2")
  })

  it("updates displayed round when tick changes", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setTick(1000)
    renderWithProviders(<RoundSelector />)

    const trigger = await screen.findByRole("combobox")
    expect(trigger).toHaveTextContent("Round 1")

    act(() => useViewerStore.getState().setTick(7000))
    expect(trigger).toHaveTextContent("Round 3")
  })
})
