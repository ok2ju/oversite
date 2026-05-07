import { useEffect, useRef, useState } from "react"
import { useViewerStore } from "@/stores/viewer"
import { TickBuffer } from "@/lib/pixi/tick-buffer"
import type { TickData } from "@/types/demo"

const POLL_INTERVAL_MS = 250

// Maps steam_id -> last-known tick frame for that player. Money / inventory
// only mutate a few times per round so polling at 4Hz is plenty, and tracking
// per-player lets us keep the most recent state for someone who just died
// without having to walk back through the buffer ourselves.
export type LoadoutSnapshot = Record<string, TickData>

// useLoadoutSnapshot owns a dedicated TickBuffer (separate from the one inside
// ViewerCanvas) and polls the buffer at 4Hz so the team bars don't re-render
// every PixiJS frame. The polled tick is read from the viewer store via
// `getState()` to avoid making the hook itself re-render whenever the tick
// advances.
export function useLoadoutSnapshot(): LoadoutSnapshot {
  const demoId = useViewerStore((s) => s.demoId)
  const [snapshot, setSnapshot] = useState<LoadoutSnapshot>({})
  const bufferRef = useRef<TickBuffer | null>(null)

  useEffect(() => {
    if (!demoId) {
      setSnapshot({})
      bufferRef.current = null
      return
    }

    const buffer = new TickBuffer(demoId)
    bufferRef.current = buffer

    let lastTick = -1
    const interval = window.setInterval(() => {
      const currentTick = useViewerStore.getState().currentTick
      if (currentTick === lastTick) return
      lastTick = currentTick

      const rows = buffer.getTickData(currentTick)
      if (!rows || rows.length === 0) return

      setSnapshot((prev) => {
        const next: LoadoutSnapshot = { ...prev }
        let changed = false
        for (const row of rows) {
          const before = prev[row.steam_id]
          if (!before || !sameLoadout(before, row)) {
            next[row.steam_id] = row
            changed = true
          }
        }
        return changed ? next : prev
      })
    }, POLL_INTERVAL_MS)

    return () => {
      window.clearInterval(interval)
      buffer.dispose()
      bufferRef.current = null
    }
  }, [demoId])

  return snapshot
}

function sameLoadout(a: TickData, b: TickData): boolean {
  return (
    a.is_alive === b.is_alive &&
    a.health === b.health &&
    a.armor === b.armor &&
    a.money === b.money &&
    a.has_helmet === b.has_helmet &&
    a.has_defuser === b.has_defuser &&
    a.weapon === b.weapon &&
    a.inventory.length === b.inventory.length &&
    a.inventory.every((w, i) => w === b.inventory[i])
  )
}
