"use client"

import { useQuery } from "@tanstack/react-query"
import type { FaceitProfileResponse, EloHistoryResponse } from "@/types/faceit"

async function fetchJSON<T>(url: string): Promise<T> {
  const res = await fetch(url, { credentials: "include" })
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`)
  }
  return res.json()
}

export function useFaceitProfile() {
  return useQuery({
    queryKey: ["faceit", "profile"],
    queryFn: () => fetchJSON<FaceitProfileResponse>("/api/v1/faceit/profile"),
    select: (res) => res.data,
  })
}

export function useEloHistory(days: number = 30) {
  return useQuery({
    queryKey: ["faceit", "elo-history", days],
    queryFn: () =>
      fetchJSON<EloHistoryResponse>(`/api/v1/faceit/elo-history?days=${days}`),
    select: (res) => res.data,
  })
}
