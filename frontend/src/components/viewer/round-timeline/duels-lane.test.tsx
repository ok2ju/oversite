import { describe, it, expect, beforeEach, afterEach } from "vitest"
import { render, cleanup, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { TooltipProvider } from "@/components/ui/tooltip"
import { useViewerStore } from "@/stores/viewer"
import { DuelsLane } from "./duels-lane"
import type { DuelEntry } from "@/types/duel"
import type { MistakeEntry } from "@/types/mistake"

function mkDuel(opts: Partial<DuelEntry> & Pick<DuelEntry, "id">): DuelEntry {
  return {
    id: opts.id,
    round_number: opts.round_number ?? 1,
    attacker_steam: opts.attacker_steam ?? "STEAM_A",
    victim_steam: opts.victim_steam ?? "STEAM_B",
    start_tick: opts.start_tick ?? 100,
    end_tick: opts.end_tick ?? 200,
    outcome: opts.outcome ?? "won",
    end_reason: opts.end_reason ?? "kill",
    hit_confirmed: opts.hit_confirmed ?? true,
    hurt_count: opts.hurt_count ?? 1,
    shot_count: opts.shot_count ?? 4,
    mutual_duel_id: opts.mutual_duel_id ?? null,
  }
}

function mkMistake(
  opts: Partial<MistakeEntry> & Pick<MistakeEntry, "id" | "kind">,
): MistakeEntry {
  return {
    id: opts.id,
    kind: opts.kind,
    category: opts.category ?? "movement",
    severity: opts.severity ?? 2,
    title: opts.title ?? "Shot while moving",
    suggestion: "",
    why_it_hurts: "",
    round_number: opts.round_number ?? 1,
    tick: opts.tick ?? 100,
    steam_id: opts.steam_id ?? "STEAM_A",
    extras: null,
    duel_id: opts.duel_id ?? null,
  }
}

describe("DuelsLane", () => {
  beforeEach(() => {
    useViewerStore.getState().reset()
  })

  afterEach(() => {
    cleanup()
  })

  it("renders the placeholder when no player is selected", () => {
    render(
      <TooltipProvider>
        <DuelsLane
          duels={[]}
          mistakes={[]}
          roundStartTick={0}
          roundEndTick={1000}
          roundNumber={1}
          selectedPlayerSteamId={null}
          hasPlayer={false}
        />
      </TooltipProvider>,
    )
    expect(
      screen.getByTestId("round-timeline-duels-placeholder"),
    ).toHaveTextContent(/select a player/i)
  })

  it("renders one band per duel in the current round", () => {
    const duels = [
      mkDuel({ id: 1, round_number: 1, start_tick: 100, end_tick: 200 }),
      mkDuel({ id: 2, round_number: 2, start_tick: 5000, end_tick: 5100 }),
    ]
    render(
      <TooltipProvider>
        <DuelsLane
          duels={duels}
          mistakes={[]}
          roundStartTick={0}
          roundEndTick={1000}
          roundNumber={1}
          selectedPlayerSteamId="STEAM_A"
          hasPlayer
        />
      </TooltipProvider>,
    )
    expect(screen.getByTestId("duel-band-1")).toBeInTheDocument()
    expect(screen.queryByTestId("duel-band-2")).not.toBeInTheDocument()
  })

  it("seeks the playhead when a band is clicked", async () => {
    const user = userEvent.setup()
    useViewerStore.getState().initDemo({
      id: "1",
      mapName: "de_dust2",
      totalTicks: 100000,
      tickRate: 64,
    })
    const duels = [mkDuel({ id: 5, start_tick: 250, end_tick: 320 })]
    render(
      <TooltipProvider>
        <DuelsLane
          duels={duels}
          mistakes={[]}
          roundStartTick={0}
          roundEndTick={1000}
          roundNumber={1}
          selectedPlayerSteamId="STEAM_A"
          hasPlayer
        />
      </TooltipProvider>,
    )
    await user.click(screen.getByTestId("duel-band-5"))
    expect(useViewerStore.getState().currentTick).toBe(250)
  })

  it("shows severity dots for mistakes attributed to a duel", () => {
    const duels = [mkDuel({ id: 9, start_tick: 100, end_tick: 200 })]
    const mistakes = [
      mkMistake({
        id: 1,
        kind: "shot_while_moving",
        severity: 2,
        steam_id: "STEAM_A",
        duel_id: 9,
      }),
      mkMistake({
        id: 2,
        kind: "no_counter_strafe",
        severity: 3,
        steam_id: "STEAM_A",
        duel_id: 9,
      }),
    ]
    render(
      <TooltipProvider>
        <DuelsLane
          duels={duels}
          mistakes={mistakes}
          roundStartTick={0}
          roundEndTick={1000}
          roundNumber={1}
          selectedPlayerSteamId="STEAM_A"
          hasPlayer
        />
      </TooltipProvider>,
    )
    const band = screen.getByTestId("duel-band-9")
    // The band contains 2 severity dots + 1 outcome glyph = 3 spans with
    // aria-hidden. We assert at least 2 dot spans render.
    const dots = band.querySelectorAll("span.rounded-full")
    expect(dots.length).toBe(2)
  })
})
