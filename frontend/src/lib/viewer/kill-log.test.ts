import { describe, expect, it } from "vitest"
import type { GameEvent } from "@/types/demo"
import { selectVisibleKills } from "./kill-log"

function killEvent(opts: {
  id: string
  tick: number
  attacker?: string
  victim?: string
  weapon?: string
  attackerSide?: "CT" | "T"
  victimSide?: "CT" | "T"
  attackerName?: string
  victimName?: string
  headshot?: boolean
}): GameEvent {
  return {
    id: opts.id,
    demo_id: "demo-1",
    round_id: null,
    tick: opts.tick,
    event_type: "kill",
    attacker_steam_id: opts.attacker ?? "76561198000000001",
    victim_steam_id: opts.victim ?? "76561198000000002",
    weapon: opts.weapon ?? "AK-47",
    x: 0,
    y: 0,
    z: 0,
    headshot: opts.headshot ?? false,
    assister_steam_id: null,
    health_damage: 0,
    attacker_name: opts.attackerName ?? "Killer",
    attacker_team: opts.attackerSide ?? "T",
    victim_name: opts.victimName ?? "Victim",
    victim_team: opts.victimSide ?? "CT",
    extra_data: {},
  }
}

const TICK_RATE = 64

describe("selectVisibleKills", () => {
  it("returns empty for empty / undefined events", () => {
    expect(selectVisibleKills(undefined, 1000)).toEqual([])
    expect(selectVisibleKills([], 1000)).toEqual([])
  })

  it("returns kills inside the rolling window, oldest first (latest last)", () => {
    const events = [
      killEvent({ id: "k1", tick: 100 }),
      killEvent({ id: "k2", tick: 300 }),
      killEvent({ id: "k3", tick: 500 }),
    ]
    const visible = selectVisibleKills(events, 600, {
      windowSecs: 10,
      tickRate: TICK_RATE,
    })
    expect(visible.map((k) => k.id)).toEqual(["k1", "k2", "k3"])
  })

  it("excludes kills in the future relative to currentTick", () => {
    const events = [
      killEvent({ id: "past", tick: 100 }),
      killEvent({ id: "future", tick: 1000 }),
    ]
    const visible = selectVisibleKills(events, 500, { tickRate: TICK_RATE })
    expect(visible.map((k) => k.id)).toEqual(["past"])
  })

  it("excludes kills older than windowSecs", () => {
    const events = [
      killEvent({ id: "stale", tick: 0 }),
      killEvent({ id: "fresh", tick: 1000 }),
    ]
    // window of 5s @ 64 tickrate = 320 ticks; minTick = 1000 - 320 = 680
    const visible = selectVisibleKills(events, 1000, {
      windowSecs: 5,
      tickRate: TICK_RATE,
    })
    expect(visible.map((k) => k.id)).toEqual(["fresh"])
  })

  it("caps to the most recent maxEntries, oldest first", () => {
    const events = Array.from({ length: 8 }, (_, i) =>
      killEvent({ id: `k${i}`, tick: i * 10 }),
    )
    const visible = selectVisibleKills(events, 100, {
      windowSecs: 10,
      tickRate: TICK_RATE,
      maxEntries: 3,
    })
    expect(visible).toHaveLength(3)
    // Three newest kills (k5, k6, k7) ordered oldest → newest so the latest
    // ends up rendered last in the feed.
    expect(visible.map((k) => k.id)).toEqual(["k5", "k6", "k7"])
  })

  it("ignores non-kill events", () => {
    const events: GameEvent[] = [
      killEvent({ id: "k1", tick: 100 }),
      {
        id: "smoke",
        demo_id: "demo-1",
        round_id: null,
        tick: 110,
        event_type: "smoke_start",
        attacker_steam_id: "x",
        victim_steam_id: null,
        weapon: "Smoke Grenade",
        x: 0,
        y: 0,
        z: 0,
        headshot: false,
        assister_steam_id: null,
        health_damage: 0,
        attacker_name: "",
        attacker_team: "",
        victim_name: "",
        victim_team: "",
        extra_data: null,
      },
    ]
    const visible = selectVisibleKills(events, 200, { tickRate: TICK_RATE })
    expect(visible.map((k) => k.id)).toEqual(["k1"])
  })

  it("skips kills missing attacker or victim", () => {
    const events = [
      killEvent({ id: "ok", tick: 100 }),
      killEvent({ id: "no-attacker", tick: 110, attacker: "" }),
      killEvent({ id: "no-victim", tick: 120, victim: "" }),
    ]
    const visible = selectVisibleKills(events, 200, { tickRate: TICK_RATE })
    expect(visible.map((k) => k.id)).toEqual(["ok"])
  })

  it("extracts side and headshot info from extra_data", () => {
    const events = [
      killEvent({
        id: "k1",
        tick: 100,
        attackerName: "s1lent",
        victimName: "Sakamoto",
        attackerSide: "T",
        victimSide: "CT",
        headshot: true,
      }),
    ]
    const [k] = selectVisibleKills(events, 200, { tickRate: TICK_RATE })
    expect(k.attackerName).toBe("s1lent")
    expect(k.victimName).toBe("Sakamoto")
    expect(k.attackerSide).toBe("T")
    expect(k.victimSide).toBe("CT")
    expect(k.headshot).toBe(true)
  })

  it("falls back to default tickRate when not provided or invalid", () => {
    // 64 default; window 5s = 320 ticks
    const events = [
      killEvent({ id: "stale", tick: 0 }),
      killEvent({ id: "fresh", tick: 700 }),
    ]
    expect(
      selectVisibleKills(events, 1000, { windowSecs: 5 }).map((k) => k.id),
    ).toEqual(["fresh"])
    expect(
      selectVisibleKills(events, 1000, { windowSecs: 5, tickRate: 0 }).map(
        (k) => k.id,
      ),
    ).toEqual(["fresh"])
  })
})
