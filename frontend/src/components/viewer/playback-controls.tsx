"use client"

import { useCallback, useMemo } from "react"
import { Play, Pause } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { formatElapsedTime } from "@/lib/viewer/timeline-utils"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu"
import { Timeline } from "./timeline"
import type { Round } from "@/types/round"

const SPEED_OPTIONS = [0.25, 0.5, 1, 2, 4] as const

function getActiveRound(rounds: Round[], currentTick: number): Round | null {
  for (let i = rounds.length - 1; i >= 0; i--) {
    if (currentTick >= rounds[i].start_tick) {
      return rounds[i]
    }
  }
  return rounds[0] ?? null
}

export function PlaybackControls() {
  const isPlaying = useViewerStore((s) => s.isPlaying)
  const currentTick = useViewerStore((s) => s.currentTick)
  const totalTicks = useViewerStore((s) => s.totalTicks)
  const speed = useViewerStore((s) => s.speed)
  const demoId = useViewerStore((s) => s.demoId)
  const tickRate = useViewerStore((s) => s.tickRate)
  const togglePlay = useViewerStore((s) => s.togglePlay)
  const setSpeed = useViewerStore((s) => s.setSpeed)
  const setTick = useViewerStore((s) => s.setTick)
  const pause = useViewerStore((s) => s.pause)

  const { data: rounds } = useRounds(demoId)

  const activeRound = useMemo(
    () => (rounds?.length ? getActiveRound(rounds, currentTick) : null),
    [rounds, currentTick],
  )

  // Timeline is scoped to the live portion of the round only (freeze time is
  // auto-skipped by the playback engine). Fall back to start_tick when
  // freeze_end_tick is missing (older parser output).
  const roundStartTick = activeRound
    ? activeRound.freeze_end_tick > 0
      ? activeRound.freeze_end_tick
      : activeRound.start_tick
    : 0
  const roundEndTick = activeRound?.end_tick ?? totalTicks
  const roundTotalTicks = Math.max(0, roundEndTick - roundStartTick)
  const roundCurrentTick = Math.max(
    0,
    Math.min(currentTick - roundStartTick, roundTotalTicks),
  )

  const handleSeek = useCallback(
    (tick: number) => {
      setTick(tick + roundStartTick)
    },
    [setTick, roundStartTick],
  )

  const handleScrubStart = useCallback(() => {
    pause()
  }, [pause])

  if (totalTicks === 0) return null

  return (
    <div
      data-testid="playback-controls"
      className="hud-panel absolute bottom-4 left-4 right-[180px] flex items-center gap-3 rounded-lg px-3 py-2"
    >
      {/* Play/Pause — luminous accent when playing */}
      <Button
        variant="ghost"
        size="icon"
        onClick={togglePlay}
        aria-label={isPlaying ? "Pause" : "Play"}
        className={cn(
          "relative h-9 w-9 shrink-0 rounded-full text-white ring-1 ring-inset ring-white/15 transition-all",
          isPlaying
            ? "bg-orange-500/15 text-orange-200 ring-orange-400/40 shadow-[0_0_22px_-2px_rgba(255,122,26,0.55)] hover:bg-orange-500/25"
            : "bg-white/[0.04] hover:bg-white/10",
        )}
      >
        {isPlaying ? (
          <Pause size={15} className="fill-current" />
        ) : (
          <Play size={15} className="ml-[1px] fill-current" />
        )}
      </Button>

      {/* Speed selector */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            data-testid="speed-trigger"
            className="hud-callsign h-7 shrink-0 rounded-md px-2 text-[10px] font-semibold text-white/80 ring-1 ring-inset ring-white/10 hover:bg-white/10 hover:text-white"
          >
            {speed}x
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="min-w-[4rem]">
          {SPEED_OPTIONS.map((s) => (
            <DropdownMenuItem key={s} onClick={() => setSpeed(s)}>
              {s}x
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Vertical divider */}
      <span aria-hidden="true" className="h-7 w-px bg-white/10" />

      {/* Timeline */}
      <div className="min-w-0 flex-1">
        <Timeline
          currentTick={roundCurrentTick}
          totalTicks={roundTotalTicks}
          roundBoundaries={[]}
          onSeek={handleSeek}
          onScrubStart={handleScrubStart}
        />
      </div>

      {/* Round clock — display font */}
      <div className="shrink-0 whitespace-nowrap text-right">
        <div className="hud-callsign text-[9px] text-white/40">ROUND TIME</div>
        <span
          data-testid="round-time"
          className="hud-display text-[15px] font-semibold leading-none tabular-nums text-white"
        >
          {formatElapsedTime(roundCurrentTick, tickRate)}
        </span>
      </div>
    </div>
  )
}
