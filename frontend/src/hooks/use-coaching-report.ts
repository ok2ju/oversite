import { useQuery } from "@tanstack/react-query"
import { GetCoachingReport } from "@wailsjs/go/main/App"
import type { CoachingReport } from "@/types/analysis"

// Aggregated coaching surface for one player across their last `lookback`
// demos. Data is derived from analyzer rows; staleTime: Infinity matches the
// pattern of useHabitReport / use-analysis (rows do not change for the
// lifetime of an import).
export function useCoachingReport(
  steamId: string | null,
  lookback: number = 10,
) {
  return useQuery({
    queryKey: ["coaching-report", steamId, lookback],
    queryFn: () =>
      GetCoachingReport(steamId!, lookback) as Promise<CoachingReport>,
    enabled: !!steamId,
    staleTime: Infinity,
  })
}
