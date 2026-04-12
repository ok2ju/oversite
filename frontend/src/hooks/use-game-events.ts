import { useQuery } from "@tanstack/react-query"
import { GetDemoEvents } from "@wailsjs/go/main/App"
import type { GameEvent } from "@/types/demo"

export function useGameEvents(demoId: string | null) {
  return useQuery({
    queryKey: ["game-events", demoId],
    queryFn: () => GetDemoEvents(demoId!) as Promise<GameEvent[]>,
    enabled: !!demoId,
    staleTime: Infinity,
  })
}
