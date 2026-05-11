import { useCallback } from "react"
import { useViewerStore } from "@/stores/viewer"
import { cn } from "@/lib/utils"
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip"
import { formatElapsedTime } from "@/lib/viewer/timeline-utils"
import type { DuelEntry } from "@/types/duel"
import type { MistakeEntry } from "@/types/mistake"

interface DuelsLaneProps {
  duels: DuelEntry[]
  mistakes: MistakeEntry[]
  roundStartTick: number
  roundEndTick: number
  roundNumber: number
  selectedPlayerSteamId: string | null
  hasPlayer: boolean
}

const SEVERITY_DOT: Record<number, string> = {
  1: "bg-yellow-400/80",
  2: "bg-orange-400/85",
  3: "bg-red-500/90",
}

function clamp01(value: number): number {
  return Math.max(0, Math.min(1, value))
}

function bandPosition(
  startTick: number,
  endTick: number,
  roundStart: number,
  roundEnd: number,
): { left: number; width: number } {
  const span = Math.max(1, roundEnd - roundStart)
  const left = clamp01((startTick - roundStart) / span)
  const right = clamp01((endTick - roundStart) / span)
  // Minimum perceptual width — single-tick clean-kill duels would
  // otherwise render as 0-width bands and disappear.
  const width = Math.max(0.005, right - left)
  return { left, width }
}

function outcomeGlyph(outcome: DuelEntry["outcome"]): string {
  switch (outcome) {
    case "won":
      return "✓"
    case "lost":
      return "✗"
    case "won_then_traded":
    case "lost_but_traded":
      return "⇄"
    default:
      return "·"
  }
}

// DuelsLane is the per-player engagement track that sits between the
// mistakes lane and the kill log on the round-timeline. Bands are tinted
// blue when the selected player was the attacker (offensive duel), red
// when victim (defensive duel). Clicking a band seeks the playhead to
// the duel's start; hovering / focusing opens a tooltip with the
// mistakes inside it.
export function DuelsLane({
  duels,
  mistakes,
  roundStartTick,
  roundEndTick,
  roundNumber,
  selectedPlayerSteamId,
  hasPlayer,
}: DuelsLaneProps) {
  const tickRate = useViewerStore((s) => s.tickRate)
  const setTick = useViewerStore((s) => s.setTick)
  const pause = useViewerStore((s) => s.pause)

  const handleClick = useCallback(
    (d: DuelEntry) => {
      pause()
      setTick(d.start_tick)
    },
    [pause, setTick],
  )

  if (!hasPlayer) {
    return (
      <div
        data-testid="round-timeline-duels-placeholder"
        className="relative flex h-5 items-center justify-center rounded-sm bg-white/[0.02] text-[10px] text-white/40"
      >
        Select a player to see duels for this round
      </div>
    )
  }

  // Round-scoped filter — duel rows arrive for every round in the demo.
  const inRound = duels.filter((d) => d.round_number === roundNumber)

  return (
    <div
      data-testid="round-timeline-duels"
      className="relative h-5 rounded-sm bg-white/[0.02]"
      aria-label="Duels timeline"
    >
      <span
        aria-hidden="true"
        className="hud-callsign pointer-events-none absolute left-1.5 top-1/2 -translate-y-1/2 text-[9px] font-semibold tracking-wider text-white/50"
      >
        DUELS
      </span>
      {inRound.map((d) => {
        const isAttacker = selectedPlayerSteamId === d.attacker_steam
        const { left, width } = bandPosition(
          d.start_tick,
          d.end_tick,
          roundStartTick,
          roundEndTick,
        )
        const tint = isAttacker
          ? "bg-sky-500/30 ring-sky-400/40"
          : "bg-rose-500/30 ring-rose-400/40"
        const duelMistakes = mistakes.filter(
          (m) =>
            m.duel_id === d.id &&
            (m.steam_id === selectedPlayerSteamId || isAttacker),
        )
        return (
          <Tooltip key={d.id}>
            <TooltipTrigger asChild>
              <button
                type="button"
                data-testid={`duel-band-${d.id}`}
                onClick={() => handleClick(d)}
                aria-label={`Duel ${isAttacker ? "vs" : "from"} enemy at ${formatElapsedTime(d.start_tick - roundStartTick, tickRate)}`}
                className={cn(
                  "absolute top-1/2 flex h-3 -translate-y-1/2 cursor-pointer items-center justify-between gap-0.5 rounded-sm px-1 ring-1 ring-inset transition-transform hover:scale-y-110 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-400/50",
                  tint,
                )}
                style={{
                  left: `${left * 100}%`,
                  width: `${width * 100}%`,
                  minWidth: "8px",
                }}
              >
                {duelMistakes.slice(0, 3).map((m) => (
                  <span
                    key={m.id}
                    aria-hidden="true"
                    className={cn(
                      "h-1.5 w-1.5 shrink-0 rounded-full",
                      SEVERITY_DOT[m.severity] ?? SEVERITY_DOT[1],
                    )}
                  />
                ))}
                <span
                  aria-hidden="true"
                  className="ml-auto text-[10px] leading-none text-white/80"
                >
                  {outcomeGlyph(d.outcome)}
                </span>
                {d.mutual_duel_id != null && (
                  <span
                    aria-hidden="true"
                    title="Mutual engagement"
                    className="text-[10px] leading-none text-white/60"
                  >
                    ⇋
                  </span>
                )}
              </button>
            </TooltipTrigger>
            <TooltipContent side="bottom" align="center">
              <div className="font-semibold">
                {isAttacker ? "Your duel" : "Defended"} ·{" "}
                {outcomeLabel(d.outcome)}
              </div>
              <div className="text-white/50">
                @ {formatElapsedTime(d.start_tick - roundStartTick, tickRate)}
                {" · "}
                {d.shot_count} shot{d.shot_count === 1 ? "" : "s"}
                {d.hurt_count > 0 ? ` · ${d.hurt_count} hit` : ""}
              </div>
              {duelMistakes.length > 0 && (
                <ul className="mt-1 list-disc space-y-0.5 pl-3 text-white/70">
                  {duelMistakes.map((m) => (
                    <li key={m.id}>{m.title || m.kind}</li>
                  ))}
                </ul>
              )}
            </TooltipContent>
          </Tooltip>
        )
      })}
    </div>
  )
}

function outcomeLabel(outcome: DuelEntry["outcome"]): string {
  switch (outcome) {
    case "won":
      return "Won"
    case "lost":
      return "Lost"
    case "won_then_traded":
      return "Won, then traded"
    case "lost_but_traded":
      return "Lost, but traded"
    case "inconclusive":
      return "Inconclusive"
    default:
      return outcome
  }
}
