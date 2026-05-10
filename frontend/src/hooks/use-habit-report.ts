import { useQuery } from "@tanstack/react-query"
import { GetHabitReport } from "@wailsjs/go/main/App"
import type { HabitReport } from "@/types/analysis"

// Per-(demo, steamID) habit checklist computed by analysis.BuildHabitReport.
// Like usePlayerAnalysis, the rows are static for the lifetime of an import
// so staleTime: Infinity matches the use-analysis / use-mistake-timeline
// pattern.
export function useHabitReport(demoId: string | null, steamId: string | null) {
  return useQuery({
    queryKey: ["habit-report", demoId, steamId],
    queryFn: () => GetHabitReport(demoId!, steamId!) as Promise<HabitReport>,
    enabled: !!demoId && !!steamId,
    staleTime: Infinity,
  })
}
