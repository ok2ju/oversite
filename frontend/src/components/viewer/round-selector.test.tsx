import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup, act } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { useViewerStore } from "@/stores/viewer"
import { mockRounds } from "@/test/fixtures/demos"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings } from "@/test/mocks/bindings"
import type { Round } from "@/types/round"
import { RoundSelector } from "./round-selector"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

function makeRound(overrides: Partial<Round>): Round {
  return {
    id: `round-${overrides.round_number}`,
    round_number: 0,
    start_tick: 0,
    freeze_end_tick: 0,
    end_tick: 0,
    winner_side: "CT",
    win_reason: "TargetBombed",
    ct_score: 0,
    t_score: 0,
    is_overtime: false,
    ...overrides,
  }
}

describe("RoundSelector", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
    mockAppBindings.GetDemoRounds.mockResolvedValue(mockRounds)
  })

  afterEach(() => {
    cleanup()
  })

  it("does not render when demoId is null", () => {
    renderWithProviders(<RoundSelector />)
    expect(screen.queryByTestId("round-selector")).not.toBeInTheDocument()
  })

  it("renders a pill for each round when loaded", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    expect(await screen.findByTestId("round-selector")).toBeInTheDocument()
    for (const round of mockRounds) {
      expect(
        screen.getByTestId(`round-pill-${round.round_number}`),
      ).toBeInTheDocument()
    }
  })

  it("colors pill border by winner side", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    const round1 = await screen.findByTestId("round-pill-1")
    const round2 = screen.getByTestId("round-pill-2")
    expect(round1.className).toMatch(/sky/) // CT
    expect(round2.className).toMatch(/amber/) // T
  })

  it("marks active round based on tick position", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setTick(4000) // round 2 range
    renderWithProviders(<RoundSelector />)

    const activePill = await screen.findByTestId("round-pill-2")
    expect(activePill).toHaveAttribute("aria-current", "true")
    expect(screen.getByTestId("round-pill-1")).not.toHaveAttribute(
      "aria-current",
    )
  })

  it("clicking a pill seeks past freeze time to the live start of the round", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    const round2Pill = await screen.findByTestId("round-pill-2")
    await user.click(round2Pill)

    const state = useViewerStore.getState()
    expect(state.currentRound).toBe(2)
    expect(state.currentTick).toBe(mockRounds[1].freeze_end_tick)
  })

  it("clicking a pill falls back to start_tick when freeze_end_tick is missing", async () => {
    const rounds: Round[] = [
      makeRound({
        round_number: 1,
        start_tick: 0,
        freeze_end_tick: 0, // unknown — e.g. pre-migration data
        end_tick: 3200,
      }),
      makeRound({
        round_number: 2,
        start_tick: 3200,
        freeze_end_tick: 0,
        end_tick: 6400,
      }),
    ]
    mockAppBindings.GetDemoRounds.mockResolvedValueOnce(rounds)
    const user = userEvent.setup()
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    const round2Pill = await screen.findByTestId("round-pill-2")
    await user.click(round2Pill)

    expect(useViewerStore.getState().currentTick).toBe(rounds[1].start_tick)
  })

  it("updates active pill when tick changes", async () => {
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    useViewerStore.getState().setTick(1000)
    renderWithProviders(<RoundSelector />)

    const pill1 = await screen.findByTestId("round-pill-1")
    expect(pill1).toHaveAttribute("aria-current", "true")

    act(() => useViewerStore.getState().setTick(7000))
    expect(screen.getByTestId("round-pill-3")).toHaveAttribute(
      "aria-current",
      "true",
    )
    expect(pill1).not.toHaveAttribute("aria-current")
  })

  it("renders halftime swap marker between rounds 12 and 13", async () => {
    const rounds = Array.from({ length: 13 }, (_, i) =>
      makeRound({
        round_number: i + 1,
        start_tick: i * 1000,
        end_tick: (i + 1) * 1000,
        winner_side: i % 2 === 0 ? "CT" : "T",
      }),
    )
    mockAppBindings.GetDemoRounds.mockResolvedValueOnce(rounds)
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    expect(
      await screen.findByTestId("round-marker-halftime-13"),
    ).toBeInTheDocument()
  })

  it("renders overtime-start clock marker at regulation → OT boundary", async () => {
    const rounds: Round[] = [
      ...Array.from({ length: 24 }, (_, i) =>
        makeRound({
          round_number: i + 1,
          start_tick: i * 1000,
          end_tick: (i + 1) * 1000,
          winner_side: i % 2 === 0 ? "CT" : "T",
        }),
      ),
      makeRound({
        round_number: 25,
        start_tick: 24_000,
        end_tick: 25_000,
        winner_side: "CT",
        is_overtime: true,
      }),
    ]
    mockAppBindings.GetDemoRounds.mockResolvedValueOnce(rounds)
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    expect(
      await screen.findByTestId("round-marker-ot-start-25"),
    ).toBeInTheDocument()
    expect(screen.getByTestId("round-marker-halftime-13")).toBeInTheDocument()
  })

  it("renders OT swap marker between OT halves (after round 27)", async () => {
    const rounds: Round[] = [
      ...Array.from({ length: 24 }, (_, i) =>
        makeRound({
          round_number: i + 1,
          start_tick: i * 1000,
          end_tick: (i + 1) * 1000,
          winner_side: "CT",
        }),
      ),
      ...Array.from({ length: 4 }, (_, i) =>
        makeRound({
          round_number: 25 + i,
          start_tick: (24 + i) * 1000,
          end_tick: (25 + i) * 1000,
          winner_side: "T",
          is_overtime: true,
        }),
      ),
    ]
    mockAppBindings.GetDemoRounds.mockResolvedValueOnce(rounds)
    useViewerStore.getState().setDemoId("1")
    useViewerStore.getState().setTotalTicks(128000)
    renderWithProviders(<RoundSelector />)

    expect(
      await screen.findByTestId("round-marker-ot-swap-28"),
    ).toBeInTheDocument()
  })
})
