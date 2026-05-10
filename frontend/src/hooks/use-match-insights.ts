import { useQuery } from "@tanstack/react-query"
import { GetMatchInsights } from "@wailsjs/go/main/App"
import type { MatchInsights } from "@/types/analysis"

// Per-(demo) team-level summary backing the analysis page's head-to-head
// view. Like the other analysis hooks, the rows are static for the lifetime
// of an import so staleTime: Infinity matches the existing pattern.
export function useMatchInsights(demoId: string | null) {
  return useQuery({
    queryKey: ["match-insights", demoId],
    queryFn: () => GetMatchInsights(demoId!) as Promise<MatchInsights>,
    enabled: !!demoId,
    staleTime: Infinity,
  })
}
