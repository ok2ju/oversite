import { describe, it, expect } from "vitest"
import { buildLanes, MIN_GAP_PX } from "./build-lanes"
import type { FilterSet } from "./types"
import type { GameEvent, GameEventType } from "@/types/demo"
import type { Round } from "@/types/round"
import type { main } from "@wailsjs/go/models"

const ALL_ON: FilterSet = {
  kills: true,
  utility: true,
  bomb: true,
  myEvents: false,
}

const ROUND: Round = {
  id: "r1",
  round_number: 3,
  start_tick: 1000,
  freeze_end_tick: 1100,
  end_tick: 2000,
  winner_side: "CT",
  win_reason: "elim",
  ct_score: 1,
  t_score: 1,
  is_overtime: false,
  ct_team_name: "CT",
  t_team_name: "T",
}

let eventCounter = 0

function mkEvent(opts: {
  event_type: GameEventType
  tick: number
  attacker_team?: string
  victim_team?: string
  attacker_steam_id?: string | null
  victim_steam_id?: string | null
  weapon?: string | null
  headshot?: boolean
  extra_data?: Record<string, unknown> | null
}): GameEvent {
  eventCounter += 1
  return {
    id: `e-${eventCounter}`,
    demo_id: "d1",
    round_id: "r1",
    tick: opts.tick,
    event_type: opts.event_type,
    attacker_steam_id: opts.attacker_steam_id ?? null,
    victim_steam_id: opts.victim_steam_id ?? null,
    weapon: opts.weapon ?? null,
    x: null,
    y: null,
    z: null,
    headshot: opts.headshot ?? false,
    assister_steam_id: null,
    health_damage: 0,
    attacker_name: "",
    victim_name: "",
    attacker_team: opts.attacker_team ?? "",
    victim_team: opts.victim_team ?? "",
    extra_data: opts.extra_data ?? null,
  }
}

function mkContact(
  overrides: Partial<main.ContactMoment> = {},
): main.ContactMoment {
  return {
    id: 1,
    demo_id: 100,
    round_id: 1000,
    round_number: ROUND.round_number,
    subject_steam: "player-1",
    t_first: 1500,
    t_last: 1600,
    t_pre: 1450,
    t_post: 1700,
    enemies: ["enemy-1"],
    outcome: "won_clean",
    signal_count: 3,
    extras: {},
    mistakes: [],
    ...overrides,
  } as unknown as main.ContactMoment
}

function mkContactMistake(
  overrides: Partial<main.ContactMistake> = {},
): main.ContactMistake {
  return {
    kind: "slow_reaction",
    category: "aim",
    severity: 2,
    phase: "pre",
    tick: 1480,
    extras: {},
    ...overrides,
  } as unknown as main.ContactMistake
}

describe("buildLanes — round-window filter", () => {
  it("drops events outside the round's tick range", () => {
    const events = [
      mkEvent({
        event_type: "kill",
        tick: 500,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
      }), // before
      mkEvent({
        event_type: "kill",
        tick: 1500,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
      }), // in
      mkEvent({
        event_type: "kill",
        tick: 2500,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
      }), // after
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.topLane).toHaveLength(1)
    expect(model.topLane[0].tick).toBe(1500)
  })
})

describe("buildLanes — team mode CT/T split", () => {
  it("routes CT grenades to the top lane and T grenades to the bottom lane", () => {
    // Round-mode lanes are grenades + bomb only after Phase 4.
    const events = [
      mkEvent({
        event_type: "grenade_throw",
        tick: 1200,
        attacker_team: "CT",
        weapon: "smokegrenade",
        extra_data: { entity_id: 11 },
      }),
      mkEvent({
        event_type: "grenade_throw",
        tick: 1300,
        attacker_team: "T",
        weapon: "smokegrenade",
        extra_data: { entity_id: 12 },
      }),
      mkEvent({
        event_type: "grenade_throw",
        tick: 1400,
        attacker_team: "CT",
        weapon: "smokegrenade",
        extra_data: { entity_id: 13 },
      }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.topLane.flatMap((c) => c.events)).toHaveLength(2)
    expect(model.bottomLane.flatMap((c) => c.events)).toHaveLength(1)
    expect(model.bottomLane[0].events[0].tick).toBe(1300)
  })

  it("places bomb plants on the T lane and defuses on the CT lane", () => {
    const events = [
      mkEvent({ event_type: "bomb_plant", tick: 1300, attacker_team: "T" }),
      mkEvent({ event_type: "bomb_defuse", tick: 1700, attacker_team: "CT" }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.topLane[0].events[0].kind).toBe("bomb_defuse")
    expect(model.bottomLane[0].events[0].kind).toBe("bomb_plant")
  })

  it("ignores events with no team affiliation", () => {
    const events = [
      mkEvent({ event_type: "kill", tick: 1300, attacker_team: "" }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.topLane).toHaveLength(0)
    expect(model.bottomLane).toHaveLength(0)
  })
})

describe("buildLanes — player mode caused/affected split", () => {
  it("places attacker-side events on the top (caused) lane and victim-side on bottom (affected)", () => {
    const me = "player-1"
    const other = "player-2"
    const events = [
      mkEvent({
        event_type: "kill",
        tick: 1200,
        attacker_steam_id: me,
        victim_steam_id: other,
        attacker_team: "CT",
        victim_team: "T",
      }),
      mkEvent({
        event_type: "kill",
        tick: 1500,
        attacker_steam_id: other,
        victim_steam_id: me,
        attacker_team: "T",
        victim_team: "CT",
      }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: me,
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.topLane.flatMap((c) => c.events)).toHaveLength(1)
    expect(model.topLane[0].events[0].tick).toBe(1200)
    expect(model.bottomLane.flatMap((c) => c.events)).toHaveLength(1)
    expect(model.bottomLane[0].events[0].tick).toBe(1500)
  })

  it("excludes events the player isn't part of", () => {
    const events = [
      mkEvent({
        event_type: "kill",
        tick: 1500,
        attacker_steam_id: "player-2",
        victim_steam_id: "player-3",
        attacker_team: "T",
        victim_team: "CT",
      }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.topLane).toHaveLength(0)
    expect(model.bottomLane).toHaveLength(0)
  })

  it("routes player_flashed to the affected lane when player is the victim", () => {
    const events = [
      mkEvent({
        event_type: "player_flashed",
        tick: 1200,
        attacker_steam_id: "player-2",
        victim_steam_id: "player-1",
      }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.bottomLane[0].events[0].kind).toBe("player_flashed")
  })
})

describe("buildLanes — grenade entity correlation", () => {
  it("folds throw + detonate for one entity into a single marker", () => {
    const events = [
      mkEvent({
        event_type: "grenade_throw",
        tick: 1200,
        attacker_team: "CT",
        weapon: "Smoke Grenade",
        extra_data: { entity_id: 42 },
      }),
      mkEvent({
        event_type: "grenade_bounce",
        tick: 1230,
        attacker_team: "CT",
        weapon: "Smoke Grenade",
        extra_data: { entity_id: 42 },
      }),
      mkEvent({
        event_type: "grenade_detonate",
        tick: 1260,
        attacker_team: "CT",
        weapon: "Smoke Grenade",
        extra_data: { entity_id: 42 },
      }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    const all = model.topLane.flatMap((c) => c.events)
    expect(all).toHaveLength(1)
    expect(all[0].kind).toBe("grenade")
    expect(all[0].tick).toBe(1200)
    expect(all[0].detonateTick).toBe(1260)
  })

  it("emits separate markers per entity_id", () => {
    const events = [
      mkEvent({
        event_type: "grenade_throw",
        tick: 1200,
        attacker_team: "CT",
        weapon: "Flashbang",
        extra_data: { entity_id: 1 },
      }),
      mkEvent({
        event_type: "grenade_throw",
        tick: 1400,
        attacker_team: "CT",
        weapon: "Flashbang",
        extra_data: { entity_id: 2 },
      }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    const all = model.topLane.flatMap((c) => c.events)
    expect(all).toHaveLength(2)
  })
})

describe("buildLanes — clustering", () => {
  it("groups events within MIN_GAP_PX into a single cluster", () => {
    // round span = 1000 ticks across 1000 px → 1 tick / px.
    // Events 5 ticks apart are well within MIN_GAP_PX = 12.
    const events = [
      mkEvent({
        event_type: "kill",
        tick: 1300,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
      }),
      mkEvent({
        event_type: "kill",
        tick: 1305,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
      }),
      mkEvent({
        event_type: "kill",
        tick: 1310,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
      }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.topLane).toHaveLength(1)
    expect(model.topLane[0].events).toHaveLength(3)
  })

  it("splits events spaced beyond MIN_GAP_PX into separate clusters", () => {
    const events = [
      mkEvent({
        event_type: "kill",
        tick: 1100,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
      }),
      mkEvent({
        event_type: "kill",
        tick: 1100 + MIN_GAP_PX + 5,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
      }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.topLane).toHaveLength(2)
  })
})

describe("buildLanes — filter chips", () => {
  const events = [
    mkEvent({ event_type: "kill", tick: 1200, attacker_team: "CT" }),
    mkEvent({
      event_type: "grenade_throw",
      tick: 1300,
      attacker_team: "CT",
      weapon: "Flashbang",
      extra_data: { entity_id: 7 },
    }),
    mkEvent({ event_type: "bomb_plant", tick: 1400, attacker_team: "T" }),
  ]

  it("kills filter hides kill events", () => {
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: { ...ALL_ON, kills: false },
      laneWidthPx: 1000,
    })
    const allKinds = [
      ...model.topLane.flatMap((c) => c.events.map((e) => e.kind)),
      ...model.bottomLane.flatMap((c) => c.events.map((e) => e.kind)),
    ]
    expect(allKinds).not.toContain("kill")
  })

  it("utility filter hides grenades", () => {
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: { ...ALL_ON, utility: false },
      laneWidthPx: 1000,
    })
    const allKinds = [
      ...model.topLane.flatMap((c) => c.events.map((e) => e.kind)),
      ...model.bottomLane.flatMap((c) => c.events.map((e) => e.kind)),
    ]
    expect(allKinds).not.toContain("grenade")
  })

  it("bomb filter hides plants and defuses", () => {
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: { ...ALL_ON, bomb: false },
      laneWidthPx: 1000,
    })
    const allKinds = [
      ...model.topLane.flatMap((c) => c.events.map((e) => e.kind)),
      ...model.bottomLane.flatMap((c) => c.events.map((e) => e.kind)),
    ]
    expect(allKinds).not.toContain("bomb_plant")
    expect(allKinds).not.toContain("bomb_defuse")
  })

  it("myEvents narrows to events involving the selected player", () => {
    const me = "player-1"
    const playerEvents = [
      mkEvent({
        event_type: "kill",
        tick: 1200,
        attacker_steam_id: me,
        victim_steam_id: "p2",
        attacker_team: "CT",
        victim_team: "T",
      }),
      mkEvent({
        event_type: "kill",
        tick: 1300,
        attacker_steam_id: "p3",
        victim_steam_id: "p4",
        attacker_team: "CT",
        victim_team: "T",
      }),
    ]
    const model = buildLanes({
      events: playerEvents,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: me,
      filters: { ...ALL_ON, myEvents: true },
      laneWidthPx: 1000,
    })
    const all = [
      ...model.topLane.flatMap((c) => c.events),
      ...model.bottomLane.flatMap((c) => c.events),
    ]
    expect(all).toHaveLength(1)
    expect(all[0].source.attacker_steam_id).toBe(me)
  })
})

describe("buildLanes — spine geometry", () => {
  it("computes the live range from the freeze-end tick to round end", () => {
    const model = buildLanes({
      events: [],
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.spine.live).toEqual({ startTick: 1100, endTick: 2000 })
    expect(model.spine.bombBar).toBeNull()
    expect(model.spine.postPlant).toBeNull()
  })

  it("uses the live-phase tick as the lane window's start", () => {
    const events = [
      mkEvent({
        event_type: "kill",
        tick: 1050,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
      }), // freeze
      mkEvent({
        event_type: "kill",
        tick: 1500,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
      }), // live
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.roundStartTick).toBe(1100)
    expect(model.topLane).toHaveLength(1)
    expect(model.topLane[0].tick).toBe(1500)
  })

  it("draws the bomb bar from plant to defuse when defused", () => {
    const events = [
      mkEvent({ event_type: "bomb_plant", tick: 1500, attacker_team: "T" }),
      mkEvent({ event_type: "bomb_defuse", tick: 1900, attacker_team: "CT" }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.spine.bombBar).toEqual({ startTick: 1500, endTick: 1900 })
    expect(model.spine.postPlant).toEqual({ startTick: 1500, endTick: 1900 })
  })

  it("draws the bomb bar from plant to explode when not defused", () => {
    const events = [
      mkEvent({ event_type: "bomb_plant", tick: 1500, attacker_team: "T" }),
      mkEvent({ event_type: "bomb_explode", tick: 1950, attacker_team: "T" }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 1000,
    })
    expect(model.spine.bombBar).toEqual({ startTick: 1500, endTick: 1950 })
  })
})

describe("round-mode lane cleanup (Phase 4)", () => {
  it("drops kill events from the lanes when no player is selected", () => {
    const events = [
      mkEvent({ event_type: "kill", tick: 1200, attacker_team: "CT" }),
      mkEvent({ event_type: "kill", tick: 1300, attacker_team: "T" }),
      mkEvent({
        event_type: "grenade_throw",
        tick: 1400,
        attacker_team: "CT",
        weapon: "smokegrenade",
        extra_data: { entity_id: 1 },
      }),
      mkEvent({ event_type: "bomb_plant", tick: 1500, attacker_team: "T" }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 800,
    })
    const topKinds = model.topLane.flatMap((c) => c.events.map((e) => e.kind))
    const bottomKinds = model.bottomLane.flatMap((c) =>
      c.events.map((e) => e.kind),
    )
    expect(topKinds).not.toContain("kill")
    expect(bottomKinds).not.toContain("kill")
    // Grenades and bomb still show up.
    const allKinds = [...topKinds, ...bottomKinds]
    expect(allKinds).toContain("grenade")
    expect(allKinds).toContain("bomb_plant")
  })

  it("drops player_hurt / player_flashed in round mode", () => {
    const events = [
      mkEvent({
        event_type: "player_hurt",
        tick: 1200,
        attacker_team: "T",
        victim_team: "CT",
      }),
      mkEvent({
        event_type: "player_flashed",
        tick: 1300,
        attacker_team: "T",
        victim_team: "CT",
      }),
      mkEvent({
        event_type: "grenade_throw",
        tick: 1400,
        attacker_team: "CT",
        weapon: "smokegrenade",
        extra_data: { entity_id: 2 },
      }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 800,
    })
    const allKinds = [
      ...model.topLane.flatMap((c) => c.events.map((e) => e.kind)),
      ...model.bottomLane.flatMap((c) => c.events.map((e) => e.kind)),
    ]
    expect(allKinds).not.toContain("player_hurt")
    expect(allKinds).not.toContain("player_flashed")
  })

  it("keeps kill events in player mode (caused/affected lanes)", () => {
    const events = [
      mkEvent({
        event_type: "kill",
        tick: 1200,
        attacker_team: "CT",
        attacker_steam_id: "player-1",
        victim_team: "T",
        victim_steam_id: "enemy-1",
      }),
    ]
    const model = buildLanes({
      events,
      contacts: [],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 800,
    })
    const kinds = model.topLane.flatMap((c) => c.events.map((e) => e.kind))
    expect(kinds).toContain("kill")
  })
})

describe("contacts lane (Phase 4)", () => {
  it("projects one marker per contact at t_first", () => {
    const model = buildLanes({
      events: [],
      contacts: [
        mkContact({ id: 1, t_first: 1500 }),
        mkContact({ id: 2, t_first: 1700 }),
      ],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 800,
    })
    expect(model.contacts).toHaveLength(2)
    expect(model.contacts.map((c) => c.tFirst)).toEqual([1500, 1700])
  })

  it("returns empty contacts in round mode regardless of input", () => {
    const model = buildLanes({
      events: [],
      contacts: [mkContact()],
      round: ROUND,
      selectedPlayerSteamId: null,
      filters: ALL_ON,
      laneWidthPx: 800,
    })
    expect(model.contacts).toEqual([])
  })

  it("filters out contacts whose t_first is outside the round window", () => {
    const model = buildLanes({
      events: [],
      contacts: [
        mkContact({ id: 1, t_first: 500 }), // before round (start=1100)
        mkContact({ id: 2, t_first: 1500 }), // in round
        mkContact({ id: 3, t_first: 6000 }), // after round (end=2000)
      ],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 800,
    })
    expect(model.contacts).toHaveLength(1)
    expect(model.contacts[0].tFirst).toBe(1500)
  })

  it("computes worstSeverity from the contact's mistakes", () => {
    const c = mkContact({
      mistakes: [
        mkContactMistake({ kind: "slow_reaction", severity: 2, phase: "pre" }),
        mkContactMistake({
          kind: "isolated_peek",
          category: "positioning",
          severity: 3,
          phase: "pre",
        }),
        mkContactMistake({
          kind: "shot_while_moving",
          category: "movement",
          severity: 2,
          phase: "during",
        }),
      ],
    })
    const model = buildLanes({
      events: [],
      contacts: [c],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 800,
    })
    expect(model.contacts[0].worstSeverity).toBe(3)
  })

  it("worstSeverity is 0 for a contact with no mistakes (clean win)", () => {
    const c = mkContact({ mistakes: [] })
    const model = buildLanes({
      events: [],
      contacts: [c],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 800,
    })
    expect(model.contacts[0].worstSeverity).toBe(0)
  })

  it("sorts contacts by worstSeverity ascending so the worst renders last", () => {
    const c1 = mkContact({ id: 1, t_first: 1500, mistakes: [] })
    const c2 = mkContact({
      id: 2,
      t_first: 1600,
      mistakes: [
        mkContactMistake({
          kind: "isolated_peek",
          category: "positioning",
          severity: 3,
          phase: "pre",
          tick: undefined,
        }),
      ],
    })
    const c3 = mkContact({
      id: 3,
      t_first: 1700,
      mistakes: [
        mkContactMistake({
          kind: "slow_reaction",
          severity: 2,
          phase: "pre",
          tick: undefined,
        }),
      ],
    })
    const model = buildLanes({
      events: [],
      contacts: [c1, c2, c3],
      round: ROUND,
      selectedPlayerSteamId: "player-1",
      filters: ALL_ON,
      laneWidthPx: 800,
    })
    expect(model.contacts.map((c) => c.id)).toEqual([1, 3, 2])
  })
})
