import { vi, describe, it, expect, beforeEach, afterEach } from "vitest"
import { screen, waitFor, cleanup } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import { useViewerStore } from "@/stores/viewer"
import { useAnalysisStore } from "@/stores/analysis"
import type { HabitReport } from "@/types/analysis"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

import { HabitChecklist } from "@/components/analysis/habit-checklist"

const DEMO_ID = "1"
const STEAM_ID = "STEAM_A"

function buildFixture(): HabitReport {
  return {
    demo_id: DEMO_ID,
    steam_id: STEAM_ID,
    as_of: "2026-05-10T00:00:00Z",
    habits: [
      {
        key: "counter_strafe",
        label: "Counter-strafe",
        description: "Stop before firing",
        unit: "ms",
        direction: "lower",
        value: 240,
        status: "warn",
        good_threshold: 100,
        warn_threshold: 200,
        good_min: 0,
        good_max: 0,
        warn_min: 0,
        warn_max: 0,
        previous_value: 252,
        delta: -12,
      },
      {
        key: "first_shot_acc",
        label: "First-shot accuracy",
        description: "Hits on the bullet that decides duels",
        unit: "%",
        direction: "higher",
        value: 54,
        status: "good",
        good_threshold: 50,
        warn_threshold: 35,
        good_min: 0,
        good_max: 0,
        warn_min: 0,
        warn_max: 0,
        previous_value: 51,
        delta: 3,
      },
      {
        key: "reaction",
        label: "Reaction",
        description: "Time from enemy visible to first shot",
        unit: "ms",
        direction: "lower",
        value: 320,
        status: "bad",
        good_threshold: 200,
        warn_threshold: 280,
        good_min: 0,
        good_max: 0,
        warn_min: 0,
        warn_max: 0,
        previous_value: null,
        delta: null,
      },
      {
        key: "trade_timing",
        label: "Trade timing",
        description: "Share of teammates' deaths you traded back",
        unit: "%",
        direction: "higher",
        value: 45,
        status: "bad",
        good_threshold: 70,
        warn_threshold: 50,
        good_min: 0,
        good_max: 0,
        warn_min: 0,
        warn_max: 0,
        previous_value: null,
        delta: null,
      },
      {
        key: "flick_balance",
        label: "Flick balance",
        description: "Over- vs under-flicks",
        unit: "%",
        direction: "balanced",
        value: 62,
        status: "warn",
        good_threshold: 0,
        warn_threshold: 0,
        good_min: 45,
        good_max: 55,
        warn_min: 40,
        warn_max: 60,
        previous_value: null,
        delta: null,
      },
    ],
  }
}

function primeViewer(steamId: string | null = STEAM_ID) {
  const viewer = useViewerStore.getState()
  viewer.initDemo({
    id: DEMO_ID,
    mapName: "de_dust2",
    totalTicks: 1000,
    tickRate: 64,
  })
  if (steamId) {
    viewer.setSelectedPlayer(steamId)
  }
}

describe("HabitChecklist", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useViewerStore.getState().reset()
    useAnalysisStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders 6 rows for the fixture", async () => {
    mockAppBindings.GetHabitReport.mockResolvedValueOnce(buildFixture())
    primeViewer()

    renderWithProviders(<HabitChecklist />)

    await waitFor(() => {
      expect(screen.getByTestId("habit-checklist")).toBeInTheDocument()
    })
    expect(
      screen.getByTestId("habit-checklist-row-counter_strafe"),
    ).toBeInTheDocument()
    expect(
      screen.getByTestId("habit-checklist-row-first_shot_acc"),
    ).toBeInTheDocument()
    expect(
      screen.getByTestId("habit-checklist-row-reaction"),
    ).toBeInTheDocument()
    expect(
      screen.getByTestId("habit-checklist-row-trade_timing"),
    ).toBeInTheDocument()
    expect(
      screen.getByTestId("habit-checklist-row-flick_balance"),
    ).toBeInTheDocument()
    expect(
      screen.getByTestId("habit-checklist-row-trade_timing"),
    ).toBeInTheDocument()
  })

  it("clicking a bad row sets selectedCategory via the habit→category map", async () => {
    mockAppBindings.GetHabitReport.mockResolvedValueOnce(buildFixture())
    primeViewer()

    renderWithProviders(<HabitChecklist />)

    const user = userEvent.setup()
    await waitFor(() => {
      expect(
        screen.getByTestId("habit-checklist-row-trade_timing"),
      ).toBeInTheDocument()
    })

    await user.click(screen.getByTestId("habit-checklist-row-trade_timing"))

    expect(useAnalysisStore.getState().selectedCategory).toBe("trade")

    // Click again on the same row clears the filter (toggle).
    await user.click(screen.getByTestId("habit-checklist-row-trade_timing"))
    expect(useAnalysisStore.getState().selectedCategory).toBeNull()
  })

  it.each([
    ["counter_strafe", "rgb(255, 194, 51)"], // warn → #ffc233
    ["first_shot_acc", "rgb(155, 188, 90)"], // good → #9bbc5a
    ["reaction", "rgb(248, 113, 113)"], // bad  → #f87171
  ])("status pill color for %s matches the status enum", async (key, rgb) => {
    mockAppBindings.GetHabitReport.mockResolvedValueOnce(buildFixture())
    primeViewer()

    renderWithProviders(<HabitChecklist />)

    await waitFor(() => {
      expect(
        screen.getByTestId(`habit-checklist-pill-${key}`),
      ).toBeInTheDocument()
    })

    const pill = screen.getByTestId(`habit-checklist-pill-${key}`)
    expect(pill.style.backgroundColor).toBe(rgb)
  })

  it("renders the empty state when no player is selected", async () => {
    mockAppBindings.GetHabitReport.mockResolvedValueOnce(buildFixture())
    primeViewer(null)

    renderWithProviders(<HabitChecklist />)

    await waitFor(() => {
      expect(screen.getByTestId("habit-checklist-empty")).toBeInTheDocument()
    })
    expect(screen.queryByTestId("habit-checklist")).not.toBeInTheDocument()
  })

  it("renders the delta line when delta is non-null", async () => {
    mockAppBindings.GetHabitReport.mockResolvedValueOnce(buildFixture())
    primeViewer()

    renderWithProviders(<HabitChecklist />)

    await waitFor(() => {
      expect(
        screen.getByTestId("habit-checklist-delta-counter_strafe"),
      ).toBeInTheDocument()
    })

    // counter_strafe is LowerIsBetter, delta = -12 → improving (down arrow,
    // green). Unit is ms.
    const csDelta = screen.getByTestId("habit-checklist-delta-counter_strafe")
    expect(csDelta.textContent).toContain("↓")
    expect(csDelta.textContent).toContain("12 ms")

    // first_shot_acc is HigherIsBetter, delta = +3 (%) → improving (up
    // arrow). Percentage deltas render in pp.
    const fsDelta = screen.getByTestId("habit-checklist-delta-first_shot_acc")
    expect(fsDelta.textContent).toContain("↑")
    expect(fsDelta.textContent).toContain("3 pp")
  })

  it("hides the delta line when delta is null", async () => {
    mockAppBindings.GetHabitReport.mockResolvedValueOnce(buildFixture())
    primeViewer()

    renderWithProviders(<HabitChecklist />)

    await waitFor(() => {
      expect(
        screen.getByTestId("habit-checklist-row-reaction"),
      ).toBeInTheDocument()
    })

    expect(
      screen.queryByTestId("habit-checklist-delta-reaction"),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByTestId("habit-checklist-delta-trade_timing"),
    ).not.toBeInTheDocument()
  })
})
