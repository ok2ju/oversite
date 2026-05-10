import { vi, describe, it, expect, beforeEach, afterEach } from "vitest"
import { screen, waitFor, cleanup } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import { useViewerStore } from "@/stores/viewer"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

import CoachingPage from "@/routes/coaching"

function makeReport(
  overrides: Partial<
    Awaited<ReturnType<typeof mockAppBindings.GetCoachingReport>>
  > = {},
) {
  return {
    steam_id: "STEAM_A",
    lookback: 10,
    habits: [
      {
        key: "counter_strafe",
        label: "Counter-strafe",
        description: "Stop before firing.",
        unit: "ms",
        direction: "lower",
        value: 120,
        status: "warn",
        good_threshold: 100,
        warn_threshold: 200,
        good_min: 0,
        good_max: 0,
        warn_min: 0,
        warn_max: 0,
        previous_value: null,
        delta: null,
        trend: [
          { demo_id: "3", match_date: "2026-05-01", value: 120 },
          { demo_id: "2", match_date: "2026-04-30", value: 150 },
        ],
      },
      {
        key: "first_shot_acc",
        label: "First-shot accuracy",
        description: "Hits on the bullet that decides duels.",
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
        previous_value: null,
        delta: null,
        trend: [
          { demo_id: "3", match_date: "2026-05-01", value: 54 },
          { demo_id: "2", match_date: "2026-04-30", value: 50 },
        ],
      },
    ],
    errors: [{ kind: "missed_first_shot", total: 5 }],
    latest_demo_id: "3",
    last_demo_at: "2026-05-01",
    ...overrides,
  }
}

describe("CoachingPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useViewerStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders empty state when no player is selected", () => {
    renderWithProviders(<CoachingPage />, { initialRoute: "/coaching" })
    expect(screen.getByTestId("coaching-page")).toBeInTheDocument()
    expect(screen.getByTestId("coaching-empty")).toBeInTheDocument()
  })

  it("renders the card grid and errors strip when a player is selected", async () => {
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetCoachingReport.mockResolvedValueOnce(makeReport())

    renderWithProviders(<CoachingPage />, { initialRoute: "/coaching" })

    await waitFor(() => {
      expect(screen.getByTestId("coaching-card-grid")).toBeInTheDocument()
    })
    expect(screen.getByTestId("micro-card-counter_strafe")).toBeInTheDocument()
    expect(screen.getByTestId("micro-card-first_shot_acc")).toBeInTheDocument()
    expect(screen.getByTestId("errors-strip")).toBeInTheDocument()
    expect(
      screen.getByTestId("errors-strip-card-missed_first_shot"),
    ).toBeInTheDocument()
  })

  it("shows the no-data state when the binding returns an empty report", async () => {
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetCoachingReport.mockResolvedValueOnce({
      steam_id: "STEAM_A",
      lookback: 10,
      habits: [],
      errors: [],
      latest_demo_id: "",
      last_demo_at: "",
    })

    renderWithProviders(<CoachingPage />, { initialRoute: "/coaching" })

    await waitFor(() => {
      expect(screen.getByTestId("coaching-no-data")).toBeInTheDocument()
    })
  })

  it("changing the player picker updates the viewer store", async () => {
    mockAppBindings.GetUniquePlayers.mockResolvedValueOnce([
      { steam_id: "STEAM_A", player_name: "Alice" },
      { steam_id: "STEAM_B", player_name: "Bob" },
    ])
    useViewerStore.getState().setSelectedPlayer("STEAM_A")
    mockAppBindings.GetCoachingReport.mockResolvedValue(makeReport())
    const user = userEvent.setup()

    renderWithProviders(<CoachingPage />, { initialRoute: "/coaching" })

    await waitFor(() => {
      expect(screen.getByTestId("coaching-player-picker")).toBeInTheDocument()
    })

    await user.click(screen.getByTestId("coaching-player-picker"))
    await user.click(await screen.findByText("Bob"))

    await waitFor(() => {
      expect(useViewerStore.getState().selectedPlayerSteamId).toBe("STEAM_B")
    })
  })
})
