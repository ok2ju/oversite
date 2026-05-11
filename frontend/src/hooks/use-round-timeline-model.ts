import { useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useGameEvents } from "./use-game-events"
import { useMistakeTimeline } from "./use-mistake-timeline"
import { buildLanes } from "@/lib/timeline/build-lanes"
import type { RoundTimelineModel } from "@/lib/timeline/types"
import type { Round } from "@/types/round"

// Composes the round model from the existing TanStack caches. Recomputes only
// when the round, the active player, the filter chips, or the lane width
// changes — never on playhead tick advances.
export function useRoundTimelineModel(
  round: Round | null,
  laneWidthPx: number,
): { model: RoundTimelineModel | null; isLoading: boolean } {
  const demoId = useViewerStore((s) => s.demoId)
  const selectedPlayerSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const filters = useViewerStore((s) => s.timelineFilters)

  const { data: events, isLoading: eventsLoading } = useGameEvents(demoId)
  const { data: mistakes, isLoading: mistakesLoading } = useMistakeTimeline(
    demoId,
    selectedPlayerSteamId,
  )

  const model = useMemo(() => {
    if (!round) return null
    if (!events) return null
    return buildLanes({
      events,
      mistakes: mistakes ?? [],
      round,
      selectedPlayerSteamId,
      filters,
      laneWidthPx,
    })
  }, [round, events, mistakes, selectedPlayerSteamId, filters, laneWidthPx])

  return {
    model,
    isLoading: eventsLoading || (!!selectedPlayerSteamId && mistakesLoading),
  }
}
