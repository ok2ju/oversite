"use client"

import { useCallback } from "react"
import { Play, Pause } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { formatTickDisplay } from "@/lib/viewer/timeline-utils"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu"
import { Timeline } from "./timeline"

const SPEED_OPTIONS = [0.25, 0.5, 1, 2, 4] as const

export function PlaybackControls() {
  const isPlaying = useViewerStore((s) => s.isPlaying)
  const currentTick = useViewerStore((s) => s.currentTick)
  const totalTicks = useViewerStore((s) => s.totalTicks)
  const speed = useViewerStore((s) => s.speed)
  const demoId = useViewerStore((s) => s.demoId)
  const togglePlay = useViewerStore((s) => s.togglePlay)
  const setSpeed = useViewerStore((s) => s.setSpeed)
  const setTick = useViewerStore((s) => s.setTick)
  const pause = useViewerStore((s) => s.pause)

  const { data } = useRounds(demoId)

  const roundBoundaries =
    data?.data.map((r) => ({
      roundNumber: r.round_number,
      startTick: r.start_tick,
      endTick: r.end_tick,
    })) ?? []

  const handleScrubStart = useCallback(() => {
    pause()
  }, [pause])

  if (totalTicks === 0) return null

  return (
    <div
      data-testid="playback-controls"
      className="absolute bottom-4 left-4 right-[180px] flex items-center gap-3 rounded-lg border border-white/20 bg-black/60 px-3 py-2 backdrop-blur-sm"
    >
      {/* Play/Pause */}
      <Button
        variant="ghost"
        size="icon"
        onClick={togglePlay}
        aria-label={isPlaying ? "Pause" : "Play"}
        className="h-8 w-8 shrink-0 text-white hover:bg-white/10"
      >
        {isPlaying ? <Pause size={16} /> : <Play size={16} />}
      </Button>

      {/* Speed selector */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            data-testid="speed-trigger"
            className="h-8 shrink-0 text-xs text-white hover:bg-white/10"
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

      {/* Timeline */}
      <div className="min-w-0 flex-1">
        <Timeline
          currentTick={currentTick}
          totalTicks={totalTicks}
          roundBoundaries={roundBoundaries}
          onSeek={setTick}
          onScrubStart={handleScrubStart}
        />
      </div>

      {/* Tick counter */}
      <span
        data-testid="tick-counter"
        className="shrink-0 whitespace-nowrap text-xs tabular-nums text-white/70"
      >
        {formatTickDisplay(currentTick, totalTicks)}
      </span>
    </div>
  )
}
