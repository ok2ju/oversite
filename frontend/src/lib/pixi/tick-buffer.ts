import type { TickData, TickDataResponse } from "@/types/demo"

export type FetchTicksFn = (
  demoId: string,
  startTick: number,
  endTick: number,
  signal: AbortSignal,
) => Promise<TickData[]>

export const DEFAULT_CHUNK_SIZE = 6400
const DEFAULT_LOOK_AHEAD_THRESHOLD = 0.75
const DEFAULT_MAX_CACHED_CHUNKS = 10

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

// Uses raw fetch instead of TanStack Query because TickBuffer manages its own
// imperative LRU chunk cache with abort-on-seek — outside React's lifecycle.
const defaultFetchFn: FetchTicksFn = async (
  demoId,
  startTick,
  endTick,
  signal,
) => {
  const params = new URLSearchParams({
    start_tick: String(startTick),
    end_tick: String(endTick),
  })
  const res = await fetch(`/api/v1/demos/${demoId}/ticks?${params}`, {
    credentials: "include",
    signal,
  })
  if (!res.ok) throw new Error(`Failed to fetch ticks: ${res.status}`)
  const json: TickDataResponse = await res.json()
  return json.data
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
      return cached.data.get(tick) ?? []
    }

    this.fetchChunk(chunkStart)
    return null
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
