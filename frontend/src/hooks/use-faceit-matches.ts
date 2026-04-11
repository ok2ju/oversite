"use client"

import { useQuery } from "@tanstack/react-query"
import type { FaceitMatchListResponse } from "@/types/faceit"

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(url, { credentials: "include" })
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`)
  }
  return res.json()
}

export function useFaceitMatches(
  page = 1,
  perPage = 20,
  filters: { map?: string; result?: string } = {},
) {
  return useQuery({
    queryKey: ["faceit-matches", page, perPage, filters.map, filters.result],
    queryFn: () => {
      const params = new URLSearchParams({
        page: String(page),
        per_page: String(perPage),
      })
      if (filters.map) params.set("map_name", filters.map)
      if (filters.result) params.set("result", filters.result)
      return fetchJSON<FaceitMatchListResponse>(
        `/api/v1/faceit/matches?${params}`,
      )
    },
  })
}
