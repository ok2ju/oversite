"use client"

import { useQuery } from "@tanstack/react-query"
import type { RoundsResponse } from "@/types/round"

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, { credentials: "include", ...init })
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`)
  }
  return res.json()
}

async function fetchRounds(demoId: string, signal?: AbortSignal): Promise<RoundsResponse> {
  return fetchJSON<RoundsResponse>(`/api/v1/demos/${demoId}/rounds`, { signal })
}

export function useRounds(demoId: string | null) {
  return useQuery({
    queryKey: ["rounds", demoId],
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchRounds(demoId!, signal),
    enabled: !!demoId,
    staleTime: Infinity,
  })
}
