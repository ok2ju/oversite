export interface RoundBoundary {
  roundNumber: number
  startTick: number
  endTick: number
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
  private _tickInterval = 4

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
      this.fractionalTick = totalTicks
      this.setTick(totalTicks)
      this.pauseFn()
      return
    }

    // Check round boundary crossing
    if (this.autoPauseEnabled && this.roundBoundaries.length > 0) {
      for (const boundary of this.roundBoundaries) {
        if (previousTick < boundary.endTick && this.fractionalTick >= boundary.endTick) {
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
    const clamped = Math.max(0, Math.min(tick, totalTicks))
    this.fractionalTick = clamped
    this.setTick(Math.floor(clamped))
  }

  setRoundBoundaries(boundaries: RoundBoundary[]): void {
    this.roundBoundaries = [...boundaries].sort((a, b) => a.endTick - b.endTick)
  }

  setAutoPause(enabled: boolean): void {
    this.autoPauseEnabled = enabled
  }

  get interpolationFactor(): number {
    return this.fractionalTick - Math.floor(this.fractionalTick)
  }

  get tickInterval(): number {
    return this._tickInterval
  }

  setTickInterval(n: number): void {
    this._tickInterval = n
  }

  dispose(): void {
    // No-op for now; reserved for future cleanup
  }
}
