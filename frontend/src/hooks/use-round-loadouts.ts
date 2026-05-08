import { useQuery } from "@tanstack/react-query"
import { GetRoundLoadouts } from "@wailsjs/go/main/App"
import type { RoundLoadoutEntry } from "@/types/demo"

// Wire shape: round_number -> entries with comma-separated inventory string.
type WireResponse = Record<
  number,
  Array<{ steam_id: string; inventory: string }>
>

// Splits the wire-format CSV inventory into a string[] once on receipt and
// keys the result by `round_number → steam_id` for O(1) team-bars lookups.
// Empty inventory ('' from Go) maps to [], not [""].
export type LoadoutByRound = Record<number, Record<string, string[]>>

// useRoundLoadouts preloads every round's freeze-end loadout for a demo in a
// single Wails call. Migration 011 moved inventory off tick_data so the
// team-bars consumer (use-loadout-snapshot) reads the round-scoped loadout
// instead of polling per-tick. ~250 rows total; staleTime is infinity since
// freeze-end loadouts don't change after parse.
export function useRoundLoadouts(demoId: string | null) {
  return useQuery({
    queryKey: ["round-loadouts", demoId],
    queryFn: async () => {
      const wire = (await GetRoundLoadouts(demoId!)) as unknown as WireResponse
      const result: LoadoutByRound = {}
      for (const [roundStr, entries] of Object.entries(wire)) {
        const round = Number(roundStr)
        const byPlayer: Record<string, string[]> = {}
        for (const e of entries as Array<
          RoundLoadoutEntry & { inventory: string | string[] }
        >) {
          // Defensive: tolerate either wire format (string from Go,
          // already-split array if a future caller ever pre-splits) so
          // hot-reload during migration doesn't surface as a runtime error.
          const inv =
            typeof e.inventory === "string"
              ? e.inventory
                ? e.inventory.split(",")
                : []
              : e.inventory
          byPlayer[e.steam_id] = inv
        }
        result[round] = byPlayer
      }
      return result
    },
    enabled: !!demoId,
    staleTime: Infinity,
  })
}
