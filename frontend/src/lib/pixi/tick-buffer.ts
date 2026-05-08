import { GetDemoTicks } from "@wailsjs/go/main/App"
import type { TickData } from "@/types/demo"

export type FetchTicksFn = (
  demoId: string,
  startTick: number,
  endTick: number,
  signal: AbortSignal,
) => Promise<TickData[]>

export const DEFAULT_CHUNK_SIZE = 6400
const DEFAULT_LOOK_AHEAD_THRESHOLD = 0.75
const DEFAULT_MAX_CACHED_CHUNKS = 10
const MAX_SAMPLE_WALK = 128

export interface SampleFrame {
  tick: number
  data: TickData[]
}

export interface FramePair {
  current: SampleFrame | null
  next: SampleFrame | null
}

interface TickBufferOptions {
  fetchFn?: FetchTicksFn
  chunkSize?: number
  lookAheadThreshold?: number
  maxCachedChunks?: number
}

interface ChunkState {
  data: Map<number, TickData[]>
  lastAccessed: number
}

// Uses the Wails binding directly instead of TanStack Query because TickBuffer
// manages its own imperative LRU chunk cache with abort-on-seek — outside
// React's lifecycle.
//
// Inventory used to be normalized here (CSV → string[]) but moved to per-round
// storage in migration 011 — see useRoundLoadouts. Tick data is now a flat
// pass-through, no per-row mutation.
const defaultFetchFn: FetchTicksFn = async (demoId, startTick, endTick) => {
  return (await GetDemoTicks(
    demoId,
    startTick,
    endTick,
  )) as unknown as TickData[]
}

export class TickBuffer {
  private demoId: string
  private chunkSize: number
  private lookAheadThreshold: number
  private maxCachedChunks: number
  private fetchFn: FetchTicksFn

  private cache = new Map<number, ChunkState>()
  private inFlight = new Map<number, AbortController>()
  private failedChunks = new Set<number>()

  // Reused per-call SampleFrame slots returned from getFramePair. The caller
  // (viewer-canvas tickerFn) destructures and consumes both within the same
  // synchronous frame and never stores the SampleFrame references — only the
  // inner `.data` arrays (which are owned by the chunk cache, not the slots).
  // Reusing the wrappers eliminates ~128 small-object allocs/sec at 64 Hz.
  private currentScratch: SampleFrame = { tick: 0, data: [] }
  private nextScratch: SampleFrame = { tick: 0, data: [] }
  // Returned object reuses the same shape so callers can keep destructuring;
  // its fields point to the scratch slots above (or null when no sample).
  private framePair: { current: SampleFrame | null; next: SampleFrame | null } =
    { current: null, next: null }

  constructor(demoId: string, options: TickBufferOptions = {}) {
    this.demoId = demoId
    this.fetchFn = options.fetchFn ?? defaultFetchFn
    this.chunkSize = options.chunkSize ?? DEFAULT_CHUNK_SIZE
    this.lookAheadThreshold =
      options.lookAheadThreshold ?? DEFAULT_LOOK_AHEAD_THRESHOLD
    this.maxCachedChunks = options.maxCachedChunks ?? DEFAULT_MAX_CACHED_CHUNKS
  }

  getTickData(tick: number): TickData[] | null {
    const chunkStart = this.chunkStartFor(tick)

    const cached = this.cache.get(chunkStart)
    if (cached) {
      cached.lastAccessed = Date.now()
      this.maybePrefetchNext(tick, chunkStart)

      // Backend samples every Nth tick (see parser.go tickInterval). Walk back
      // to the most recent sample within this chunk so frames between samples
      // render the last known state instead of flickering to empty.
      const floor = Math.max(chunkStart, tick - MAX_SAMPLE_WALK)
      for (let t = tick; t >= floor; t--) {
        const data = cached.data.get(t)
        if (data) return data
      }
      return []
    }

    this.fetchChunk(chunkStart)
    return null
  }

  // getFramePair returns the sample at-or-before floor(fractionalTick) and the
  // next sample strictly after it, so the renderer can interpolate positions
  // between snapshots. Returns { current: null, next: null } if the containing
  // chunk is not yet loaded (fetch is triggered).
  //
  // ALLOCATION CONTRACT: The returned FramePair object and its non-null
  // SampleFrame slots are reused across calls (this.framePair / this.*Scratch).
  // Callers must consume `current.tick`, `current.data`, `next.tick`,
  // `next.data` synchronously within the same tick and MUST NOT retain the
  // FramePair or SampleFrame references across frames. The inner `.data`
  // arrays remain owned by the chunk cache and are safe to forward to a
  // consumer (e.g. PlayerLayer.update) for the duration of the synchronous
  // frame.
  getFramePair(fractionalTick: number): FramePair {
    const intTick = Math.floor(fractionalTick)
    const chunkStart = this.chunkStartFor(intTick)
    const cached = this.cache.get(chunkStart)

    if (!cached) {
      this.fetchChunk(chunkStart)
      this.framePair.current = null
      this.framePair.next = null
      return this.framePair
    }
    cached.lastAccessed = Date.now()
    this.maybePrefetchNext(intTick, chunkStart)

    const chunkEnd = chunkStart + this.chunkSize - 1

    // Walk back from intTick to find the most recent sample; write into the
    // pre-allocated currentScratch instead of constructing a fresh literal.
    let currentFound = false
    const backFloor = Math.max(chunkStart, intTick - MAX_SAMPLE_WALK)
    for (let t = intTick; t >= backFloor; t--) {
      const d = cached.data.get(t)
      if (d) {
        this.currentScratch.tick = t
        this.currentScratch.data = d
        currentFound = true
        break
      }
    }
    this.framePair.current = currentFound ? this.currentScratch : null

    let nextFound = false
    const forwardCeiling = Math.min(chunkEnd, intTick + MAX_SAMPLE_WALK)
    for (let t = intTick + 1; t <= forwardCeiling; t++) {
      const d = cached.data.get(t)
      if (d) {
        this.nextScratch.tick = t
        this.nextScratch.data = d
        nextFound = true
        break
      }
    }

    // At the tail of a chunk, fall through to the next chunk if it's cached so
    // playback stays smooth across the boundary. We don't fetch here.
    if (!nextFound && intTick + MAX_SAMPLE_WALK > chunkEnd) {
      const nextChunkStart = chunkStart + this.chunkSize
      const nextCached = this.cache.get(nextChunkStart)
      if (nextCached) {
        const nextChunkEnd = nextChunkStart + this.chunkSize - 1
        const ceiling = Math.min(nextChunkEnd, intTick + MAX_SAMPLE_WALK)
        for (let t = nextChunkStart; t <= ceiling; t++) {
          const d = nextCached.data.get(t)
          if (d) {
            this.nextScratch.tick = t
            this.nextScratch.data = d
            nextFound = true
            break
          }
        }
      }
    }
    this.framePair.next = nextFound ? this.nextScratch : null

    return this.framePair
  }

  seek(tick: number): void {
    const targetChunk = this.chunkStartFor(tick)

    // Abort in-flight fetches for chunks that are not the target
    for (const [chunkStart, controller] of this.inFlight) {
      if (chunkStart !== targetChunk) {
        controller.abort()
        this.inFlight.delete(chunkStart)
      }
    }

    if (!this.cache.has(targetChunk) && !this.inFlight.has(targetChunk)) {
      this.fetchChunk(targetChunk)
    }
  }

  dispose(): void {
    for (const controller of this.inFlight.values()) {
      controller.abort()
    }
    this.inFlight.clear()
    this.cache.clear()
    this.failedChunks.clear()
  }

  private chunkStartFor(tick: number): number {
    return Math.floor(tick / this.chunkSize) * this.chunkSize
  }

  private maybePrefetchNext(tick: number, chunkStart: number): void {
    const positionInChunk = tick - chunkStart
    if (positionInChunk >= this.chunkSize * this.lookAheadThreshold) {
      const nextChunk = chunkStart + this.chunkSize
      if (!this.cache.has(nextChunk) && !this.inFlight.has(nextChunk)) {
        this.fetchChunk(nextChunk)
      }
    }
  }

  private fetchChunk(chunkStart: number): void {
    if (this.inFlight.has(chunkStart)) return

    const controller = new AbortController()
    this.inFlight.set(chunkStart, controller)
    this.failedChunks.delete(chunkStart)

    const chunkEnd = chunkStart + this.chunkSize - 1

    this.fetchFn(this.demoId, chunkStart, chunkEnd, controller.signal)
      .then((rows) => {
        this.inFlight.delete(chunkStart)

        // Group by tick
        const tickMap = new Map<number, TickData[]>()
        for (const row of rows) {
          let arr = tickMap.get(row.tick)
          if (!arr) {
            arr = []
            tickMap.set(row.tick, arr)
          }
          arr.push(row)
        }

        this.evictIfNeeded()
        this.cache.set(chunkStart, { data: tickMap, lastAccessed: Date.now() })
      })
      .catch((err) => {
        this.inFlight.delete(chunkStart)
        if (err instanceof DOMException && err.name === "AbortError") return
        this.failedChunks.add(chunkStart)
      })
  }

  private evictIfNeeded(): void {
    while (this.cache.size >= this.maxCachedChunks) {
      let lruKey: number | null = null
      let lruTime = Infinity
      for (const [key, state] of this.cache) {
        if (state.lastAccessed < lruTime) {
          lruTime = state.lastAccessed
          lruKey = key
        }
      }
      if (lruKey !== null) {
        this.cache.delete(lruKey)
      } else {
        break
      }
    }
  }
}
