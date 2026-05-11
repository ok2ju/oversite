"use client"

import { memo, useCallback, useMemo } from "react"
import { Play, Pause, SkipBack, SkipForward } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
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
// nothing is. Mirrors the design's "george" tag attached to the round
// picker row.
function FocalPlayerChip({ player }: { player: PlayerRosterEntry | null }) {
  if (!player) {
    return (
      <span
        data-testid="focal-player-chip"
        className="hud-callsign inline-flex h-5 items-center rounded-[3px] bg-white/[0.06] px-2 text-[10px] font-semibold text-white/55"
      >
        ALL PLAYERS
      </span>
    )
  }
  const isT = player.team_side === "T"
  const initials = player.player_name.slice(0, 2).toUpperCase()
  return (
    <span
      data-testid="focal-player-chip"
      data-player-side={player.team_side}
      className={cn(
        "inline-flex h-5 items-center gap-1.5 rounded-[3px] pl-0.5 pr-2",
        isT ? "bg-amber-400" : "bg-sky-400",
      )}
    >
      <span className="flex h-[18px] w-[18px] items-center justify-center rounded-[2px] bg-black/30 font-mono text-[9px] font-bold text-white">
        {initials}
      </span>
      <span className="font-mono text-[10px] font-bold tracking-[0.02em] text-black">
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
      className="flex items-center gap-2 px-1 pt-0.5 pb-1.5 font-mono text-[11px] text-white/55"
      data-testid="playback-transport"
    >
      <div className="inline-flex items-center gap-px">
        <button
          type="button"
          onClick={handlePrev}
          aria-label="Step back 5 seconds"
          data-testid="playback-prev"
          className="inline-flex h-5 w-5 items-center justify-center rounded-[3px] bg-white/[0.05] text-white/80 transition-colors hover:bg-white/15"
        >
          <SkipBack size={10} className="fill-current" />
        </button>
        <button
          type="button"
          onClick={togglePlay}
          aria-label={isPlaying ? "Pause" : "Play"}
          data-testid="playback-dock-play"
          className={cn(
            "inline-flex h-5 w-5 items-center justify-center rounded-[3px] transition-colors",
            isPlaying
              ? "bg-orange-400 text-black hover:bg-orange-300"
              : "bg-white text-black hover:bg-white/90",
          )}
        >
          {isPlaying ? (
            <Pause size={9} className="fill-current" />
          ) : (
            <Play size={9} className="ml-px fill-current" />
          )}
        </button>
        <button
          type="button"
          onClick={handleNext}
          aria-label="Step forward 5 seconds"
          data-testid="playback-next"
          className="inline-flex h-5 w-5 items-center justify-center rounded-[3px] bg-white/[0.05] text-white/80 transition-colors hover:bg-white/15"
        >
          <SkipForward size={10} className="fill-current" />
        </button>
      </div>

      <select
        data-testid="speed-trigger"
        aria-label="Playback speed"
        value={speed}
        onChange={(e) => setSpeed(Number(e.target.value))}
        className="h-5 w-[52px] cursor-pointer appearance-none rounded-[3px] border border-white/10 bg-white/[0.04] py-0 pl-1.5 pr-3.5 font-mono text-[10px] tabular-nums text-white/85 outline-none transition-colors hover:bg-white/10 focus:border-white/25"
        style={{
          backgroundImage:
            "url(\"data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='8' height='6' viewBox='0 0 8 6'><path fill='%23a0a3aa' d='M0 0h8L4 6z'/></svg>\")",
          backgroundRepeat: "no-repeat",
          backgroundPosition: "right 4px center",
        }}
      >
        {SPEED_OPTIONS.map((s) => (
          <option key={s} value={s}>
            {s}×
          </option>
        ))}
      </select>

      {activeRound ? (
        <>
          <span className="tabular-nums text-white/85" data-testid="round-time">
            {formatElapsedTime(elapsed, tickRate)}
          </span>
          <span className="text-white/30">·</span>
          <span className="text-white/55">
            ROUND {activeRound.round_number}
          </span>
        </>
      ) : (
        <span>—</span>
      )}

      <span className="ml-auto tabular-nums text-white/55">
        {formatElapsedTime(total, tickRate)}
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
      className="hud-panel pointer-events-auto absolute bottom-4 left-4 right-4 flex flex-col rounded-lg px-3 pt-2 pb-2"
    >
      {/* Row 1 — focal player chip + round picker + side-win legend */}
      <div className="flex items-center gap-2.5 pb-2">
        <FocalPlayerChip player={focalPlayer} />
        <RoundSelector variant="embedded" />
        <div
          aria-hidden="true"
          className="ml-auto flex items-center gap-3 font-mono text-[10px] text-white/55"
        >
          <span className="inline-flex items-center gap-1.5">
            <span className="inline-block h-[3px] w-2.5 bg-amber-400" />T win
          </span>
          <span className="inline-flex items-center gap-1.5">
            <span className="inline-block h-[3px] w-2.5 bg-sky-400" />
            CT win
          </span>
        </div>
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
