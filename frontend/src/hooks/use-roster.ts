import type { PlayerRosterEntry, PlayerRosterResponse } from "@/types/roster"

// fetchRoster is an imperative fetch function — used in the PixiJS lifecycle
// (similar to TickBuffer) rather than as a TanStack Query hook, because roster
// data is loaded in response to PixiJS subscription events, not React renders.
export async function fetchRoster(
  demoId: string,
  roundNumber: number,
  signal: AbortSignal
): Promise<PlayerRosterEntry[]> {
  const res = await fetch(
    `/api/v1/demos/${demoId}/rounds/${roundNumber}/players`,
    { credentials: "include", signal }
  )
  if (!res.ok) throw new Error(`Failed to fetch roster: ${res.status}`)
  const json: PlayerRosterResponse = await res.json()
  return json.data
}
