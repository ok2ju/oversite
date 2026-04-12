"use client"

import { useQuery } from "@tanstack/react-query"
import type { GameEventsResponse } from "@/types/demo"

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, { credentials: "include", ...init })
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`)
  }
  return res.json()
}

async function fetchGameEvents(
  demoId: string,
  signal?: AbortSignal,
): Promise<GameEventsResponse> {
  return fetchJSON<GameEventsResponse>(`/api/v1/demos/${demoId}/events`, {
    signal,
  })
}

export function useGameEvents(demoId: string | null) {
  return useQuery({
    queryKey: ["game-events", demoId],
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchGameEvents(demoId!, signal),
    enabled: !!demoId,
    staleTime: Infinity,
  })
}
