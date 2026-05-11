import { useCallback } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useAnalysisStore } from "@/stores/analysis"
import { cn } from "@/lib/utils"
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip"
import { formatElapsedTime } from "@/lib/viewer/timeline-utils"
import type { MistakeMarker } from "@/lib/timeline/types"

interface MistakesLaneProps {
  mistakes: MistakeMarker[]
  roundStartTick: number
  roundEndTick: number
  // When false, render the "Select a player…" placeholder banner instead.
  hasPlayer: boolean
}

const SEVERITY_CLASS: Record<number, string> = {
  1: "bg-yellow-400/70 ring-yellow-300/40",
  2: "bg-orange-400/80 ring-orange-300/40",
  3: "bg-red-500/85 ring-red-300/40",
}

function position(tick: number, start: number, end: number): number {
  const span = Math.max(1, end - start)
  return Math.max(0, Math.min(1, (tick - start) / span))
}

export function MistakesLane({
  mistakes,
  roundStartTick,
  roundEndTick,
  hasPlayer,
}: MistakesLaneProps) {
  const tickRate = useViewerStore((s) => s.tickRate)
  const setTick = useViewerStore((s) => s.setTick)
  const pause = useViewerStore((s) => s.pause)
  const setSelectedMistakeId = useAnalysisStore((s) => s.setSelectedMistakeId)

  const handleClick = useCallback(
    (m: MistakeMarker) => {
      pause()
      setTick(m.tick)
      setSelectedMistakeId(m.id || null)
    },
    [pause, setTick, setSelectedMistakeId],
  )

  if (!hasPlayer) {
    return (
      <div
        data-testid="round-timeline-mistakes-placeholder"
        className="relative flex h-5 items-center justify-center rounded-sm bg-white/[0.02] text-[10px] text-white/40"
      >
        Select a player to see mistakes for this round
      </div>
    )
  }

  return (
    <div
      data-testid="round-timeline-mistakes"
      className="relative h-5 rounded-sm bg-white/[0.02]"
      aria-label="Mistakes timeline"
    >
      <span
        aria-hidden="true"
        className="hud-callsign pointer-events-none absolute left-1.5 top-1/2 -translate-y-1/2 text-[9px] font-semibold tracking-wider text-white/50"
      >
        ISSUES
      </span>
      {mistakes.map((m) => {
        const pos = position(m.tick, roundStartTick, roundEndTick)
        const sev = SEVERITY_CLASS[m.severity] ?? SEVERITY_CLASS[1]
        return (
          <Tooltip key={`${m.id}-${m.tick}`}>
            <TooltipTrigger asChild>
              <button
                type="button"
                data-testid={`mistake-marker-${m.id || m.tick}`}
                onClick={() => handleClick(m)}
                aria-label={`${m.title || m.kind} at ${formatElapsedTime(m.tick - roundStartTick, tickRate)}`}
                className={cn(
                  "absolute top-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rotate-45 cursor-pointer rounded-sm ring-1 ring-inset transition-transform hover:scale-125 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-400/50",
                  sev,
                )}
                style={{ left: `${pos * 100}%` }}
              />
            </TooltipTrigger>
            <TooltipContent side="bottom" align="center">
              <div className="font-semibold">{m.title || m.kind}</div>
              <div className="text-white/50">
                @ {formatElapsedTime(m.tick - roundStartTick, tickRate)}
              </div>
            </TooltipContent>
          </Tooltip>
        )
      })}
    </div>
  )
}
