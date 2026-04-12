import { useQuery } from "@tanstack/react-query"
import { GetFaceitProfile, GetEloHistory } from "@wailsjs/go/main/App"
import type { FaceitProfile, EloHistoryPoint } from "@/types/faceit"

export function useFaceitProfile() {
  return useQuery({
    queryKey: ["faceit", "profile"],
    queryFn: () => GetFaceitProfile() as Promise<FaceitProfile>,
  })
}

export function useEloHistory(days: number = 30) {
  return useQuery({
    queryKey: ["faceit", "elo-history", days],
    queryFn: () => GetEloHistory(days) as Promise<EloHistoryPoint[]>,
  })
}
