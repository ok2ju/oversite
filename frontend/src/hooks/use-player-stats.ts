import { useQuery } from "@tanstack/react-query"
import { GetPlayerMatchStats } from "@wailsjs/go/main/App"
import type { PlayerMatchStats } from "@/types/player-stats"

// Per-(demo, steamID) deep stats. Computed on the Go side from already-ingested
// data, so the underlying inputs are deterministic — staleTime: Infinity is
// safe and matches use-scoreboard.
export function usePlayerStats(demoId: string | null, steamId: string | null) {
  return useQuery({
    queryKey: ["player-stats", demoId, steamId],
    queryFn: () =>
      GetPlayerMatchStats(demoId!, steamId!) as Promise<PlayerMatchStats>,
    enabled: !!demoId && !!steamId,
    staleTime: Infinity,
  })
}
