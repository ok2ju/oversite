"use client"

import { memo, useCallback, useMemo } from "react"
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
import { RoundSelector } from "./round-selector"
import { RoundTimeline } from "./round-timeline/round-timeline"
import { FilterBar } from "./round-timeline/filter-bar"
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

const PlaybackHeader = memo(function PlaybackHeader({
  activeRound,
}: {
  activeRound: Round | null
}) {
  const isPlaying = useViewerStore((s) => s.isPlaying)
  const speed = useViewerStore((s) => s.speed)
  const currentTick = useViewerStore((s) => s.currentTick)
  const tickRate = useViewerStore((s) => s.tickRate)
  const togglePlay = useViewerStore((s) => s.togglePlay)
  const setSpeed = useViewerStore((s) => s.setSpeed)

  const roundStart = activeRound
    ? activeRound.freeze_end_tick > 0
      ? activeRound.freeze_end_tick
      : activeRound.start_tick
    : 0
  const roundEnd = activeRound?.end_tick ?? 0
  const elapsed = Math.max(
    0,
    Math.min(currentTick - roundStart, roundEnd - roundStart),
  )
  const total = Math.max(0, roundEnd - roundStart)

  return (
    <div className="flex items-center gap-2 px-2 py-1.5">
      <Button
        variant="ghost"
        size="icon"
        onClick={togglePlay}
        aria-label={isPlaying ? "Pause" : "Play"}
        data-testid="playback-dock-play"
        className={cn(
          "relative h-8 w-8 shrink-0 rounded-full text-white ring-1 ring-inset ring-white/15 transition-all",
          isPlaying
            ? "bg-orange-500/15 text-orange-200 ring-orange-400/40 shadow-[0_0_22px_-2px_rgba(255,122,26,0.55)] hover:bg-orange-500/25"
            : "bg-white/[0.04] hover:bg-white/10",
        )}
      >
        {isPlaying ? (
          <Pause size={14} className="fill-current" />
        ) : (
          <Play size={14} className="ml-[1px] fill-current" />
        )}
      </Button>

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

      <span aria-hidden="true" className="h-6 w-px bg-white/10" />

      <div className="hud-callsign flex items-center gap-2 text-[10px] text-white/55">
        {activeRound ? (
          <>
            <span className="text-white/80">
              Round {activeRound.round_number}
            </span>
            <span className="tabular-nums" data-testid="round-time">
              {formatElapsedTime(elapsed, tickRate)} /{" "}
              {formatElapsedTime(total, tickRate)}
            </span>
          </>
        ) : (
          <span>—</span>
        )}
      </div>

      <div className="ml-auto">
        <FilterBar />
      </div>
    </div>
  )
})

// PlaybackDock — composite container that bundles the round selector pills,
// a slim controls strip (play / speed / clock + filter chips), and the rich
// round timeline (lanes + spine + mistakes + playhead). Replaces the legacy
// separate <PlaybackControls /> + <RoundSelector /> anchors.
export function PlaybackDock() {
  const demoId = useViewerStore((s) => s.demoId)
  const currentTick = useViewerStore((s) => s.currentTick)
  const totalTicks = useViewerStore((s) => s.totalTicks)
  const { data: rounds } = useRounds(demoId)

  const activeRound = useMemo(
    () => (rounds?.length ? getActiveRound(rounds, currentTick) : null),
    [rounds, currentTick],
  )

  // Prevent clicks inside the dock from bubbling into the canvas (PixiJS pan
  // interaction listens on the parent).
  const stop = useCallback((e: React.SyntheticEvent) => {
    e.stopPropagation()
  }, [])

  if (totalTicks === 0) return null

  return (
    <div
      data-testid="playback-dock"
      onMouseDown={stop}
      onPointerDown={stop}
      onClick={stop}
      className="hud-panel pointer-events-auto absolute bottom-4 left-4 right-4 flex flex-col rounded-lg"
    >
      <div className="border-b border-white/[0.06] px-2 pt-1.5 pb-1">
        <RoundSelector variant="embedded" />
      </div>
      <PlaybackHeader activeRound={activeRound} />
      {activeRound ? (
        <div className="px-2 pb-2">
          <RoundTimeline round={activeRound} />
        </div>
      ) : null}
    </div>
  )
}
