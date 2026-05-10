import { useViewerStore } from "@/stores/viewer"
import { useMistakeTimeline } from "@/hooks/use-mistake-timeline"
import { useRounds } from "@/hooks/use-rounds"
import type { MistakeEntry } from "@/types/mistake"
import type { Round } from "@/types/round"

// Human-readable label per mistake kind. Falls back to the raw kind string for
// kinds the frontend hasn't been taught about yet — keeps a future Go-only
// rule from rendering as a blank row.
const KIND_LABEL: Record<string, string> = {
  no_trade_death: "Untraded death",
}

function formatClock(
  tick: number,
  round: Round | undefined,
  tickRate: number,
): string {
  if (!round || tickRate <= 0) return "?:??"
  // Prefer freeze_end_tick (the round-live tick) so the clock matches what the
  // player remembers seeing in-game. Older imports may still have 0 there;
  // fall back to start_tick in that case.
  const base =
    round.freeze_end_tick > 0 ? round.freeze_end_tick : round.start_tick
  const secs = Math.max(0, Math.round((tick - base) / tickRate))
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return `${m}:${s.toString().padStart(2, "0")}`
}

function formatMistakeText(
  entry: MistakeEntry,
  rounds: Round[] | undefined,
  tickRate: number,
): string {
  const label = KIND_LABEL[entry.kind] ?? entry.kind
  const round = rounds?.find((r) => r.round_number === entry.round_number)
  return `${label} — round ${entry.round_number}, ${formatClock(
    entry.tick,
    round,
    tickRate,
  )}`
}

interface MistakeListProps {
  // Optional override for the player whose mistakes to render. When omitted
  // (the default), the list reads selectedPlayerSteamId from useViewerStore —
  // matching the player-stats-panel.tsx convention.
  steamId?: string | null
}

export function MistakeList({ steamId: steamIdProp }: MistakeListProps = {}) {
  const demoId = useViewerStore((s) => s.demoId)
  const selectedSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const tickRate = useViewerStore((s) => s.tickRate)
  const steamId = steamIdProp ?? selectedSteamId
  const { data: rounds } = useRounds(demoId)
  const { data: mistakes, isLoading } = useMistakeTimeline(demoId, steamId)

  if (!steamId) return null

  const items = mistakes ?? []

  return (
    <aside
      data-testid="mistake-list"
      className="absolute left-0 top-0 z-30 flex h-full w-72 flex-col border-r border-white/10 bg-black/85 text-white shadow-2xl backdrop-blur"
    >
      <header className="border-b border-white/10 px-3 py-2 text-xs font-semibold uppercase tracking-wide text-white/70">
        Mistakes
      </header>
      <div className="flex-1 overflow-y-auto p-2">
        {isLoading ? (
          <p
            data-testid="mistake-list-loading"
            className="text-sm text-white/60"
          >
            Loading mistakes…
          </p>
        ) : items.length === 0 ? (
          <p data-testid="mistake-list-empty" className="text-sm text-white/60">
            No mistakes
          </p>
        ) : (
          <ul className="space-y-1">
            {items.map((m, i) => (
              <li
                key={`${m.kind}-${m.tick}-${i}`}
                data-testid={`mistake-list-row-${i}`}
                className="rounded border border-white/10 bg-white/5 px-2 py-1 text-sm tabular-nums"
              >
                {formatMistakeText(m, rounds, tickRate)}
              </li>
            ))}
          </ul>
        )}
      </div>
    </aside>
  )
}
