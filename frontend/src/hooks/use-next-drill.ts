import { useQuery } from "@tanstack/react-query"
import { GetNextDrill } from "@wailsjs/go/main/App"
import type { NextDrill } from "@/types/analysis"

// Per-(demo, steamID) drill prescription. Mirrors useHabitReport — the rows
// are static for the lifetime of an import so staleTime: Infinity matches
// the use-player-stats / use-mistake-timeline pattern.
export function useNextDrill(demoId: string | null, steamId: string | null) {
  return useQuery({
    queryKey: ["next-drill", demoId, steamId],
    queryFn: () => GetNextDrill(demoId!, steamId!) as Promise<NextDrill>,
    enabled: !!demoId && !!steamId,
    staleTime: Infinity,
  })
}
