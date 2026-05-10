import { useEffect, useState } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useTickBufferStore } from "@/stores/tick-buffer"
import type { TickData } from "@/types/demo"

const POLL_INTERVAL_MS = 250
const SPEED_WINDOW = 8

export interface LiveHudFrame {
  tick: number
  data: TickData
  // Speed in CS2 units/second derived from a SPEED_WINDOW-sample sliding
  // window. Null until the buffer has produced enough samples to estimate.
  speedUps: number | null
}

// Polls the shared TickBuffer (owned by viewer-canvas, published via
// useTickBufferStore) at 4 Hz for one player's current frame and a derived
// speed estimate. Mirrors useLoadoutSnapshot — same lifecycle pattern, same
// no-extra-buffer policy. Returns null when no frame is available yet (buffer
// not mounted, player not in current sample, or selection cleared).
export function usePlayerLiveHud(steamId: string | null): LiveHudFrame | null {
  const demoId = useViewerStore((s) => s.demoId)
  const tickRate = useViewerStore((s) => s.tickRate)
  const [frame, setFrame] = useState<LiveHudFrame | null>(null)

  useEffect(() => {
    if (!demoId || !steamId) {
      setFrame(null)
      return
    }

    let lastTick = -1
    const speedSamples: Array<{ tick: number; x: number; y: number }> = []

    const interval = window.setInterval(() => {
      const { demoId: bufferDemoId, buffer } = useTickBufferStore.getState()
      if (!buffer || bufferDemoId !== demoId) return

      const currentTick = useViewerStore.getState().currentTick
      if (currentTick === lastTick) return
      lastTick = currentTick

      const rows = buffer.getTickData(currentTick)
      if (!rows || rows.length === 0) return

      const row = rows.find((r) => r.steam_id === steamId)
      if (!row) return

      // Track samples for an SPEED_WINDOW-sample sliding window. The tick
      // buffer samples every Nth tick (parser default 4) so the window covers
      // ~0.5s at 64 tps. Only append when alive — speed during death is noise.
      if (row.is_alive) {
        speedSamples.push({ tick: currentTick, x: row.x, y: row.y })
        if (speedSamples.length > SPEED_WINDOW) speedSamples.shift()
      } else {
        speedSamples.length = 0
      }

      let speedUps: number | null = null
      if (speedSamples.length >= 2 && tickRate > 0) {
        const first = speedSamples[0]
        const last = speedSamples[speedSamples.length - 1]
        const dt = (last.tick - first.tick) / tickRate
        if (dt > 0) {
          const dx = last.x - first.x
          const dy = last.y - first.y
          speedUps = Math.sqrt(dx * dx + dy * dy) / dt
        }
      }

      setFrame({ tick: currentTick, data: row, speedUps })
    }, POLL_INTERVAL_MS)

    return () => {
      window.clearInterval(interval)
    }
  }, [demoId, steamId, tickRate])

  return frame
}
