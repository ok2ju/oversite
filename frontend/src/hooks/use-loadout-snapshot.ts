import { useEffect, useState } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useTickBufferStore } from "@/stores/tick-buffer"
import type { TickData } from "@/types/demo"

const POLL_INTERVAL_MS = 250

// Maps steam_id -> last-known tick frame for that player. Money / inventory
// only mutate a few times per round so polling at 4Hz is plenty, and tracking
// per-player lets us keep the most recent state for someone who just died
// without having to walk back through the buffer ourselves.
export type LoadoutSnapshot = Record<string, TickData>

// useLoadoutSnapshot polls the shared TickBuffer (owned by ViewerCanvas and
// published via useTickBufferStore) at 4Hz so the team bars don't re-render
// every PixiJS frame. The polled tick is read from the viewer store via
// `getState()` to avoid making the hook itself re-render whenever the tick
// advances. We deliberately do NOT allocate our own TickBuffer — that would
// double the frame-buffer memory and duplicate the network/decode work for
// the same demoId.
export function useLoadoutSnapshot(): LoadoutSnapshot {
  const demoId = useViewerStore((s) => s.demoId)
  const [snapshot, setSnapshot] = useState<LoadoutSnapshot>({})

  useEffect(() => {
    if (!demoId) {
      setSnapshot({})
      return
    }

    let lastTick = -1
    const interval = window.setInterval(() => {
      // Read the buffer fresh on each tick — viewer-canvas may not have
      // mounted yet, or may have just swapped to a new demo. The store's
      // demoId guards against reading a buffer that belongs to a previous
      // demo if a stale tick fires during the swap.
      const { demoId: bufferDemoId, buffer } = useTickBufferStore.getState()
      if (!buffer || bufferDemoId !== demoId) return

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
    }
  }, [demoId])

  return snapshot
}

// Inventory intentionally omitted: it's per-round (migration 011) and
// supplied separately via useRoundLoadouts, so the per-tick equality check
// only covers fields that actually mutate during a round.
function sameLoadout(a: TickData, b: TickData): boolean {
  return (
    a.is_alive === b.is_alive &&
    a.health === b.health &&
    a.armor === b.armor &&
    a.money === b.money &&
    a.has_helmet === b.has_helmet &&
    a.has_defuser === b.has_defuser &&
    a.weapon === b.weapon &&
    a.ammo_clip === b.ammo_clip &&
    a.ammo_reserve === b.ammo_reserve
  )
}
