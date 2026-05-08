import { useQuery } from "@tanstack/react-query"
import { GetAllRosters, GetRoundRoster } from "@wailsjs/go/main/App"
import type { PlayerRosterEntry } from "@/types/roster"

// One demo's full per-round roster, keyed by round_number. Returned by
// GetAllRosters so consumers can resolve any round locally without a
// per-round Wails round-trip.
export type RosterByRound = Record<number, PlayerRosterEntry[]>

const EMPTY_ROSTER: PlayerRosterEntry[] = []

// useAllRosters preloads every round's roster for a demo in a single Wails
// call, used by the viewer at demo open. Round transitions read from this
// cache via fetchRoster()/useRoundRoster() — whoever resolves first wins,
// and the per-round fallback only fires for rounds that aren't in the map
// (e.g. mid-parse races).
export function useAllRosters(demoId: string | null) {
  return useQuery({
    queryKey: ["all-rosters", demoId],
    queryFn: () => GetAllRosters(demoId!) as unknown as Promise<RosterByRound>,
    enabled: !!demoId,
    staleTime: Infinity,
  })
}

// fetchRoster is an imperative fetch function — used in the PixiJS lifecycle
// (similar to TickBuffer) rather than as a TanStack Query hook, because roster
// data is loaded in response to PixiJS subscription events, not React renders.
//
// Resolves from the in-memory map populated by useAllRosters when present;
// otherwise falls back to the per-round binding so a cold viewer load (where
// the all-rosters fetch hasn't resolved yet) still paints.
export async function fetchRoster(
  demoId: string,
  roundNumber: number,
  signal: AbortSignal,
  rosters?: RosterByRound,
): Promise<PlayerRosterEntry[]> {
  void signal
  if (rosters && rosters[roundNumber]) return rosters[roundNumber]
  if (rosters) return EMPTY_ROSTER
  return GetRoundRoster(demoId, roundNumber) as Promise<PlayerRosterEntry[]>
}

export function useRoundRoster(
  demoId: string | null,
  roundNumber: number | null,
) {
  return useQuery({
    queryKey: ["round-roster", demoId, roundNumber],
    queryFn: () =>
      GetRoundRoster(demoId!, roundNumber!) as Promise<PlayerRosterEntry[]>,
    enabled: !!demoId && roundNumber != null,
    staleTime: Infinity,
  })
}
