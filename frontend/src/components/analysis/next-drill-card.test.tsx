import { vi, describe, it, expect, beforeEach, afterEach } from "vitest"
import { screen, waitFor, cleanup } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import { useViewerStore } from "@/stores/viewer"
import type { NextDrill } from "@/types/analysis"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

import { NextDrillCard } from "@/components/analysis/next-drill-card"

const DEMO_ID = "1"
const STEAM_ID = "STEAM_A"

function counterStrafeDrill(): NextDrill {
  return {
    key: "counter_strafe",
    title: 'Aim Lab — "Strafe Aim" routine',
    why: "Stop before firing so the first bullet lands.",
    duration: "10 min",
    chips: ["counter-strafe", "1 habit"],
  }
}

function maintenanceDrill(): NextDrill {
  return {
    key: "",
    title: "Light warmup — keep your routine",
    why: "You're hitting your norms. Keep the muscle memory warm.",
    duration: "5 min",
    chips: ["warmup", "maintenance"],
  }
}

function primeViewerStore() {
  useViewerStore.getState().reset()
  useViewerStore.getState().initDemo({
    id: DEMO_ID,
    mapName: "de_mirage",
    totalTicks: 100000,
    tickRate: 64,
  })
  useViewerStore.getState().setSelectedPlayer(STEAM_ID)
}

describe("NextDrillCard", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
    useViewerStore.getState().reset()
  })

  it("shows empty state when no player is selected", () => {
    useViewerStore.getState().reset()
    useViewerStore.getState().initDemo({
      id: DEMO_ID,
      mapName: "de_mirage",
      totalTicks: 100000,
      tickRate: 64,
    })
    renderWithProviders(<NextDrillCard />)
    expect(screen.getByTestId("next-drill-card-empty")).toBeInTheDocument()
  })

  it("renders a catalog drill with title, why, duration, and chips", async () => {
    mockAppBindings.GetNextDrill.mockResolvedValueOnce(counterStrafeDrill())
    primeViewerStore()

    renderWithProviders(<NextDrillCard />)

    await waitFor(() => {
      expect(screen.getByTestId("next-drill-card")).toBeInTheDocument()
    })
    expect(screen.getByTestId("next-drill-card-title")).toHaveTextContent(
      'Aim Lab — "Strafe Aim" routine',
    )
    expect(screen.getByTestId("next-drill-card-why")).toHaveTextContent(
      "Stop before firing so the first bullet lands.",
    )
    expect(screen.getByTestId("next-drill-card-duration")).toHaveTextContent(
      "10 min",
    )
    const chips = screen.getByTestId("next-drill-card-chips")
    expect(chips).toHaveTextContent("counter-strafe")
    expect(chips).toHaveTextContent("1 habit")
    expect(screen.getByTestId("next-drill-card")).not.toHaveAttribute(
      "data-maintenance",
    )
  })

  it("flags the maintenance fallback via data-maintenance", async () => {
    mockAppBindings.GetNextDrill.mockResolvedValueOnce(maintenanceDrill())
    primeViewerStore()

    renderWithProviders(<NextDrillCard />)

    await waitFor(() => {
      expect(screen.getByTestId("next-drill-card")).toBeInTheDocument()
    })
    expect(screen.getByTestId("next-drill-card")).toHaveAttribute(
      "data-maintenance",
      "true",
    )
    expect(screen.getByTestId("next-drill-card-title")).toHaveTextContent(
      "Light warmup — keep your routine",
    )
  })

  it("hides the why paragraph when copy is empty", async () => {
    mockAppBindings.GetNextDrill.mockResolvedValueOnce({
      ...counterStrafeDrill(),
      why: "",
    })
    primeViewerStore()

    renderWithProviders(<NextDrillCard />)

    await waitFor(() => {
      expect(screen.getByTestId("next-drill-card")).toBeInTheDocument()
    })
    expect(screen.queryByTestId("next-drill-card-why")).not.toBeInTheDocument()
  })

  it("hides the chips list when chips is empty", async () => {
    mockAppBindings.GetNextDrill.mockResolvedValueOnce({
      ...counterStrafeDrill(),
      chips: [],
    })
    primeViewerStore()

    renderWithProviders(<NextDrillCard />)

    await waitFor(() => {
      expect(screen.getByTestId("next-drill-card")).toBeInTheDocument()
    })
    expect(
      screen.queryByTestId("next-drill-card-chips"),
    ).not.toBeInTheDocument()
  })
})
