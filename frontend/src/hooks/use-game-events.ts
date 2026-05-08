import { useQuery } from "@tanstack/react-query"
import { GetDemoEvents, GetEventsByTypes } from "@wailsjs/go/main/App"
import type { GameEvent } from "@/types/demo"

// The wails type generator sees Go's json.RawMessage as []byte and emits
// `extra_data: number[]` even though the runtime wire format is the same
// JSON object as before — RawMessage.MarshalJSON emits the raw bytes verbatim,
// not a byte array. The local GameEvent type (Record<string, unknown> | null)
// reflects the actual shape, so cast through unknown to silence the spurious
// generator-side type mismatch without losing inference at the use sites.
//
// `enabled` lets a consumer disable the observer after it has handed the data
// off to a long-lived owner (e.g. PixiJS EventLayer); without it, the viewer
// would keep two retained copies of the same ~1.6 MB payload — one in the
// TanStack cache and one inside the layer.
export function useGameEvents(demoId: string | null, enabled = true) {
  return useQuery({
    queryKey: ["game-events", demoId],
    queryFn: () => GetDemoEvents(demoId!) as unknown as Promise<GameEvent[]>,
    enabled: !!demoId && enabled,
    staleTime: Infinity,
  })
}

// Stable key so unrelated kill-feed call sites share the same query cache
// entry even though the JS reference for the literal array would otherwise
// differ. Kept as a module-level constant to avoid re-allocating on every
// hook call.
const KILL_FEED_TYPES = ["kill"] as const

// useKillFeed fetches only kill events for a demo, used by the kill-log
// overlay. Bypasses the full game-events payload (and per-row extra_data
// JSON decode) for non-kill rows the kill-log never reads.
export function useKillFeed(demoId: string | null) {
  return useQuery({
    queryKey: ["game-events", demoId, "kill-feed"],
    queryFn: () =>
      GetEventsByTypes(
        demoId!,
        KILL_FEED_TYPES as unknown as string[],
      ) as unknown as Promise<GameEvent[]>,
    enabled: !!demoId,
    staleTime: Infinity,
  })
}
