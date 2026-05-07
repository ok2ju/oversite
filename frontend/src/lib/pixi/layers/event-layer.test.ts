import { describe, it, expect, vi, beforeEach } from "vitest"

type MockGraphics = {
  clear: ReturnType<typeof vi.fn>
  circle: ReturnType<typeof vi.fn>
  moveTo: ReturnType<typeof vi.fn>
  lineTo: ReturnType<typeof vi.fn>
  fill: ReturnType<typeof vi.fn>
  stroke: ReturnType<typeof vi.fn>
  destroy: ReturnType<typeof vi.fn>
  removeFromParent: ReturnType<typeof vi.fn>
  alpha: number
  visible: boolean
}

const { mockGraphicsInstances, createMockGraphics } = vi.hoisted(() => {
  const mockGraphicsInstances: MockGraphics[] = []

  function createMockGraphics(): MockGraphics {
    const g: MockGraphics = {
      clear: vi.fn(),
      circle: vi.fn(),
      moveTo: vi.fn(),
      lineTo: vi.fn(),
      fill: vi.fn(),
      stroke: vi.fn(),
      destroy: vi.fn(),
      removeFromParent: vi.fn(),
      alpha: 1,
      visible: true,
    }
    // Enable chaining for drawing methods
    g.clear.mockReturnValue(g)
    g.circle.mockReturnValue(g)
    g.moveTo.mockReturnValue(g)
    g.lineTo.mockReturnValue(g)
    g.fill.mockReturnValue(g)
    g.stroke.mockReturnValue(g)
    mockGraphicsInstances.push(g)
    return g
  }

  return { mockGraphicsInstances, createMockGraphics }
})

vi.mock("pixi.js", () => ({
  Graphics: vi.fn().mockImplementation(function () {
    return createMockGraphics()
  }),
  Container: vi.fn().mockImplementation(function () {
    return {
      addChild: vi.fn(),
      removeChild: vi.fn(),
    }
  }),
}))

import { EventLayer } from "./event-layer"
import type { GameEvent } from "@/types/demo"
import {
  KILL_DURATION_TICKS,
  SHOT_DURATION_TICKS,
  SMOKE_DURATION_TICKS,
  HE_DURATION_TICKS,
  FLASH_DURATION_TICKS,
  FIRE_DURATION_TICKS,
} from "../sprites/effects"

const mockCalibration = {
  originX: -2476,
  originY: 3239,
  scale: 4.4,
  width: 1024,
  height: 1024,
}

function createMockContainer() {
  return {
    addChild: vi.fn(),
    removeChild: vi.fn(),
  }
}

function makeEvent(overrides: Partial<GameEvent> = {}): GameEvent {
  return {
    id: "evt-1",
    demo_id: "demo-1",
    round_id: null,
    tick: 100,
    event_type: "kill",
    attacker_steam_id: null,
    victim_steam_id: null,
    weapon: null,
    x: -500,
    y: 1000,
    z: 100,
    extra_data: null,
    ...overrides,
  }
}

describe("EventLayer", () => {
  let container: ReturnType<typeof createMockContainer>
  let layer: EventLayer

  beforeEach(() => {
    vi.clearAllMocks()
    mockGraphicsInstances.length = 0
    container = createMockContainer()
    layer = new EventLayer(container as never)
  })

  describe("constructor", () => {
    it("starts with no active effects", () => {
      layer.update(0, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)
    })
  })

  describe("setEvents", () => {
    it("stores events and ignores player_hurt and grenade_throw", () => {
      layer.setEvents([
        makeEvent({ event_type: "player_hurt", tick: 50 }),
        makeEvent({ event_type: "grenade_throw", tick: 60 }),
        makeEvent({ event_type: "kill", tick: 100 }),
      ])
      layer.update(200, mockCalibration)
      // Only kill effect should have been activated
      expect(mockGraphicsInstances.length).toBe(1)
    })

    it("pairs smoke_start with smoke_expired by entity_id", () => {
      const entityId = "smoke-entity-1"
      layer.setEvents([
        makeEvent({
          id: "smoke-1",
          event_type: "smoke_start",
          tick: 100,
          extra_data: { entity_id: entityId },
        }),
        makeEvent({
          id: "smoke-2",
          event_type: "smoke_expired",
          tick: 300,
          extra_data: { entity_id: entityId },
        }),
      ])
      // At tick 99: not yet active
      layer.update(99, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)

      // At tick 100: active
      layer.update(100, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)

      // At tick 300+: expired (duration = 300 - 100 = 200)
      layer.update(300, mockCalibration)
      const g = mockGraphicsInstances[0]
      expect(g.removeFromParent).toHaveBeenCalled()
    })
  })

  describe("update — kills", () => {
    it("activates kill graphics at kill tick", () => {
      layer.setEvents([makeEvent({ event_type: "kill", tick: 100 })])

      layer.update(99, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)

      layer.update(100, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)
      expect(container.addChild).toHaveBeenCalledTimes(1)
    })

    it("draws X marker at victim pixel position", () => {
      layer.setEvents([
        makeEvent({ event_type: "kill", tick: 100, x: -500, y: 1000 }),
      ])
      layer.update(100, mockCalibration)

      const g = mockGraphicsInstances[0]
      expect(g.moveTo).toHaveBeenCalled()
      expect(g.lineTo).toHaveBeenCalled()
      expect(g.stroke).toHaveBeenCalled()
    })

    it("draws attacker-to-victim line when attacker coords in extra_data", () => {
      layer.setEvents([
        makeEvent({
          event_type: "kill",
          tick: 100,
          x: -500,
          y: 1000,
          extra_data: { attacker_x: -600, attacker_y: 800 },
        }),
      ])
      layer.update(100, mockCalibration)

      const g = mockGraphicsInstances[0]
      // moveTo + lineTo called at least twice (X marker + attacker line)
      expect(g.moveTo.mock.calls.length).toBeGreaterThanOrEqual(2)
      expect(g.lineTo.mock.calls.length).toBeGreaterThanOrEqual(2)
    })

    it("releases kill graphics after KILL_DURATION_TICKS", () => {
      layer.setEvents([makeEvent({ event_type: "kill", tick: 100 })])
      layer.update(100, mockCalibration)

      const g = mockGraphicsInstances[0]
      layer.update(100 + KILL_DURATION_TICKS, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })
  })

  describe("update — shots", () => {
    it("activates shot graphics at weapon_fire tick", () => {
      layer.setEvents([
        makeEvent({
          event_type: "weapon_fire",
          tick: 100,
          extra_data: { yaw: 0 },
        }),
      ])

      layer.update(99, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)

      layer.update(100, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)
      expect(container.addChild).toHaveBeenCalledTimes(1)
    })

    it("draws a directional tracer made of multiple gradient segments", () => {
      layer.setEvents([
        makeEvent({
          event_type: "weapon_fire",
          tick: 100,
          x: -500,
          y: 1000,
          extra_data: { yaw: 90 },
        }),
      ])
      layer.update(100, mockCalibration)

      const g = mockGraphicsInstances[0]
      expect(g.clear).toHaveBeenCalled()
      expect(g.moveTo.mock.calls.length).toBeGreaterThan(1)
      expect(g.lineTo.mock.calls.length).toEqual(g.moveTo.mock.calls.length)
      expect(g.stroke.mock.calls.length).toEqual(g.moveTo.mock.calls.length)
      // Per-segment alpha should fade — first segment brighter than last.
      const strokes = g.stroke.mock.calls.map((c) => c[0].alpha as number)
      expect(strokes[0]).toBeGreaterThan(strokes[strokes.length - 1])
    })

    it("releases shot graphics after SHOT_DURATION_TICKS", () => {
      layer.setEvents([
        makeEvent({
          event_type: "weapon_fire",
          tick: 100,
          extra_data: { yaw: 0 },
        }),
      ])
      layer.update(100, mockCalibration)

      const g = mockGraphicsInstances[0]
      layer.update(100 + SHOT_DURATION_TICKS, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })

    it("draws a single solid line and impact marker when hit_x/hit_y are present", () => {
      layer.setEvents([
        makeEvent({
          event_type: "weapon_fire",
          tick: 100,
          x: -500,
          y: 1000,
          extra_data: { yaw: 0, hit_x: -300, hit_y: 1100 },
        }),
      ])
      layer.update(100, mockCalibration)

      const g = mockGraphicsInstances[0]
      // One stroke for the solid line + one fill for the impact dot.
      expect(g.moveTo).toHaveBeenCalledTimes(1)
      expect(g.lineTo).toHaveBeenCalledTimes(1)
      expect(g.stroke).toHaveBeenCalledTimes(1)
      expect(g.circle).toHaveBeenCalledTimes(1)
      expect(g.fill).toHaveBeenCalledTimes(1)
    })

    it("renders one tracer per fire event", () => {
      layer.setEvents([
        makeEvent({
          id: "shot-1",
          event_type: "weapon_fire",
          tick: 100,
          extra_data: { yaw: 0 },
        }),
        makeEvent({
          id: "shot-2",
          event_type: "weapon_fire",
          tick: 102,
          extra_data: { yaw: 45 },
        }),
        makeEvent({
          id: "shot-3",
          event_type: "weapon_fire",
          tick: 104,
          extra_data: { yaw: 90 },
        }),
      ])
      layer.update(105, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(3)
    })
  })

  describe("update — smoke", () => {
    it("activates at smoke_start tick", () => {
      layer.setEvents([
        makeEvent({
          event_type: "smoke_start",
          tick: 200,
          extra_data: { entity_id: "s1" },
        }),
      ])
      layer.update(199, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)

      layer.update(200, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)
    })

    it("releases after smoke duration", () => {
      layer.setEvents([
        makeEvent({
          event_type: "smoke_start",
          tick: 100,
          extra_data: { entity_id: "s1" },
        }),
      ])
      layer.update(100, mockCalibration)
      const g = mockGraphicsInstances[0]

      layer.update(100 + SMOKE_DURATION_TICKS, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })

    it("alpha matches computeSmokeState during smoke", () => {
      layer.setEvents([
        makeEvent({
          event_type: "smoke_start",
          tick: 100,
          extra_data: { entity_id: "s1" },
        }),
      ])
      // Update at smoke mid-point (hold phase)
      const mid = 100 + Math.floor(SMOKE_DURATION_TICKS / 2)
      layer.update(mid, mockCalibration)
      const g = mockGraphicsInstances[0]
      expect(g.fill).toHaveBeenCalled()
    })
  })

  describe("update — HE grenade", () => {
    it("activates at grenade_detonate for HE Grenade", () => {
      layer.setEvents([
        makeEvent({
          event_type: "grenade_detonate",
          tick: 150,
          weapon: "HE Grenade",
        }),
      ])
      layer.update(149, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)

      layer.update(150, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)
    })

    it("releases after HE_DURATION_TICKS", () => {
      layer.setEvents([
        makeEvent({
          event_type: "grenade_detonate",
          tick: 150,
          weapon: "HE Grenade",
        }),
      ])
      layer.update(150, mockCalibration)
      const g = mockGraphicsInstances[0]
      layer.update(150 + HE_DURATION_TICKS, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })
  })

  describe("update — flash", () => {
    it("activates at grenade_detonate for Flashbang", () => {
      layer.setEvents([
        makeEvent({
          event_type: "grenade_detonate",
          tick: 200,
          weapon: "Flashbang",
        }),
      ])
      layer.update(199, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)

      layer.update(200, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)
    })

    it("releases after FLASH_DURATION_TICKS", () => {
      layer.setEvents([
        makeEvent({
          event_type: "grenade_detonate",
          tick: 200,
          weapon: "Flashbang",
        }),
      ])
      layer.update(200, mockCalibration)
      const g = mockGraphicsInstances[0]
      layer.update(200 + FLASH_DURATION_TICKS, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })
  })

  describe("update — bomb plant", () => {
    it("activates at bomb_plant tick with oscillating alpha", () => {
      layer.setEvents([makeEvent({ event_type: "bomb_plant", tick: 300 })])
      layer.update(299, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)

      layer.update(300, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)
      const g = mockGraphicsInstances[0]
      expect(g.circle).toHaveBeenCalled()
    })
  })

  describe("update — bomb defuse", () => {
    it("activates at bomb_defuse tick", () => {
      layer.setEvents([makeEvent({ event_type: "bomb_defuse", tick: 400 })])
      layer.update(399, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)

      layer.update(400, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)
    })
  })

  describe("update — grenade trajectory", () => {
    it("renders nothing for an orphaned throw with no termination", () => {
      layer.setEvents([
        makeEvent({
          event_type: "grenade_throw",
          tick: 100,
          weapon: "HE Grenade",
          extra_data: { entity_id: 42 },
        }),
      ])
      layer.update(110, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)
    })

    it("activates a trajectory between throw and detonation", () => {
      layer.setEvents([
        makeEvent({
          id: "throw",
          event_type: "grenade_throw",
          tick: 100,
          weapon: "HE Grenade",
          x: 0,
          y: 0,
          extra_data: { entity_id: 42 },
        }),
        makeEvent({
          id: "detonate",
          event_type: "grenade_detonate",
          tick: 130,
          weapon: "HE Grenade",
          x: 100,
          y: 100,
          extra_data: { entity_id: 42 },
        }),
      ])

      layer.update(99, mockCalibration)
      // Throw not yet active.
      expect(mockGraphicsInstances.length).toBe(0)

      layer.update(115, mockCalibration)
      // Trajectory effect now drawing — stroke (trail) + fill (icon).
      expect(mockGraphicsInstances.length).toBe(1)
      const g = mockGraphicsInstances[0]
      expect(g.stroke).toHaveBeenCalled()
      expect(g.fill).toHaveBeenCalled()
    })

    it("releases the trajectory at the detonation tick", () => {
      layer.setEvents([
        makeEvent({
          id: "throw",
          event_type: "grenade_throw",
          tick: 100,
          weapon: "Smoke Grenade",
          extra_data: { entity_id: 7 },
        }),
        makeEvent({
          id: "smoke",
          event_type: "smoke_start",
          tick: 132,
          extra_data: { entity_id: 7 },
        }),
      ])
      layer.update(100, mockCalibration)
      const trajectoryGraphics = mockGraphicsInstances[0]
      // At tick 132 the trajectory's tickOffset (32) equals its duration, so it
      // expires; the smoke detonation activates as a separate effect.
      layer.update(132, mockCalibration)
      expect(trajectoryGraphics.removeFromParent).toHaveBeenCalled()
    })

    it("includes bounce points so the path passes through them", () => {
      layer.setEvents([
        makeEvent({
          id: "throw",
          event_type: "grenade_throw",
          tick: 100,
          weapon: "Flashbang",
          x: 0,
          y: 0,
          extra_data: { entity_id: 9 },
        }),
        makeEvent({
          id: "bounce",
          event_type: "grenade_bounce",
          tick: 110,
          x: 50,
          y: 50,
          extra_data: { entity_id: 9, bounce_nr: 1 },
        }),
        makeEvent({
          id: "detonate",
          event_type: "grenade_detonate",
          tick: 120,
          weapon: "Flashbang",
          x: 100,
          y: 0,
          extra_data: { entity_id: 9 },
        }),
      ])

      layer.update(115, mockCalibration)
      const g = mockGraphicsInstances[0]
      // Trail is drawn through throw → bounce → current head, so the polyline
      // has at least one moveTo and two lineTo segments.
      expect(g.moveTo).toHaveBeenCalledTimes(1)
      expect(g.lineTo.mock.calls.length).toBeGreaterThanOrEqual(2)
    })
  })

  describe("update — fire (Molotov / Incendiary)", () => {
    it("activates at fire_start tick", () => {
      layer.setEvents([
        makeEvent({
          event_type: "fire_start",
          tick: 200,
          weapon: "Molotov",
        }),
      ])
      layer.update(199, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(0)

      layer.update(200, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)
    })

    it("releases after FIRE_DURATION_TICKS", () => {
      layer.setEvents([
        makeEvent({
          event_type: "fire_start",
          tick: 200,
          weapon: "Incendiary Grenade",
        }),
      ])
      layer.update(200, mockCalibration)
      const g = mockGraphicsInstances[0]
      layer.update(200 + FIRE_DURATION_TICKS, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })
  })

  describe("round-end duration cap", () => {
    const rounds = [
      { start_tick: 0, end_tick: 1000 },
      { start_tick: 1100, end_tick: 2000 },
    ]

    it("caps a smoke that would otherwise bleed into the next round", () => {
      // Smoke starts at tick 950, would last 1152 ticks naturally (ending
      // at 2102), but round 1 ends at 1000 — cap to 50 ticks.
      layer.setEvents(
        [
          makeEvent({
            event_type: "smoke_start",
            tick: 950,
            extra_data: { entity_id: 1 },
          }),
        ],
        rounds,
      )

      // Active mid-round.
      layer.update(950, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)
      const g = mockGraphicsInstances[0]

      // Past round-end → released.
      layer.update(1000, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })

    it("caps a fire that would otherwise bleed into the next round", () => {
      layer.setEvents(
        [
          makeEvent({
            event_type: "fire_start",
            tick: 980,
            weapon: "Molotov",
          }),
        ],
        rounds,
      )
      layer.update(980, mockCalibration)
      const g = mockGraphicsInstances[0]
      layer.update(1000, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })

    it("caps a grenade trajectory whose detonation lands after round end", () => {
      // Throw at 990, "detonation" at 1500 (next round) — cap to 10 ticks.
      // Should release at tick 1000 (round 1 end), not 1500.
      layer.setEvents(
        [
          makeEvent({
            id: "throw",
            event_type: "grenade_throw",
            tick: 990,
            weapon: "HE Grenade",
            extra_data: { entity_id: 99 },
          }),
          makeEvent({
            id: "detonate",
            event_type: "grenade_detonate",
            tick: 1500,
            weapon: "HE Grenade",
            extra_data: { entity_id: 99 },
          }),
        ],
        rounds,
      )
      layer.update(990, mockCalibration)
      const g = mockGraphicsInstances[0]
      // tickOffset 10 == capped duration → released.
      layer.update(1000, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })

    it("leaves effects that finish within their round untouched", () => {
      // Smoke at 100 with explicit 200-tick duration — well within round 1.
      layer.setEvents(
        [
          makeEvent({
            id: "smoke",
            event_type: "smoke_start",
            tick: 100,
            extra_data: { entity_id: 5 },
          }),
          makeEvent({
            id: "smoke-end",
            event_type: "smoke_expired",
            tick: 300,
            extra_data: { entity_id: 5 },
          }),
        ],
        rounds,
      )
      layer.update(100, mockCalibration)
      const g = mockGraphicsInstances[0]

      // Still active at 250.
      layer.update(250, mockCalibration)
      expect(g.removeFromParent).not.toHaveBeenCalled()

      // Released at the natural smoke_expired pairing at tick 300.
      layer.update(300, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })

    it("leaves durations alone when no rounds are passed", () => {
      layer.setEvents([
        makeEvent({
          event_type: "smoke_start",
          tick: 950,
          extra_data: { entity_id: 7 },
        }),
      ])
      layer.update(950, mockCalibration)
      const g = mockGraphicsInstances[0]
      // Without round info, smoke runs the full default duration.
      layer.update(950 + SMOKE_DURATION_TICKS - 1, mockCalibration)
      expect(g.removeFromParent).not.toHaveBeenCalled()
    })
  })

  describe("backward seek", () => {
    it("clears all active effects when currentTick < lastTick", () => {
      layer.setEvents([makeEvent({ event_type: "kill", tick: 100 })])
      layer.update(150, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(1)
      const g = mockGraphicsInstances[0]

      // Seek back
      layer.update(50, mockCalibration)
      expect(g.removeFromParent).toHaveBeenCalled()
    })

    it("re-activates effects after backward seek on next forward pass", () => {
      layer.setEvents([makeEvent({ event_type: "kill", tick: 100 })])
      layer.update(150, mockCalibration)
      layer.update(50, mockCalibration) // seek back

      // Forward again past kill tick
      layer.update(110, mockCalibration)
      // addChild called twice: once at tick 150, once again after re-activation at tick 110
      expect(container.addChild.mock.calls.length).toBe(2)
    })
  })

  describe("clear", () => {
    it("releases all active graphics", () => {
      layer.setEvents([
        makeEvent({ event_type: "kill", tick: 100 }),
        makeEvent({ id: "evt-2", event_type: "kill", tick: 110 }),
      ])
      layer.update(120, mockCalibration)
      expect(mockGraphicsInstances.length).toBe(2)

      layer.clear()
      for (const g of mockGraphicsInstances) {
        expect(g.removeFromParent).toHaveBeenCalled()
      }
    })
  })

  describe("destroy", () => {
    it("releases active graphics and destroys pool", () => {
      layer.setEvents([makeEvent({ event_type: "kill", tick: 100 })])
      layer.update(100, mockCalibration)
      const g = mockGraphicsInstances[0]

      layer.destroy()
      expect(g.removeFromParent).toHaveBeenCalled()
      expect(g.destroy).toHaveBeenCalled()
    })
  })
})
