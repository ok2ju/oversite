import { useQuery } from "@tanstack/react-query"
import { GetPlayerAnalysis } from "@wailsjs/go/main/App"
import type { PlayerAnalysis } from "@/types/analysis"

// Per-(demo, steamID) summary row computed by analysis.RunMatchSummary. Like
// useMistakeTimeline, the rows are static for the lifetime of an import so
// staleTime: Infinity matches the use-player-stats / use-mistake-timeline
// pattern.
export function usePlayerAnalysis(
  demoId: string | null,
  steamId: string | null,
) {
  return useQuery({
    queryKey: ["player-analysis", demoId, steamId],
    queryFn: () =>
      GetPlayerAnalysis(demoId!, steamId!) as Promise<PlayerAnalysis>,
    enabled: !!demoId && !!steamId,
    staleTime: Infinity,
  })
}
