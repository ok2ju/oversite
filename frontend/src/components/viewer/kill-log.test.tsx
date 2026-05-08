import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { screen, cleanup, act, waitFor } from "@testing-library/react"
import { useViewerStore } from "@/stores/viewer"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings } from "@/test/mocks/bindings"
import type { GameEvent } from "@/types/demo"
import { KillLog } from "./kill-log"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

const DEMO_ID = "demo-1"

function makeKill(opts: {
  id: string
  tick: number
  attackerName: string
  attackerSide: "CT" | "T"
  victimName: string
  victimSide: "CT" | "T"
  weapon?: string
  headshot?: boolean
}): GameEvent {
  return {
    id: opts.id,
    demo_id: DEMO_ID,
    round_id: null,
    tick: opts.tick,
    event_type: "kill",
    attacker_steam_id: "76561198000000001",
    victim_steam_id: "76561198000000002",
    weapon: opts.weapon ?? "AK-47",
    x: 0,
    y: 0,
    z: 0,
    headshot: opts.headshot ?? false,
    assister_steam_id: null,
    health_damage: 0,
    attacker_name: opts.attackerName,
    attacker_team: opts.attackerSide,
    victim_name: opts.victimName,
    victim_team: opts.victimSide,
    extra_data: {},
  }
}

describe("KillLog", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
    mockAppBindings.GetEventsByTypes.mockClear()
  })

  it("renders nothing when there is no demo", () => {
    renderWithProviders(<KillLog />)
    expect(screen.queryByTestId("kill-log")).not.toBeInTheDocument()
  })

  it("renders nothing when no kills are inside the rolling window", async () => {
    mockAppBindings.GetEventsByTypes.mockResolvedValueOnce([
      makeKill({
        id: "old",
        tick: 0,
        attackerName: "s1lent",
        attackerSide: "T",
        victimName: "Sakamoto",
        victimSide: "CT",
      }),
    ])
    useViewerStore.getState().setDemoId(DEMO_ID)
    useViewerStore.getState().setTick(64 * 60) // 60s later → outside default 7s window

    renderWithProviders(<KillLog />)
    // Wait for the query to resolve.
    await waitFor(() => {
      expect(mockAppBindings.GetEventsByTypes).toHaveBeenCalled()
    })
    expect(screen.queryByTestId("kill-log")).not.toBeInTheDocument()
  })

  it("renders kills in the rolling window with side colors and headshot icon", async () => {
    mockAppBindings.GetEventsByTypes.mockResolvedValueOnce([
      makeKill({
        id: "k1",
        tick: 64 * 1, // 1s ago
        attackerName: "s1lent",
        attackerSide: "T",
        victimName: "Sakamoto",
        victimSide: "CT",
        weapon: "AK-47",
        headshot: true,
      }),
      makeKill({
        id: "k2",
        tick: 64 * 3, // 0s ago — most recent
        attackerName: "yeda",
        attackerSide: "T",
        victimName: "xns",
        victimSide: "CT",
        weapon: "AK-47",
        headshot: false,
      }),
    ])
    useViewerStore.getState().setDemoId(DEMO_ID)
    useViewerStore.getState().setTick(64 * 3)

    renderWithProviders(<KillLog />)

    expect(await screen.findByTestId("kill-log")).toBeInTheDocument()

    // Latest kill renders last so newer rows slide in at the bottom.
    const rows = screen.getAllByTestId(/kill-log-row-/)
    expect(rows).toHaveLength(2)
    expect(rows[0]).toHaveAttribute("data-testid", "kill-log-row-k1")
    expect(rows[1]).toHaveAttribute("data-testid", "kill-log-row-k2")

    // Side coloring
    expect(screen.getByTestId("kill-attacker-k1")).toHaveTextContent("s1lent")
    expect(screen.getByTestId("kill-attacker-k1").className).toContain(
      "text-orange-400",
    )
    expect(screen.getByTestId("kill-victim-k1").className).toContain(
      "text-sky-400",
    )

    // Headshot icon shows for k1, hides for k2
    expect(screen.getByTestId("kill-headshot-k1")).toBeInTheDocument()
    expect(screen.queryByTestId("kill-headshot-k2")).not.toBeInTheDocument()

    // Weapon icon resolved to /equipment SVG
    const weaponIcons = screen.getAllByTestId("weapon-icon-AK-47")
    expect(weaponIcons.length).toBeGreaterThan(0)
    expect(weaponIcons[0]).toHaveAttribute("src", "/equipment/ak47.svg")
  })

  it("hides kills that haven't happened yet at currentTick", async () => {
    mockAppBindings.GetEventsByTypes.mockResolvedValueOnce([
      makeKill({
        id: "future",
        tick: 64 * 100,
        attackerName: "x",
        attackerSide: "T",
        victimName: "y",
        victimSide: "CT",
      }),
    ])
    useViewerStore.getState().setDemoId(DEMO_ID)
    useViewerStore.getState().setTick(64 * 50)

    renderWithProviders(<KillLog />)
    await waitFor(() => {
      expect(mockAppBindings.GetEventsByTypes).toHaveBeenCalled()
    })
    expect(screen.queryByTestId("kill-log")).not.toBeInTheDocument()
  })

  it("updates rolling kill list as currentTick advances", async () => {
    mockAppBindings.GetEventsByTypes.mockResolvedValueOnce([
      makeKill({
        id: "k1",
        tick: 100,
        attackerName: "alpha",
        attackerSide: "T",
        victimName: "bravo",
        victimSide: "CT",
      }),
      makeKill({
        id: "k2",
        tick: 200,
        attackerName: "charlie",
        attackerSide: "T",
        victimName: "delta",
        victimSide: "CT",
      }),
    ])
    useViewerStore.getState().setDemoId(DEMO_ID)
    useViewerStore.getState().setTick(150)

    renderWithProviders(<KillLog />)

    await waitFor(() => {
      expect(screen.queryByTestId("kill-log-row-k1")).toBeInTheDocument()
    })
    expect(screen.queryByTestId("kill-log-row-k2")).not.toBeInTheDocument()

    act(() => useViewerStore.getState().setTick(250))
    expect(screen.getByTestId("kill-log-row-k2")).toBeInTheDocument()
  })
})
