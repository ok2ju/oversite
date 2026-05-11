import { useQuery } from "@tanstack/react-query"
import { ListDuelsForPlayer } from "@wailsjs/go/main/App"
import type { DuelEntry } from "@/types/duel"

// Per-(demo, steamID) duels reconstructed from the event stream. Static
// for the lifetime of an import (analysis is deterministic per
// AnalysisVersion), so staleTime: Infinity mirrors useMistakeTimeline.
export function useDuelTimeline(demoId: string | null, steamId: string | null) {
  return useQuery({
    queryKey: ["duels", demoId, steamId],
    queryFn: () =>
      ListDuelsForPlayer(demoId!, steamId!) as Promise<DuelEntry[]>,
    enabled: !!demoId && !!steamId,
    staleTime: Infinity,
  })
}
