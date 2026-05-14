import { describe, it, expect, vi, beforeEach } from "vitest"

// --- Mock PlayerSprite ---

type MockPlayerSprite = {
  container: { x: number; y: number }
  update: ReturnType<typeof vi.fn>
  setClickHandler: ReturnType<typeof vi.fn>
  destroy: ReturnType<typeof vi.fn>
}

const { mockSpriteInstances, createMockSprite } = vi.hoisted(() => {
  const mockSpriteInstances: MockPlayerSprite[] = []
  const createMockSprite = (): MockPlayerSprite => {
    const instance: MockPlayerSprite = {
      container: { x: 0, y: 0 },
      update: vi.fn(),
      setClickHandler: vi.fn(),
      destroy: vi.fn(),
    }
    mockSpriteInstances.push(instance)
    return instance
  }
  return { mockSpriteInstances, createMockSprite }
})

vi.mock("../sprites/player", () => ({
  PlayerSprite: vi.fn().mockImplementation(function () {
    return createMockSprite()
  }),
}))

vi.mock("@/lib/maps/calibration", () => ({
  worldToPixel: vi.fn(
    (
      world: { x: number; y: number },
      calibration: { originX: number; originY: number; scale: number },
    ) => ({
      x: (world.x - calibration.originX) / calibration.scale,
      y: (calibration.originY - world.y) / calibration.scale,
    }),
  ),
  worldToPixelInto: vi.fn(
    (
      out: { x: number; y: number },
      worldX: number,
      worldY: number,
      calibration: { originX: number; originY: number; scale: number },
    ) => {
      out.x = (worldX - calibration.originX) / calibration.scale
      out.y = (calibration.originY - worldY) / calibration.scale
      return out
    },
  ),
}))

import { PlayerLayer } from "./player-layer"
import type { TickData } from "@/types/demo"
import type { PlayerRosterEntry } from "@/types/roster"
import type { MapCalibration } from "@/lib/maps/calibration"

function createMockContainer() {
  return {
    addChild: vi.fn(),
    removeChild: vi.fn(),
  }
}

const testCalibration: MapCalibration = {
  originX: -2476,
  originY: 3239,
  scale: 4.4,
  width: 1024,
  height: 1024,
}

function makeTickData(overrides: Partial<TickData> = {}): TickData {
  return {
    tick: 100,
    steam_id: "76561198000000001",
    x: -672,
    y: -672,
    z: 0,
    yaw: 0,
    health: 100,
    armor: 100,
    is_alive: true,
    weapon: null,
    money: 0,
    has_helmet: false,
    has_defuser: false,
    ammo_clip: 0,
    ammo_reserve: 0,
    inventory: [],
    ...overrides,
  }
}

function makeRosterEntry(
  overrides: Partial<PlayerRosterEntry> = {},
): PlayerRosterEntry {
  return {
    steam_id: "76561198000000001",
    player_name: "player1",
    team_side: "CT",
    ...overrides,
  }
}

describe("PlayerLayer", () => {
  let container: ReturnType<typeof createMockContainer>
  let layer: PlayerLayer

  // Most tests only care about the single-sample path; helper keeps call sites
  // readable. Interpolation-specific behaviour has its own describe block below.
  function updateSingle(
    ticks: TickData[],
    calibration: MapCalibration,
    selected: string | null,
  ): void {
    layer.update(ticks, null, 0, calibration, selected)
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockSpriteInstances.length = 0
    container = createMockContainer()
    layer = new PlayerLayer(container as never)
  })

  describe("update()", () => {
    it("creates a PlayerSprite for each player in tick data", () => {
      const ticks = [
        makeTickData({ steam_id: "player1" }),
        makeTickData({ steam_id: "player2" }),
      ]
      updateSingle(ticks, testCalibration, null)
      expect(mockSpriteInstances).toHaveLength(2)
    })

    it("adds new sprite container to layer container", () => {
      layer.update([makeTickData()], null, 0, testCalibration, null)
      expect(container.addChild).toHaveBeenCalledWith(
        mockSpriteInstances[0].container,
      )
    })

    it("calls update on each sprite with correct data", () => {
      const tick = makeTickData({
        steam_id: "player1",
        is_alive: true,
        yaw: 90,
      })
      updateSingle([tick], testCalibration, null)
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({ isAlive: true, isSelected: false }),
      )
    })

    it("passes correct pixel coordinates from worldToPixel", () => {
      const tick = makeTickData({ steam_id: "player1", x: -672, y: -672 })
      updateSingle([tick], testCalibration, null)
      // worldToPixel(-672, -672) with de_dust2 calibration = (410, 889)
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({
          x: expect.closeTo(410, 0),
          y: expect.closeTo(889, 0),
        }),
      )
    })

    it("marks selected player as isSelected=true", () => {
      const tick = makeTickData({ steam_id: "player1" })
      updateSingle([tick], testCalibration, "player1")
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({ isSelected: true }),
      )
    })

    it("marks non-selected player as isSelected=false", () => {
      const tick = makeTickData({ steam_id: "player1" })
      updateSingle([tick], testCalibration, "player2")
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({ isSelected: false }),
      )
    })

    it("reuses existing sprite on subsequent ticks", () => {
      const tick = makeTickData({ steam_id: "player1" })
      updateSingle([tick], testCalibration, null)
      updateSingle([tick], testCalibration, null)
      expect(mockSpriteInstances).toHaveLength(1)
      expect(mockSpriteInstances[0].update).toHaveBeenCalledTimes(2)
    })

    it("removes sprite for player no longer in tick data", () => {
      const tick = makeTickData({ steam_id: "player1" })
      updateSingle([tick], testCalibration, null)
      updateSingle([], testCalibration, null)
      expect(container.removeChild).toHaveBeenCalledWith(
        mockSpriteInstances[0].container,
      )
      expect(mockSpriteInstances[0].destroy).toHaveBeenCalled()
    })

    it("sets click handler on new sprites when onPlayerClick is registered", () => {
      const cb = vi.fn()
      layer.onPlayerClick(cb)
      updateSingle(
        [makeTickData({ steam_id: "player1" })],
        testCalibration,
        null,
      )
      expect(mockSpriteInstances[0].setClickHandler).toHaveBeenCalledWith(
        cb,
        "player1",
      )
    })
  })

  describe("setRoster()", () => {
    it("provides team_side from roster to sprite update", () => {
      layer.setRoster([
        makeRosterEntry({ steam_id: "player1", team_side: "T" }),
      ])
      updateSingle(
        [makeTickData({ steam_id: "player1" })],
        testCalibration,
        null,
      )
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({ team: "T" }),
      )
    })

    it("provides player_name from roster to sprite update", () => {
      layer.setRoster([
        makeRosterEntry({ steam_id: "player1", player_name: "TestPlayer" }),
      ])
      updateSingle(
        [makeTickData({ steam_id: "player1" })],
        testCalibration,
        null,
      )
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({ name: "TestPlayer" }),
      )
    })

    it("falls back to truncated steamId when no roster entry", () => {
      updateSingle(
        [makeTickData({ steam_id: "76561198000000001" })],
        testCalibration,
        null,
      )
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({ name: "7656119800" }),
      )
    })

    it("falls back to CT side when no roster entry", () => {
      updateSingle(
        [makeTickData({ steam_id: "76561198000000001" })],
        testCalibration,
        null,
      )
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({ team: "CT" }),
      )
    })
  })

  describe("onPlayerClick()", () => {
    it("registers callback that is forwarded to new sprites", () => {
      const cb = vi.fn()
      layer.onPlayerClick(cb)
      updateSingle(
        [makeTickData({ steam_id: "player1" })],
        testCalibration,
        null,
      )
      expect(mockSpriteInstances[0].setClickHandler).toHaveBeenCalledWith(
        cb,
        "player1",
      )
    })
  })

  describe("clear()", () => {
    it("removes all sprites from container", () => {
      updateSingle(
        [makeTickData({ steam_id: "p1" }), makeTickData({ steam_id: "p2" })],
        testCalibration,
        null,
      )
      layer.clear()
      expect(container.removeChild).toHaveBeenCalledTimes(2)
      expect(mockSpriteInstances[0].destroy).toHaveBeenCalled()
      expect(mockSpriteInstances[1].destroy).toHaveBeenCalled()
    })

    it("is no-op when no sprites exist", () => {
      layer.clear()
      expect(container.removeChild).not.toHaveBeenCalled()
    })
  })

  describe("destroy()", () => {
    it("delegates to clear", () => {
      updateSingle([makeTickData({ steam_id: "p1" })], testCalibration, null)
      layer.destroy()
      expect(container.removeChild).toHaveBeenCalled()
      expect(mockSpriteInstances[0].destroy).toHaveBeenCalled()
    })
  })

  describe("interpolation", () => {
    it("lerps x/y linearly between current and next samples", () => {
      const cur = makeTickData({ steam_id: "p1", x: 0, y: 0 })
      const nxt = makeTickData({ steam_id: "p1", x: 100, y: 200 })
      layer.update([cur], [nxt], 0.25, testCalibration, null)
      // World position at alpha=0.25: (25, 50). Converted via mock worldToPixel.
      const { originX, originY, scale } = testCalibration
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({
          x: (25 - originX) / scale,
          y: (originY - 50) / scale,
        }),
      )
    })

    it("uses current sample when next is null", () => {
      const cur = makeTickData({ steam_id: "p1", x: 10, y: 20 })
      layer.update([cur], null, 0.5, testCalibration, null)
      const { originX, originY, scale } = testCalibration
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({
          x: (10 - originX) / scale,
          y: (originY - 20) / scale,
        }),
      )
    })

    it("uses current sample when alpha is 0", () => {
      const cur = makeTickData({ steam_id: "p1", x: 0, y: 0 })
      const nxt = makeTickData({ steam_id: "p1", x: 100, y: 100 })
      layer.update([cur], [nxt], 0, testCalibration, null)
      const { originX, originY, scale } = testCalibration
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({
          x: (0 - originX) / scale,
          y: (originY - 0) / scale,
        }),
      )
    })

    it("uses current sample for players missing from next", () => {
      const cur = makeTickData({ steam_id: "p1", x: 0, y: 0 })
      // next has a different steam_id
      const nxtOther = makeTickData({ steam_id: "p2", x: 999, y: 999 })
      layer.update([cur], [nxtOther], 0.5, testCalibration, null)
      const { originX, originY, scale } = testCalibration
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({
          x: (0 - originX) / scale,
          y: (originY - 0) / scale,
        }),
      )
    })

    it("lerps yaw via shortest arc across the 360 boundary", () => {
      const cur = makeTickData({ steam_id: "p1", yaw: 350 })
      const nxt = makeTickData({ steam_id: "p1", yaw: 10 })
      layer.update([cur], [nxt], 0.5, testCalibration, null)
      // Shortest arc: 350 → 10 is +20°. At t=0.5, yaw = 360 (== 0).
      expect(mockSpriteInstances[0].update).toHaveBeenCalledWith(
        expect.objectContaining({ yaw: 360 }),
      )
    })
  })
})
