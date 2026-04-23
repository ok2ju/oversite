import { describe, it, expect, vi } from "vitest"
import {
  PlaybackEngine,
  type PlaybackState,
  type RoundBoundary,
} from "./playback-engine"

function createEngine(overrides?: {
  state?: Partial<PlaybackState>
  tickRate?: number
}) {
  const state: PlaybackState = {
    currentTick: 0,
    totalTicks: 128000,
    isPlaying: true,
    speed: 1,
    ...overrides?.state,
  }
  const setTick = vi.fn((tick: number) => {
    state.currentTick = tick
  })
  const pause = vi.fn(() => {
    state.isPlaying = false
  })
  const getState = vi.fn(() => ({ ...state }))
  const engine = new PlaybackEngine({
    tickRate: overrides?.tickRate ?? 64,
    getState,
    setTick,
    pause,
  })
  return { engine, state, setTick, pause, getState }
}

describe("PlaybackEngine", () => {
  describe("tick advancement", () => {
    it("does not advance when paused", () => {
      const { engine, setTick } = createEngine({
        state: { isPlaying: false },
      })
      engine.update(16.67)
      expect(setTick).not.toHaveBeenCalled()
    })

    it("advances at correct rate at 1x speed (64 tick rate)", () => {
      const { engine, setTick } = createEngine()
      // 62.5ms at 64tr = exactly 4 ticks/frame. 16 frames = 64 ticks.
      for (let i = 0; i < 16; i++) engine.update(62.5)
      const lastCall = setTick.mock.calls[setTick.mock.calls.length - 1]
      expect(lastCall[0]).toBe(64)
    })

    it("advances at correct rate at 2x speed", () => {
      const { engine, setTick } = createEngine({ state: { speed: 2 } })
      // 62.5ms at 2x = 8 ticks/frame. 16 frames = 128 ticks.
      for (let i = 0; i < 16; i++) engine.update(62.5)
      const lastCall = setTick.mock.calls[setTick.mock.calls.length - 1]
      expect(lastCall[0]).toBe(128)
    })

    it("advances at correct rate at 0.5x speed", () => {
      const { engine, setTick } = createEngine({ state: { speed: 0.5 } })
      // 62.5ms at 0.5x = 2 ticks/frame. 16 frames = 32 ticks.
      for (let i = 0; i < 16; i++) engine.update(62.5)
      const lastCall = setTick.mock.calls[setTick.mock.calls.length - 1]
      expect(lastCall[0]).toBe(32)
    })

    it("advances at correct rate at 0.25x speed", () => {
      const { engine, setTick } = createEngine({ state: { speed: 0.25 } })
      // 62.5ms at 0.25x = 1 tick/frame. 16 frames = 16 ticks.
      for (let i = 0; i < 16; i++) engine.update(62.5)
      const lastCall = setTick.mock.calls[setTick.mock.calls.length - 1]
      expect(lastCall[0]).toBe(16)
    })

    it("advances at correct rate at 4x speed", () => {
      const { engine, setTick } = createEngine({ state: { speed: 4 } })
      // 62.5ms at 4x = 16 ticks/frame. 16 frames = 256 ticks.
      for (let i = 0; i < 16; i++) engine.update(62.5)
      const lastCall = setTick.mock.calls[setTick.mock.calls.length - 1]
      expect(lastCall[0]).toBe(256)
    })

    it("accumulates fractional ticks without drift", () => {
      const { engine, setTick } = createEngine()
      // 16 frames x 62.5ms = 1000ms = exactly 64 ticks
      for (let i = 0; i < 16; i++) {
        engine.update(62.5)
      }
      const lastCall = setTick.mock.calls[setTick.mock.calls.length - 1]
      expect(lastCall[0]).toBe(64)
    })

    it("only calls setTick when integer tick changes", () => {
      const { engine, setTick } = createEngine()
      // At 64tr, 1ms = 0.064 ticks, so first ~15 frames of 1ms won't cross integer
      engine.update(1) // 0.064 ticks, floor=0 same as previous 0
      expect(setTick).not.toHaveBeenCalled()
    })
  })

  describe("bounds", () => {
    it("stops and pauses at totalTicks", () => {
      const { engine, setTick, pause } = createEngine({
        state: { totalTicks: 64 },
      })
      // 62.5ms at 64tr = 4 ticks/frame. 17 frames = 68 ticks, but clamped to 64.
      for (let i = 0; i < 17; i++) engine.update(62.5)
      expect(setTick).toHaveBeenCalledWith(63)
      expect(pause).toHaveBeenCalled()
    })

    it("does not advance past totalTicks", () => {
      const { engine, setTick, pause } = createEngine({
        state: { totalTicks: 64, currentTick: 60 },
      })
      // Seek to 60 first so engine's internal state matches
      engine.seek(60)
      setTick.mockClear()
      pause.mockClear()

      // 62.5ms = 4 ticks, from 60 would go to 64 (totalTicks), clamped to 63
      engine.update(62.5)
      expect(setTick).toHaveBeenCalledWith(63)
      expect(pause).toHaveBeenCalled()
    })

    it("does not advance when already at totalTicks - 1", () => {
      const { engine, setTick } = createEngine({
        state: { currentTick: 127999, totalTicks: 128000 },
      })
      engine.seek(127999)
      setTick.mockClear()
      engine.update(16.67)
      // fractionalTick advances from 127999 by ~1.07 ticks, hitting >= totalTicks
      // Engine clamps to totalTicks - 1 = 127999, same as current, so setTick fires once
      expect(setTick).toHaveBeenCalledWith(127999)
    })
  })

  describe("deltaMS clamping", () => {
    it("clamps large deltaMS to 100ms", () => {
      const { engine, setTick } = createEngine()
      // 5000ms unclamped = 320 ticks, clamped 100ms = 6.4 ticks = 6
      engine.update(5000)
      const tick = setTick.mock.calls[setTick.mock.calls.length - 1][0]
      expect(tick).toBe(6) // floor(100ms / 1000 * 64) = floor(6.4) = 6
    })
  })

  describe("speed change", () => {
    it("has no tick discontinuity when speed changes mid-playback", () => {
      const { engine, state, setTick } = createEngine()
      // 8 frames x 62.5ms at 1x = 500ms = 32 ticks
      for (let i = 0; i < 8; i++) engine.update(62.5)
      const tickAfterFirstUpdate =
        setTick.mock.calls[setTick.mock.calls.length - 1][0]
      expect(tickAfterFirstUpdate).toBe(32)

      // Change speed to 2x
      state.speed = 2
      // 62.5ms at 2x = 8 ticks more
      engine.update(62.5)
      const tickAfterSecond =
        setTick.mock.calls[setTick.mock.calls.length - 1][0]
      // Should be 32 + 8 = 40
      expect(tickAfterSecond).toBe(40)
    })
  })

  describe("seek", () => {
    it("jumps to exact tick", () => {
      const { engine, setTick } = createEngine()
      engine.seek(5000)
      expect(setTick).toHaveBeenCalledWith(5000)
    })

    it("resets fractional accumulator", () => {
      const { engine } = createEngine()
      // Advance partially
      engine.update(10) // 0.64 ticks fractional
      engine.seek(5000)
      // After seek, interpolation factor should be 0
      expect(engine.interpolationFactor).toBe(0)
    })

    it("clamps to 0 for negative tick", () => {
      const { engine, setTick } = createEngine()
      engine.seek(-100)
      expect(setTick).toHaveBeenCalledWith(0)
    })

    it("clamps to totalTicks - 1", () => {
      const { engine, setTick } = createEngine({
        state: { totalTicks: 128000 },
      })
      engine.seek(200000)
      expect(setTick).toHaveBeenCalledWith(127999)
    })

    it("continues smoothly from new position after seek", () => {
      const { engine, setTick } = createEngine()
      engine.seek(1000)
      setTick.mockClear()

      // 1 second at 1x = 64 ticks from new position
      engine.update(100) // clamped to 100ms = 6.4 ticks
      const newTick = setTick.mock.calls[setTick.mock.calls.length - 1][0]
      expect(newTick).toBe(1006) // 1000 + floor(6.4)
    })
  })

  describe("interpolation factor", () => {
    it("returns fractional part of internal tick", () => {
      const { engine } = createEngine()
      // 10ms at 64tr = 0.64 ticks
      engine.update(10)
      expect(engine.interpolationFactor).toBeCloseTo(0.64, 5)
    })

    it("resets on seek", () => {
      const { engine } = createEngine()
      engine.update(10) // build up fractional part
      engine.seek(500)
      expect(engine.interpolationFactor).toBe(0)
    })
  })

  describe("round boundaries", () => {
    const boundaries: RoundBoundary[] = [
      { roundNumber: 1, startTick: 0, endTick: 5000 },
      { roundNumber: 2, startTick: 5001, endTick: 10000 },
      { roundNumber: 3, startTick: 10001, endTick: 15000 },
    ]

    it("auto-pauses when crossing endTick", () => {
      const { engine, setTick, pause } = createEngine({
        state: { currentTick: 4990 },
      })
      engine.seek(4990)
      setTick.mockClear()
      engine.setRoundBoundaries(boundaries)
      engine.setAutoPause(true)

      // At 64tr, 100ms = 6.4 ticks. From 4990, we'd reach 4996.4, crossing endTick 5000? No, 4996 < 5000.
      // Need more time: from 4990, need 10+ ticks. At 64tr: 10/64*1000 = 156ms, but clamped to 100ms = 6.4 ticks.
      // So we need multiple frames. Let's use a larger starting point.
      // Actually let's start closer:
      engine.seek(4995)
      setTick.mockClear()
      pause.mockClear()

      // 100ms at 64tr = 6.4 ticks, from 4995 goes to 5001.4 - crosses 5000
      engine.update(100)
      expect(setTick).toHaveBeenCalledWith(5000)
      expect(pause).toHaveBeenCalled()
    })

    it("does not pause when auto-pause is disabled", () => {
      const { engine, setTick, pause } = createEngine()
      engine.setRoundBoundaries(boundaries)
      engine.setAutoPause(false)
      engine.seek(4995)
      setTick.mockClear()
      pause.mockClear()

      engine.update(100) // crosses endTick 5000
      expect(pause).not.toHaveBeenCalled()
      // Should advance normally past the boundary
      const lastTick = setTick.mock.calls[setTick.mock.calls.length - 1][0]
      expect(lastTick).toBeGreaterThan(5000)
    })

    it("does not re-pause when resuming at boundary tick", () => {
      const { engine, state, setTick, pause } = createEngine()
      engine.setRoundBoundaries(boundaries)
      engine.setAutoPause(true)

      // Move to exact boundary
      engine.seek(5000)
      setTick.mockClear()
      pause.mockClear()

      // Resume playback from boundary
      state.isPlaying = true
      engine.update(100) // 6.4 ticks from 5000 = 5006.4
      // Should NOT pause at 5000 since we started there
      // But it should check next boundary (10000) which we haven't crossed
      const lastTick = setTick.mock.calls[setTick.mock.calls.length - 1][0]
      expect(lastTick).toBe(5006)
      expect(pause).not.toHaveBeenCalled()
    })

    it("snaps to exact endTick when crossing boundary", () => {
      const { engine, setTick } = createEngine()
      engine.setRoundBoundaries(boundaries)
      engine.setAutoPause(true)
      engine.seek(4998)
      setTick.mockClear()

      engine.update(100) // would go to 5004.4, but should snap to 5000
      expect(setTick).toHaveBeenCalledWith(5000)
    })
  })

  describe("freeze-time auto-skip", () => {
    // Round 1: start 0, freeze_end 960, end 3200
    // Round 2: start 3200, freeze_end 4160, end 6400
    const freezeWindows = [
      { startTick: 0, freezeEndTick: 960 },
      { startTick: 3200, freezeEndTick: 4160 },
    ]

    it("snaps fractionalTick to round 1 live start when rounds load", () => {
      const { engine, setTick } = createEngine()
      // Demo opens with seek(0); rounds arrive after.
      engine.seek(0)
      setTick.mockClear()

      engine.setFreezeWindows(freezeWindows)
      expect(setTick).toHaveBeenCalledWith(960)
      expect(engine.interpolationFactor).toBe(0)
    })

    it("does not snap if already past freeze into live", () => {
      const { engine, setTick } = createEngine()
      engine.seek(2000) // live phase of round 1
      setTick.mockClear()

      engine.setFreezeWindows(freezeWindows)
      expect(setTick).not.toHaveBeenCalled()
    })

    it("seek into a freeze window snaps to the round's live start", () => {
      const { engine, setTick } = createEngine()
      engine.setFreezeWindows(freezeWindows)
      setTick.mockClear()

      engine.seek(3300) // inside round 2 freeze window
      expect(setTick).toHaveBeenCalledWith(4160)
    })

    it("seek at the exact live boundary does not snap", () => {
      const { engine, setTick } = createEngine()
      engine.setFreezeWindows(freezeWindows)
      setTick.mockClear()

      engine.seek(960) // first live tick of round 1
      expect(setTick).toHaveBeenCalledWith(960)
    })

    it("playback crossing from round 1 live into round 2 freeze skips to round 2 live", () => {
      const { engine, setTick } = createEngine()
      engine.setFreezeWindows(freezeWindows)
      engine.seek(3100) // late in round 1 live
      setTick.mockClear()

      // 100ms at 64tr = 6.4 ticks → fractionalTick ~3106.4, not yet in freeze.
      engine.update(100)
      expect(setTick).toHaveBeenLastCalledWith(3106)

      // Advance multiple frames to cross into round 2 freeze (tick 3200).
      // From 3106 we need >94 ticks at 100ms/6.4-per-frame ≈ 15 frames.
      for (let i = 0; i < 20; i++) engine.update(100)

      // Must have snapped to 4160 (round 2 live start) at some point.
      const snappedCall = setTick.mock.calls.find((c) => c[0] === 4160)
      expect(snappedCall).toBeDefined()
    })

    it("seek before round 1 start jumps to round 1 live start", () => {
      const { engine, setTick } = createEngine()
      // Round 1 starts at 500, not 0 — pre-match gap at ticks [0, 499].
      engine.setFreezeWindows([
        { startTick: 500, freezeEndTick: 1460 },
        { startTick: 3200, freezeEndTick: 4160 },
      ])
      setTick.mockClear()

      engine.seek(100)
      expect(setTick).toHaveBeenCalledWith(1460)
    })

    it("filters out windows with freeze_end_tick <= start_tick", () => {
      const { engine, setTick } = createEngine()
      engine.setFreezeWindows([
        { startTick: 0, freezeEndTick: 0 }, // unknown freeze end; ignored
        { startTick: 3200, freezeEndTick: 4160 },
      ])
      setTick.mockClear()

      // 3300 is inside the remaining window → snap.
      engine.seek(3300)
      expect(setTick).toHaveBeenCalledWith(4160)
    })
  })

  describe("dispose", () => {
    it("can be called without error", () => {
      const { engine } = createEngine()
      expect(() => engine.dispose()).not.toThrow()
    })
  })
})
