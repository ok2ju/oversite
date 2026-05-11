import { memo, useCallback, useMemo } from "react"
import { ArrowLeftRight, Clock } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import type { Round } from "@/types/round"

function getActiveRound(
  rounds: Round[],
  currentTick: number,
): number | undefined {
  for (let i = rounds.length - 1; i >= 0; i--) {
    if (currentTick >= rounds[i].start_tick) {
      return rounds[i].round_number
    }
  }
  return rounds[0]?.round_number
}

type MarkerKind = "halftime" | "ot-start" | "ot-swap"

function markerBetween(prev: Round, next: Round): MarkerKind | null {
  if (!prev.is_overtime && next.is_overtime) return "ot-start"
  if (!prev.is_overtime && prev.round_number === 12) return "halftime"
  if (
    prev.is_overtime &&
    next.is_overtime &&
    (prev.round_number - 24) % 3 === 0
  ) {
    return "ot-swap"
  }
  return null
}

function pillClasses(side: string, active: boolean): string {
  // Active pills are filled with the side accent and glow gently. Inactive
  // pills are hollow with a thin bottom border in the side color so the
  // win/loss cadence reads at a glance even before hovering. The "sky" /
  // "amber" substrings are preserved for downstream tests.
  if (side === "CT") {
    return active
      ? "border-sky-400/70 bg-sky-400/90 text-white shadow-[0_0_10px_-1px_rgba(56,189,248,0.7)]"
      : "border-transparent border-b-sky-400/60 bg-white/[0.03] text-sky-300/85 hover:bg-sky-400/15 hover:text-sky-100"
  }
  if (side === "T") {
    return active
      ? "border-amber-400/70 bg-amber-400/90 text-white shadow-[0_0_10px_-1px_rgba(251,191,36,0.7)]"
      : "border-transparent border-b-amber-400/60 bg-white/[0.03] text-amber-300/85 hover:bg-amber-400/15 hover:text-amber-100"
  }
  return active
    ? "border-gray-400 bg-gray-500 text-white"
    : "border-gray-400/70 text-gray-300 hover:bg-gray-500/20"
}

interface RoundSelectorProps {
  // "panel" (default) keeps the absolute-positioned standalone HUD chrome.
  // "embedded" strips it so the strip can live inside another container
  // (e.g. PlaybackDock).
  variant?: "panel" | "embedded"
}

export function RoundSelector({ variant = "panel" }: RoundSelectorProps = {}) {
  const demoId = useViewerStore((s) => s.demoId)
  const currentTick = useViewerStore((s) => s.currentTick)
  const setRound = useViewerStore((s) => s.setRound)
  const setTick = useViewerStore((s) => s.setTick)

  const { data: rounds, isLoading } = useRounds(demoId)

  const activeRound = useMemo(
    () => (rounds?.length ? getActiveRound(rounds, currentTick) : undefined),
    [rounds, currentTick],
  )

  const handleSelect = useCallback(
    (round: Round) => {
      setRound(round.round_number)
      // Skip freeze time — seek to the live start of the round.
      setTick(
        round.freeze_end_tick > 0 ? round.freeze_end_tick : round.start_tick,
      )
    },
    [setRound, setTick],
  )

  if (!demoId || isLoading || !rounds?.length) return null

  const pills = rounds.map((round, i) => {
    const isActive = activeRound === round.round_number
    const prev = i > 0 ? rounds[i - 1] : null
    const marker = prev ? markerBetween(prev, round) : null
    return (
      <RoundPill
        key={round.id}
        round={round}
        isActive={isActive}
        marker={marker}
        onSelect={handleSelect}
      />
    )
  })

  if (variant === "embedded") {
    return (
      <div
        data-testid="round-selector"
        className="flex max-w-full items-center gap-1.5 overflow-x-auto"
      >
        {pills}
      </div>
    )
  }

  return (
    <div
      data-testid="round-selector"
      className="pointer-events-none absolute bottom-16 left-4 right-[180px] z-10 flex justify-center"
    >
      <div className="hud-panel pointer-events-auto flex max-w-full items-center gap-1.5 overflow-x-auto rounded-md px-2.5 py-1.5">
        {pills}
      </div>
    </div>
  )
}

const RoundPill = memo(function RoundPill({
  round,
  isActive,
  marker,
  onSelect,
}: {
  round: Round
  isActive: boolean
  marker: MarkerKind | null
  onSelect: (round: Round) => void
}) {
  return (
    <div className="flex items-center gap-1.5">
      {marker === "halftime" || marker === "ot-swap" ? (
        <ArrowLeftRight
          size={14}
          className="shrink-0 text-white/50"
          data-testid={`round-marker-${marker}-${round.round_number}`}
          aria-label={marker === "halftime" ? "Halftime" : "Overtime side swap"}
        />
      ) : null}
      {marker === "ot-start" ? (
        <Clock
          size={14}
          className="shrink-0 text-white/50"
          data-testid={`round-marker-ot-start-${round.round_number}`}
          aria-label="Overtime start"
        />
      ) : null}
      <button
        type="button"
        data-testid={`round-pill-${round.round_number}`}
        aria-label={`Round ${round.round_number}`}
        aria-current={isActive ? "true" : undefined}
        onClick={() => onSelect(round)}
        className={`hud-display flex h-7 min-w-[1.75rem] shrink-0 items-center justify-center rounded-md border-2 px-1.5 text-[12px] font-semibold tabular-nums transition-all duration-150 ${pillClasses(round.winner_side, isActive)}`}
      >
        {round.round_number}
      </button>
    </div>
  )
})
