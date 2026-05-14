import { useEffect, useMemo, useState } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useKillFeed } from "./use-game-events"

const POLL_INTERVAL_MS = 250

export interface LiveKDA {
  kills: number
  assists: number
  deaths: number
}

export type LiveKDAMap = Record<string, LiveKDA>

// useLiveKDA returns running KDA totals up to the current playback tick,
// keyed by steam_id. It folds the cached kill-feed (useKillFeed,
// staleTime: Infinity) over the viewer-store tick instead of issuing a
// per-tick Wails call.
//
// Accounting matches internal/demo/stats.go: self-kills don't credit the
// attacker (but still credit the victim's death), world kills (empty
// attacker) only credit the death, and assists come from the promoted
// assister_steam_id column. The polling cadence + getState() pattern mirror
// useLoadoutSnapshot so the team-bars don't re-render every PixiJS frame.
export function useLiveKDA(): LiveKDAMap {
  const demoId = useViewerStore((s) => s.demoId)
  const { data: kills } = useKillFeed(demoId)
  const [snapshot, setSnapshot] = useState<LiveKDAMap>({})

  // Sort once per kill-feed payload so the per-tick walk can short-circuit
  // on the first event past the current tick.
  const sorted = useMemo(() => {
    if (!kills || kills.length === 0) return []
    return [...kills].sort((a, b) => a.tick - b.tick)
  }, [kills])

  useEffect(() => {
    if (!demoId) {
      setSnapshot({})
      return
    }
    if (sorted.length === 0) {
      // Either the feed hasn't arrived yet (then we'll re-run on `sorted`
      // changing) or the demo simply has no kills. Either way reset so a
      // stale snapshot from a previous demo doesn't bleed through.
      setSnapshot({})
      return
    }

    let lastTick = -1
    const interval = window.setInterval(() => {
      const currentTick = useViewerStore.getState().currentTick
      if (currentTick === lastTick) return
      lastTick = currentTick

      const next: LiveKDAMap = {}
      const ensure = (sid: string): LiveKDA => {
        let row = next[sid]
        if (!row) {
          row = { kills: 0, assists: 0, deaths: 0 }
          next[sid] = row
        }
        return row
      }

      for (const ev of sorted) {
        if (ev.tick > currentTick) break
        const attacker = ev.attacker_steam_id
        const victim = ev.victim_steam_id
        const assister = ev.assister_steam_id
        const isSelfKill = !!attacker && attacker === victim
        if (attacker && !isSelfKill) {
          ensure(attacker).kills += 1
        }
        if (victim) {
          ensure(victim).deaths += 1
        }
        if (assister) {
          ensure(assister).assists += 1
        }
      }

      setSnapshot(next)
    }, POLL_INTERVAL_MS)

    return () => {
      window.clearInterval(interval)
    }
  }, [demoId, sorted])

  return snapshot
}
