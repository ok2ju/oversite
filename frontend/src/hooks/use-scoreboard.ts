import { useQuery } from "@tanstack/react-query"
import { GetScoreboard } from "@wailsjs/go/main/App"
import type { ScoreboardEntry } from "@/types/scoreboard"

export function useScoreboard(demoId: string | null) {
  return useQuery({
    queryKey: ["scoreboard", demoId],
    queryFn: () => GetScoreboard(demoId!) as Promise<ScoreboardEntry[]>,
    enabled: !!demoId,
    staleTime: Infinity,
  })
}
