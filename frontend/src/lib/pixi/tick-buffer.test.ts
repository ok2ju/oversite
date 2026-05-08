import { describe, it, expect, vi, beforeEach } from "vitest"
import type { TickData } from "@/types/demo"
import { TickBuffer, type FetchTicksFn } from "./tick-buffer"

function makeTick(tick: number, steamId = "76561198000000001"): TickData {
  return {
    tick,
    steam_id: steamId,
    x: tick * 1.0,
    y: tick * 2.0,
    z: 0,
    yaw: 90,
    health: 100,
    armor: 100,
    is_alive: true,
    weapon: "ak47",
    money: 0,
    has_helmet: false,
    has_defuser: false,
    ammo_clip: 0,
    ammo_reserve: 0,
  }
}

function makeChunkData(start: number, end: number): TickData[] {
  const data: TickData[] = []
  for (let t = start; t <= end; t++) {
    data.push(makeTick(t, "76561198000000001"))
    data.push(makeTick(t, "76561198000000002"))
  }
  return data
}

describe("TickBuffer", () => {
  let fetchFn: ReturnType<typeof vi.fn<FetchTicksFn>>
  let buffer: TickBuffer

  beforeEach(() => {
    fetchFn = vi.fn<FetchTicksFn>()
    buffer = new TickBuffer("demo-1", {
      fetchFn,
      chunkSize: 100,
      maxCachedChunks: 3,
    })
  })

  describe("chunk calculation", () => {
    it("calculates correct chunk boundaries for tick 0", () => {
      fetchFn.mockResolvedValue([])
      buffer.getTickData(0)
      expect(fetchFn).toHaveBeenCalledWith(
        "demo-1",
        0,
        99,
        expect.any(AbortSignal),
      )
    })

    it("calculates correct chunk boundaries for tick 150", () => {
      fetchFn.mockResolvedValue([])
      buffer.getTickData(150)
      expect(fetchFn).toHaveBeenCalledWith(
        "demo-1",
        100,
        199,
        expect.any(AbortSignal),
      )
    })

    it("calculates correct chunk boundaries for tick 99", () => {
      fetchFn.mockResolvedValue([])
      buffer.getTickData(99)
      expect(fetchFn).toHaveBeenCalledWith(
        "demo-1",
        0,
        99,
        expect.any(AbortSignal),
      )
    })
  })

  describe("fetch triggers", () => {
    it("fetches on cache miss", () => {
      fetchFn.mockResolvedValue([])
      buffer.getTickData(50)
      expect(fetchFn).toHaveBeenCalledTimes(1)
    })

    it("does not fetch on cache hit", async () => {
      const data = makeChunkData(0, 99)
      fetchFn.mockResolvedValue(data)
      buffer.getTickData(50)
      // Wait for the fetch to complete
      await vi.waitFor(() => {
        expect(buffer.getTickData(50)).not.toBeNull()
      })
      // Reset call count after initial fetch
      fetchFn.mockClear()
      buffer.getTickData(50)
      expect(fetchFn).not.toHaveBeenCalled()
    })

    it("returns null synchronously on cache miss", () => {
      fetchFn.mockResolvedValue([])
      const result = buffer.getTickData(50)
      expect(result).toBeNull()
    })

    it("returns cached data synchronously after fetch completes", async () => {
      const data = makeChunkData(0, 99)
      fetchFn.mockResolvedValue(data)
      buffer.getTickData(0)
      await vi.waitFor(() => {
        const result = buffer.getTickData(0)
        expect(result).not.toBeNull()
      })
      const result = buffer.getTickData(0)
      expect(result).toHaveLength(2) // 2 players at tick 0
      expect(result![0].steam_id).toBe("76561198000000001")
    })
  })

  describe("look-ahead", () => {
    it("pre-fetches next chunk when entering last 25% of current chunk", async () => {
      const chunkData = makeChunkData(0, 99)
      fetchFn.mockResolvedValue(chunkData)
      // Access tick in first chunk to populate it
      buffer.getTickData(0)
      await vi.waitFor(() => {
        expect(buffer.getTickData(0)).not.toBeNull()
      })

      fetchFn.mockClear()
      fetchFn.mockResolvedValue(makeChunkData(100, 199))

      // Access tick at 75% of chunk (tick 75 out of 0-99)
      buffer.getTickData(75)
      expect(fetchFn).toHaveBeenCalledWith(
        "demo-1",
        100,
        199,
        expect.any(AbortSignal),
      )
    })

    it("does not pre-fetch when in first 75% of chunk", async () => {
      const chunkData = makeChunkData(0, 99)
      fetchFn.mockResolvedValue(chunkData)
      buffer.getTickData(0)
      await vi.waitFor(() => {
        expect(buffer.getTickData(0)).not.toBeNull()
      })

      fetchFn.mockClear()
      // Access tick at 50% of chunk
      buffer.getTickData(50)
      expect(fetchFn).not.toHaveBeenCalled()
    })
  })

  describe("seek", () => {
    it("fetches the target chunk on seek", () => {
      fetchFn.mockResolvedValue([])
      buffer.seek(500)
      expect(fetchFn).toHaveBeenCalledWith(
        "demo-1",
        500,
        599,
        expect.any(AbortSignal),
      )
    })

    it("aborts in-flight fetches for unneeded chunks on seek", async () => {
      const abortSignals: AbortSignal[] = []
      fetchFn.mockImplementation((_demoId, _start, _end, signal) => {
        abortSignals.push(signal)
        return new Promise(() => {}) // never resolves
      })

      // Start fetch for chunk 0-99
      buffer.getTickData(50)
      expect(abortSignals).toHaveLength(1)

      // Seek far away — should abort the pending fetch
      buffer.seek(5000)
      expect(abortSignals[0].aborted).toBe(true)
    })
  })

  describe("out-of-order responses", () => {
    it("late response for old chunk is still cached", async () => {
      const resolvers: Array<(v: TickData[]) => void> = []
      fetchFn.mockImplementation(() => {
        return new Promise<TickData[]>((resolve) => {
          resolvers.push(resolve)
        })
      })

      // Request chunk 0
      buffer.getTickData(0)
      // Request chunk 100
      buffer.getTickData(100)

      expect(resolvers).toHaveLength(2)

      // Resolve chunk 100 first
      resolvers[1](makeChunkData(100, 199))
      await vi.waitFor(() => {
        expect(buffer.getTickData(100)).not.toBeNull()
      })

      // Now resolve chunk 0
      resolvers[0](makeChunkData(0, 99))
      await vi.waitFor(() => {
        expect(buffer.getTickData(0)).not.toBeNull()
      })
    })
  })

  describe("sparse sampling", () => {
    it("returns the nearest prior sample for unsampled ticks in a loaded chunk", async () => {
      // Simulate backend sampling every 4th tick.
      const sparse: TickData[] = []
      for (let t = 0; t < 100; t += 4) {
        sparse.push(makeTick(t, "76561198000000001"))
        sparse.push(makeTick(t, "76561198000000002"))
      }
      fetchFn.mockResolvedValue(sparse)

      buffer.getTickData(0)
      await vi.waitFor(() => expect(buffer.getTickData(0)).not.toBeNull())

      // Tick 5 is not sampled; should return tick 4's data, not empty.
      const result = buffer.getTickData(5)
      expect(result).toHaveLength(2)
      expect(result![0].tick).toBe(4)
    })
  })

  describe("error handling", () => {
    it("returns null on network error without crashing", async () => {
      fetchFn.mockRejectedValue(new Error("network error"))
      const result = buffer.getTickData(50)
      expect(result).toBeNull()

      // Wait for rejection to settle
      await vi.waitFor(() => {
        // After error, re-fetching should be allowed
        fetchFn.mockClear()
        fetchFn.mockResolvedValue([])
        buffer.getTickData(50)
        expect(fetchFn).toHaveBeenCalled()
      })
    })

    it("retries on next access after error", async () => {
      fetchFn.mockRejectedValueOnce(new Error("network error"))
      buffer.getTickData(50)

      // Wait for rejection to settle
      await new Promise((r) => setTimeout(r, 10))

      fetchFn.mockResolvedValue(makeChunkData(0, 99))
      buffer.getTickData(50)
      expect(fetchFn).toHaveBeenCalledTimes(2)
    })
  })

  describe("cache eviction", () => {
    it("evicts LRU chunk when exceeding maxCachedChunks", async () => {
      // maxCachedChunks = 3
      fetchFn.mockImplementation((_demoId, start, end) => {
        return Promise.resolve(makeChunkData(start, end))
      })

      // Fill 3 chunks
      buffer.getTickData(0)
      await vi.waitFor(() => expect(buffer.getTickData(0)).not.toBeNull())

      buffer.getTickData(100)
      await vi.waitFor(() => expect(buffer.getTickData(100)).not.toBeNull())

      buffer.getTickData(200)
      await vi.waitFor(() => expect(buffer.getTickData(200)).not.toBeNull())

      // Access chunk 0 to make it recently used (chunk 100 becomes LRU)
      buffer.getTickData(0)

      // Load a 4th chunk — should evict chunk 100 (LRU)
      buffer.getTickData(300)
      await vi.waitFor(() => expect(buffer.getTickData(300)).not.toBeNull())

      // Chunk 100 should be evicted
      fetchFn.mockClear()
      fetchFn.mockResolvedValue(makeChunkData(100, 199))
      buffer.getTickData(100)
      expect(fetchFn).toHaveBeenCalled() // re-fetched because evicted
    })
  })

  describe("dispose", () => {
    it("aborts all in-flight requests", () => {
      const abortSignals: AbortSignal[] = []
      fetchFn.mockImplementation((_demoId, _start, _end, signal) => {
        abortSignals.push(signal)
        return new Promise(() => {})
      })

      buffer.getTickData(0)
      buffer.getTickData(100)
      expect(abortSignals).toHaveLength(2)

      buffer.dispose()

      expect(abortSignals[0].aborted).toBe(true)
      expect(abortSignals[1].aborted).toBe(true)
    })

    it("clears cache", async () => {
      fetchFn.mockResolvedValue(makeChunkData(0, 99))
      buffer.getTickData(0)
      await vi.waitFor(() => expect(buffer.getTickData(0)).not.toBeNull())

      buffer.dispose()

      fetchFn.mockClear()
      fetchFn.mockResolvedValue(makeChunkData(0, 99))
      // After dispose, getTickData should return null (cache cleared)
      const result = buffer.getTickData(0)
      expect(result).toBeNull()
    })
  })

  describe("getFramePair", () => {
    it("returns null pair when chunk is not yet loaded", () => {
      fetchFn.mockResolvedValue([])
      const pair = buffer.getFramePair(50)
      expect(pair.current).toBeNull()
      expect(pair.next).toBeNull()
    })

    it("returns the sample at-or-before and the next sample after", async () => {
      // Sparse: samples every 4th tick. fractionalTick=5.5 should pair
      // (current=4, next=8).
      const sparse: TickData[] = []
      for (let t = 0; t < 100; t += 4) {
        sparse.push(makeTick(t, "76561198000000001"))
      }
      fetchFn.mockResolvedValue(sparse)
      buffer.getTickData(0)
      await vi.waitFor(() => expect(buffer.getTickData(0)).not.toBeNull())

      const pair = buffer.getFramePair(5.5)
      expect(pair.current?.tick).toBe(4)
      expect(pair.next?.tick).toBe(8)
    })

    it("reuses the same FramePair and SampleFrame objects across calls", async () => {
      // Allocation contract: callers must consume references synchronously
      // and not retain them. This test guards that we keep the contract by
      // reusing the buffer-owned scratch slots.
      const sparse: TickData[] = []
      for (let t = 0; t < 100; t += 4) {
        sparse.push(makeTick(t, "76561198000000001"))
      }
      fetchFn.mockResolvedValue(sparse)
      buffer.getTickData(0)
      await vi.waitFor(() => expect(buffer.getTickData(0)).not.toBeNull())

      const pairA = buffer.getFramePair(5.5)
      const currentA = pairA.current
      const nextA = pairA.next
      const pairB = buffer.getFramePair(9.5)
      // Same FramePair wrapper — refs reused
      expect(pairB).toBe(pairA)
      // Same SampleFrame slots, but their fields advance with the call
      expect(pairB.current).toBe(currentA)
      expect(pairB.next).toBe(nextA)
      expect(pairB.current?.tick).toBe(8)
      expect(pairB.next?.tick).toBe(12)
    })
  })
})
