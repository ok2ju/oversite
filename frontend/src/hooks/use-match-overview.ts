import { useQuery } from "@tanstack/react-query"
import { GetMatchOverview } from "@wailsjs/go/main/App"
import type { MatchOverview } from "@/types/match-overview"

export function useMatchOverview(demoId: string | null) {
  return useQuery({
    queryKey: ["match-overview", demoId],
    queryFn: () =>
      GetMatchOverview(demoId!) as unknown as Promise<MatchOverview>,
    enabled: !!demoId,
    staleTime: Infinity,
  })
}
