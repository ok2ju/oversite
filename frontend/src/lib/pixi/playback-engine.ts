export interface RoundBoundary {
  roundNumber: number
  startTick: number
  endTick: number
}

export interface FreezeWindow {
  startTick: number
  freezeEndTick: number
}

export interface PlaybackState {
  currentTick: number
  totalTicks: number
  isPlaying: boolean
  speed: number
}

export type GetStateFn = () => PlaybackState
export type SetTickFn = (tick: number) => void
export type PauseFn = () => void

interface PlaybackEngineOptions {
  tickRate: number
  getState: GetStateFn
  setTick: SetTickFn
  pause: PauseFn
}

const MAX_DELTA_MS = 100

export class PlaybackEngine {
  private readonly tickRate: number
  private readonly getState: GetStateFn
  private readonly setTick: SetTickFn
  private readonly pauseFn: PauseFn

  private fractionalTick = 0
  private roundBoundaries: RoundBoundary[] = []
  private autoPauseEnabled = false
  private freezeWindows: FreezeWindow[] = []

  constructor({ tickRate, getState, setTick, pause }: PlaybackEngineOptions) {
    this.tickRate = tickRate
    this.getState = getState
    this.setTick = setTick
    this.pauseFn = pause
  }

  update(deltaMS: number): void {
    const { isPlaying, totalTicks, speed } = this.getState()

    if (!isPlaying || this.fractionalTick >= totalTicks) return

    const clampedDelta = Math.min(deltaMS, MAX_DELTA_MS)
    const ticksToAdvance = (clampedDelta / 1000) * this.tickRate * speed

    const previousTick = Math.floor(this.fractionalTick)
    this.fractionalTick += ticksToAdvance

    // Check end of demo
    if (this.fractionalTick >= totalTicks) {
      this.fractionalTick = totalTicks - 1
      this.setTick(totalTicks - 1)
      this.pauseFn()
      return
    }

    // Auto-skip freeze time: if advancing crossed into a round's freeze window
    // (or the pre-match gap before round 1), jump to that round's live start.
    const liveTick = this.nextLiveTick(this.fractionalTick)
    if (liveTick !== this.fractionalTick) {
      this.fractionalTick = liveTick
      this.setTick(liveTick)
      return
    }

    // Check round boundary crossing
    if (this.autoPauseEnabled && this.roundBoundaries.length > 0) {
      for (const boundary of this.roundBoundaries) {
        if (
          previousTick < boundary.endTick &&
          this.fractionalTick >= boundary.endTick
        ) {
          this.fractionalTick = boundary.endTick
          this.setTick(boundary.endTick)
          this.pauseFn()
          return
        }
      }
    }

    const newIntegerTick = Math.floor(this.fractionalTick)
    if (newIntegerTick !== previousTick) {
      this.setTick(newIntegerTick)
    }
  }

  seek(tick: number): void {
    const { totalTicks } = this.getState()
    const clamped = Math.max(0, Math.min(tick, totalTicks - 1))
    const target = this.nextLiveTick(clamped)
    this.fractionalTick = target
    this.setTick(Math.floor(target))
  }

  setRoundBoundaries(boundaries: RoundBoundary[]): void {
    this.roundBoundaries = [...boundaries].sort((a, b) => a.endTick - b.endTick)
  }

  setAutoPause(enabled: boolean): void {
    this.autoPauseEnabled = enabled
  }

  // setFreezeWindows lets the engine auto-skip pre-round freeze time. Windows
  // come from the rounds query; rounds missing freeze_end_tick are ignored.
  // If the current position is inside a freeze window (typical on demo open,
  // where rounds arrive after the initial seek(0)), re-snap immediately.
  setFreezeWindows(windows: FreezeWindow[]): void {
    this.freezeWindows = windows
      .filter((w) => w.freezeEndTick > w.startTick)
      .sort((a, b) => a.startTick - b.startTick)

    const target = this.nextLiveTick(this.fractionalTick)
    if (target !== this.fractionalTick) {
      this.fractionalTick = target
      this.setTick(target)
    }
  }

  // nextLiveTick returns the next live-match tick at or after `tick`. A tick is
  // "live" when it is not inside any round's [startTick, freezeEndTick) window
  // and not in the pre-match gap before round 1.
  private nextLiveTick(tick: number): number {
    if (this.freezeWindows.length === 0) return tick
    const first = this.freezeWindows[0]
    if (tick < first.startTick) return first.freezeEndTick
    for (const w of this.freezeWindows) {
      if (tick >= w.startTick && tick < w.freezeEndTick) return w.freezeEndTick
    }
    return tick
  }

  get interpolationFactor(): number {
    return this.fractionalTick - Math.floor(this.fractionalTick)
  }

  dispose(): void {
    // No-op for now; reserved for future cleanup
  }
}
