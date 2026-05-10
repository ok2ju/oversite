import { useQuery } from "@tanstack/react-query"
import { GetPlayerRoundAnalysis } from "@wailsjs/go/main/App"
import type { PlayerRoundEntry } from "@/types/analysis"

// Per-(demo, player) round breakdown computed by
// analysis.RunPlayerRoundAnalysis. Rows are static for the lifetime of an
// import, so staleTime: Infinity matches usePlayerAnalysis /
// useMistakeTimeline. Invalidated by useRecomputeAnalysis on the legacy
// backfill path.
export function usePlayerRoundAnalysis(
  demoId: string | null,
  steamId: string | null,
) {
  return useQuery({
    queryKey: ["player-round-analysis", demoId, steamId],
    queryFn: () =>
      GetPlayerRoundAnalysis(demoId!, steamId!) as Promise<PlayerRoundEntry[]>,
    enabled: !!demoId && !!steamId,
    staleTime: Infinity,
  })
}
