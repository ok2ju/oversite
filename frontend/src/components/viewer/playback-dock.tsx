"use client"

import { memo, useCallback, useMemo } from "react"
import { Play, Pause, SkipBack, SkipForward } from "lucide-react"
import {
  useViewerStore,
  MIN_TIMELINE_ZOOM,
  MAX_TIMELINE_ZOOM,
} from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { useRoundRoster } from "@/hooks/use-roster"
import { formatElapsedTime } from "@/lib/viewer/timeline-utils"
import { cn } from "@/lib/utils"
import { RoundSelector } from "./round-selector"
import { RoundTimeline } from "./round-timeline/round-timeline"
import type { Round } from "@/types/round"
import type { PlayerRosterEntry } from "@/types/roster"

const SPEED_OPTIONS = [0.25, 0.5, 1, 2, 4, 8] as const

function getActiveRound(rounds: Round[], currentTick: number): Round | null {
  for (let i = rounds.length - 1; i >= 0; i--) {
    if (currentTick >= rounds[i].start_tick) {
      return rounds[i]
    }
  }
  return rounds[0] ?? null
}

function findPlayer(
  roster: PlayerRosterEntry[] | undefined,
  steamId: string | null,
): PlayerRosterEntry | null {
  if (!roster || !steamId) return null
  return roster.find((p) => p.steam_id === steamId) ?? null
}

// Focal player chip — gold pill when a player is selected, neutral chip when
// nothing is. Mirrors the design's player tag attached to the round picker row.
function FocalPlayerChip({ player }: { player: PlayerRosterEntry | null }) {
  if (!player) {
    return (
      <span
        data-testid="focal-player-chip"
        className="inline-flex h-6 items-center rounded-[4px] bg-white/[0.06] px-2.5 text-[10.5px] font-medium text-white/60"
      >
        All players
      </span>
    )
  }
  const isT = player.team_side === "T"
  return (
    <span
      data-testid="focal-player-chip"
      data-player-side={player.team_side}
      className={cn(
        "inline-flex h-6 shrink-0 items-center gap-1.5 rounded-[4px] px-2",
        isT ? "bg-amber-400" : "bg-sky-400",
      )}
    >
      <span
        aria-hidden="true"
        className="h-1.5 w-1.5 rounded-full bg-black/55"
      />
      <span className="text-[11.5px] font-semibold leading-none text-black">
        {player.player_name}
      </span>
    </span>
  )
}

// Bottom legend strip — explains the event tile vocabulary used inside the
// timeline track (duel won/lost, grenade, plant/defuse, mistakes).
function TimelineLegend() {
  const items: Array<{
    label: string
    swatch: React.ReactNode
    className?: string
  }> = [
    {
      label: "duel won",
      swatch: <span className="h-2.5 w-2.5 rounded-[2px] bg-amber-400" />,
    },
    {
      label: "duel lost",
      swatch: <span className="h-2.5 w-2.5 rounded-[2px] bg-sky-400" />,
    },
    {
      label: "grenade",
      swatch: (
        <span className="h-2.5 w-2.5 rounded-[2px] border border-amber-300/80 bg-white/[0.04]" />
      ),
    },
    {
      label: "plant / defuse",
      swatch: <span className="h-2.5 w-2.5 rounded-[2px] bg-orange-400" />,
    },
    {
      label: "has mistakes",
      swatch: <span className="h-2 w-2 rounded-full bg-rose-400" />,
      className: "text-rose-300/80",
    },
  ]
  return (
    <div
      data-testid="timeline-legend"
      className="flex items-center gap-3.5 px-1 pt-1.5 pb-0.5 font-mono text-[10px] text-white/55"
    >
      {items.map((it) => (
        <span
          key={it.label}
          className={cn("inline-flex items-center gap-1.5", it.className)}
        >
          {it.swatch}
          {it.label}
        </span>
      ))}
      <span className="ml-auto text-white/40">Click any event for details</span>
    </div>
  )
}

const TransportRow = memo(function TransportRow({
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
  const setTick = useViewerStore((s) => s.setTick)
  const pause = useViewerStore((s) => s.pause)
  const timelineZoom = useViewerStore((s) => s.timelineZoom)
  const zoomTimelineIn = useViewerStore((s) => s.zoomTimelineIn)
  const zoomTimelineOut = useViewerStore((s) => s.zoomTimelineOut)
  const canZoomIn = timelineZoom < MAX_TIMELINE_ZOOM - 1e-6
  const canZoomOut = timelineZoom > MIN_TIMELINE_ZOOM + 1e-6
  const zoomLabel = Number.isInteger(timelineZoom)
    ? `${timelineZoom}×`
    : `${timelineZoom.toFixed(1)}×`

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

  // Coarse seek-by-5s for prev/next, scoped to the active round window. The
  // keyboard `,` / `.` shortcuts handle precise event navigation; these
  // buttons keep the design's transport idiom available to mouse users.
  const stepSeconds = 5
  const stepTicks = stepSeconds * tickRate
  const handlePrev = useCallback(() => {
    pause()
    setTick(Math.max(roundStart, currentTick - stepTicks))
  }, [pause, setTick, currentTick, roundStart, stepTicks])
  const handleNext = useCallback(() => {
    pause()
    setTick(Math.min(roundEnd, currentTick + stepTicks))
  }, [pause, setTick, currentTick, roundEnd, stepTicks])

  return (
    <div
      className="flex items-center gap-3 pb-2 pt-1.5 text-[11.5px] text-white/55"
      data-testid="playback-transport"
    >
      <div className="inline-flex items-center gap-1">
        <button
          type="button"
          onClick={handlePrev}
          aria-label="Step back 5 seconds"
          data-testid="playback-prev"
          className="inline-flex h-6 w-6 items-center justify-center rounded-[4px] text-white/70 transition-colors hover:bg-white/[0.08] hover:text-white"
        >
          <SkipBack size={12} className="fill-current" />
        </button>
        <button
          type="button"
          onClick={togglePlay}
          aria-label={isPlaying ? "Pause" : "Play"}
          data-testid="playback-dock-play"
          className={cn(
            "inline-flex h-6 w-6 items-center justify-center rounded-[4px] transition-colors",
            isPlaying
              ? "bg-amber-400 text-black hover:bg-amber-300"
              : "bg-white text-black hover:bg-white/90",
          )}
        >
          {isPlaying ? (
            <Pause size={11} className="fill-current" />
          ) : (
            <Play size={11} className="ml-0.5 fill-current" />
          )}
        </button>
        <button
          type="button"
          onClick={handleNext}
          aria-label="Step forward 5 seconds"
          data-testid="playback-next"
          className="inline-flex h-6 w-6 items-center justify-center rounded-[4px] text-white/70 transition-colors hover:bg-white/[0.08] hover:text-white"
        >
          <SkipForward size={12} className="fill-current" />
        </button>
      </div>

      <select
        data-testid="speed-trigger"
        aria-label="Playback speed"
        value={speed}
        onChange={(e) => setSpeed(Number(e.target.value))}
        className="h-6 w-[56px] cursor-pointer appearance-none rounded-[4px] border border-white/10 bg-white/[0.04] py-0 pl-2 pr-4 text-[11px] tabular-nums text-white/85 outline-none transition-colors hover:bg-white/[0.08] focus:border-white/25"
        style={{
          backgroundImage:
            "url(\"data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='8' height='6' viewBox='0 0 8 6'><path fill='%23a0a3aa' d='M0 0h8L4 6z'/></svg>\")",
          backgroundRepeat: "no-repeat",
          backgroundPosition: "right 5px center",
        }}
      >
        {SPEED_OPTIONS.map((s) => (
          <option key={s} value={s}>
            {s}×
          </option>
        ))}
      </select>

      {activeRound ? (
        <div className="flex items-baseline gap-1.5 leading-none">
          <span
            className="font-semibold tabular-nums text-white"
            data-testid="round-time"
          >
            {formatElapsedTime(elapsed, tickRate)}
          </span>
          <span className="text-white/25">·</span>
          <span className="text-white/55">
            Round {activeRound.round_number}
          </span>
          <span className="text-white/35 tabular-nums">
            {formatElapsedTime(total, tickRate)}
          </span>
        </div>
      ) : (
        <span>—</span>
      )}

      <span
        className="ml-auto inline-flex items-center gap-1 text-white/55"
        data-testid="timeline-zoom"
      >
        <span
          className="w-7 text-right text-[11px] tabular-nums"
          data-testid="timeline-zoom-level"
        >
          {zoomLabel}
        </span>
        <button
          type="button"
          aria-label="Zoom timeline out"
          data-testid="timeline-zoom-out"
          onClick={zoomTimelineOut}
          disabled={!canZoomOut}
          className="inline-flex h-5 w-5 items-center justify-center rounded text-white/55 transition-colors hover:bg-white/[0.06] hover:text-white disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-white/55"
        >
          −
        </button>
        <button
          type="button"
          aria-label="Zoom timeline in"
          data-testid="timeline-zoom-in"
          onClick={zoomTimelineIn}
          disabled={!canZoomIn}
          className="inline-flex h-5 w-5 items-center justify-center rounded text-white/55 transition-colors hover:bg-white/[0.06] hover:text-white disabled:cursor-not-allowed disabled:opacity-30 disabled:hover:bg-transparent disabled:hover:text-white/55"
        >
          +
        </button>
      </span>
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
  const selectedPlayerSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const { data: rounds } = useRounds(demoId)

  const activeRound = useMemo(
    () => (rounds?.length ? getActiveRound(rounds, currentTick) : null),
    [rounds, currentTick],
  )

  const { data: roster } = useRoundRoster(
    demoId,
    activeRound?.round_number ?? null,
  )

  const focalPlayer = useMemo(
    () => findPlayer(roster, selectedPlayerSteamId),
    [roster, selectedPlayerSteamId],
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
      className="pointer-events-auto absolute bottom-4 left-4 right-4 flex flex-col gap-1 rounded-lg border border-white/[0.06] bg-[#0e1115]/95 px-4 pb-3 pt-2.5 backdrop-blur-md"
    >
      {/* Row 1 — focal player chip + round picker */}
      <div className="flex items-center gap-2.5">
        <FocalPlayerChip player={focalPlayer} />
        <RoundSelector variant="embedded" />
      </div>

      {/* Row 2 — compact transport, speed select, round clock */}
      <TransportRow activeRound={activeRound} />

      {/* Row 3 — the rich timeline track itself (lanes / spine / playhead) */}
      {activeRound ? <RoundTimeline round={activeRound} /> : null}

      {/* Row 4 — tile-type legend */}
      <TimelineLegend />
    </div>
  )
}
