import { useQuery } from "@tanstack/react-query"
import { GetFaceitMatches } from "@wailsjs/go/main/App"
import type { FaceitMatchListResponse } from "@/types/faceit"

export function useFaceitMatches(
  page = 1,
  perPage = 20,
  filters: { map?: string; result?: string } = {},
) {
  return useQuery({
    queryKey: ["faceit-matches", page, perPage, filters.map, filters.result],
    queryFn: () =>
      GetFaceitMatches(
        page,
        perPage,
        filters.map ?? "",
        filters.result ?? "",
      ) as Promise<FaceitMatchListResponse>,
  })
}
