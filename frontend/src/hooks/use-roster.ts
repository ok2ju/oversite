import { GetRoundRoster } from "@wailsjs/go/main/App"
import type { PlayerRosterEntry } from "@/types/roster"

// fetchRoster is an imperative fetch function — used in the PixiJS lifecycle
// (similar to TickBuffer) rather than as a TanStack Query hook, because roster
// data is loaded in response to PixiJS subscription events, not React renders.
export async function fetchRoster(
  demoId: string,
  roundNumber: number,
  signal: AbortSignal,
): Promise<PlayerRosterEntry[]> {
  void signal
  return GetRoundRoster(demoId, roundNumber) as Promise<PlayerRosterEntry[]>
}
