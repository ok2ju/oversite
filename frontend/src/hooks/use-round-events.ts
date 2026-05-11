import { useMemo } from "react"
import { useGameEvents } from "./use-game-events"
import type { GameEvent } from "@/types/demo"

// Thin selector over the full demo-events cache. Returns the events whose
// tick falls inside [startTick, endTick]. The full cache is held by
// useGameEvents (staleTime: Infinity), so flipping rounds is a synchronous
// memo lookup — no second Wails round-trip, no second TanStack cache entry.
export function useRoundEvents(
  demoId: string | null,
  startTick: number,
  endTick: number,
): { data: GameEvent[]; isLoading: boolean } {
  const { data, isLoading } = useGameEvents(demoId)
  const events = useMemo(() => {
    if (!data) return []
    return data.filter((e) => e.tick >= startTick && e.tick <= endTick)
  }, [data, startTick, endTick])
  return { data: events, isLoading }
}
