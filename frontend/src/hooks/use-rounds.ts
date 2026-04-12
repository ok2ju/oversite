import { useQuery } from "@tanstack/react-query"
import { GetDemoRounds } from "@wailsjs/go/main/App"
import type { Round } from "@/types/round"

export function useRounds(demoId: string | null) {
  return useQuery({
    queryKey: ["rounds", demoId],
    queryFn: () => GetDemoRounds(demoId!) as Promise<Round[]>,
    enabled: !!demoId,
    staleTime: Infinity,
  })
}
