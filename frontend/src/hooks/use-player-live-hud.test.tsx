import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { useViewerStore } from "@/stores/viewer"
import { useTickBufferStore } from "@/stores/tick-buffer"
import type { TickBuffer } from "@/lib/pixi/tick-buffer"
import type { TickData } from "@/types/demo"
import { usePlayerLiveHud } from "./use-player-live-hud"

function row(overrides: Partial<TickData>): TickData {
  return {
    tick: 0,
    steam_id: "STEAM_A",
    x: 0,
    y: 0,
    z: 0,
    yaw: 0,
    health: 100,
    armor: 100,
    is_alive: true,
    weapon: "ak47",
    money: 800,
    has_helmet: true,
    has_defuser: false,
    ammo_clip: 30,
    ammo_reserve: 90,
    inventory: [],
    ...overrides,
  }
}

// Synthetic TickBuffer stub that returns rows from a per-tick map.
function fakeBuffer(byTick: Map<number, TickData[]>): TickBuffer {
  return {
    getTickData: (tick: number) => byTick.get(tick) ?? null,
  } as unknown as TickBuffer
}

describe("usePlayerLiveHud", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    useViewerStore.getState().reset()
    useTickBufferStore.setState({ demoId: null, buffer: null })
  })

  afterEach(() => {
    vi.useRealTimers()
    useTickBufferStore.setState({ demoId: null, buffer: null })
  })

  it("returns null until demoId and steamId are present", () => {
    const { result } = renderHook(() => usePlayerLiveHud(null))
    expect(result.current).toBeNull()

    act(() => {
      vi.advanceTimersByTime(300)
    })
    expect(result.current).toBeNull()
  })

  it("publishes the current frame for the selected player", () => {
    const byTick = new Map<number, TickData[]>([
      [
        100,
        [row({ tick: 100, health: 87, armor: 50, money: 1500, x: 0, y: 0 })],
      ],
    ])

    useViewerStore.getState().initDemo({
      id: "demo-1",
      mapName: "de_dust2",
      totalTicks: 10000,
      tickRate: 64,
    })
    useTickBufferStore.setState({
      demoId: "demo-1",
      buffer: fakeBuffer(byTick),
    })
    useViewerStore.getState().setTick(100)

    const { result } = renderHook(() => usePlayerLiveHud("STEAM_A"))

    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(result.current?.tick).toBe(100)
    expect(result.current?.data.health).toBe(87)
    expect(result.current?.data.armor).toBe(50)
    expect(result.current?.data.money).toBe(1500)
  })

  it("computes a positive speed estimate from successive samples", () => {
    // Two samples 64 ticks apart at 64 tps → dt = 1s. dx = 130, dy = 0 →
    // speed = 130 u/s.
    const byTick = new Map<number, TickData[]>([
      [100, [row({ tick: 100, x: 0, y: 0 })]],
      [164, [row({ tick: 164, x: 130, y: 0 })]],
    ])

    useViewerStore.getState().initDemo({
      id: "demo-1",
      mapName: "de_dust2",
      totalTicks: 10000,
      tickRate: 64,
    })
    useTickBufferStore.setState({
      demoId: "demo-1",
      buffer: fakeBuffer(byTick),
    })
    useViewerStore.getState().setTick(100)

    const { result } = renderHook(() => usePlayerLiveHud("STEAM_A"))

    // First poll: only one sample, speed still null.
    act(() => {
      vi.advanceTimersByTime(300)
    })
    expect(result.current?.tick).toBe(100)
    expect(result.current?.speedUps).toBeNull()

    // Advance viewer tick — second poll picks up sample at 164 and derives
    // a speed of ~130 u/s.
    act(() => {
      useViewerStore.getState().setTick(164)
    })
    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(result.current?.tick).toBe(164)
    expect(result.current?.speedUps).toBeCloseTo(130, 1)
  })

  it("clears the speed window when the player dies", () => {
    const byTick = new Map<number, TickData[]>([
      [100, [row({ tick: 100, x: 0, y: 0 })]],
      [164, [row({ tick: 164, x: 130, y: 0, is_alive: false, health: 0 })]],
    ])

    useViewerStore.getState().initDemo({
      id: "demo-1",
      mapName: "de_dust2",
      totalTicks: 10000,
      tickRate: 64,
    })
    useTickBufferStore.setState({
      demoId: "demo-1",
      buffer: fakeBuffer(byTick),
    })
    useViewerStore.getState().setTick(100)

    const { result } = renderHook(() => usePlayerLiveHud("STEAM_A"))

    act(() => {
      vi.advanceTimersByTime(300)
    })
    act(() => {
      useViewerStore.getState().setTick(164)
    })
    act(() => {
      vi.advanceTimersByTime(300)
    })

    expect(result.current?.data.is_alive).toBe(false)
    expect(result.current?.speedUps).toBeNull()
  })
})
