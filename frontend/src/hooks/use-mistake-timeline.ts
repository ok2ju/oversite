import { useQuery } from "@tanstack/react-query"
import { GetMistakeTimeline } from "@wailsjs/go/main/App"
import type { MistakeEntry } from "@/types/mistake"

// Per-(demo, steamID) analyzer findings. The Go side recomputes on every
// import and the rows are static for the lifetime of an import, so
// staleTime: Infinity matches the use-player-stats / use-scoreboard pattern.
export function useMistakeTimeline(
  demoId: string | null,
  steamId: string | null,
) {
  return useQuery({
    queryKey: ["mistakes", demoId, steamId],
    queryFn: () =>
      GetMistakeTimeline(demoId!, steamId!) as Promise<MistakeEntry[]>,
    enabled: !!demoId && !!steamId,
    staleTime: Infinity,
  })
}
