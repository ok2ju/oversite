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
  if (side === "CT") {
    return active
      ? "border-sky-400 bg-sky-500 text-white"
      : "border-sky-400/70 text-sky-300 hover:bg-sky-500/20"
  }
  if (side === "T") {
    return active
      ? "border-amber-400 bg-amber-500 text-white"
      : "border-amber-400/70 text-amber-300 hover:bg-amber-500/20"
  }
  return active
    ? "border-gray-400 bg-gray-500 text-white"
    : "border-gray-400/70 text-gray-300 hover:bg-gray-500/20"
}

export function RoundSelector() {
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

  return (
    <div
      data-testid="round-selector"
      className="pointer-events-none absolute bottom-16 left-4 right-[180px] z-10 flex justify-center"
    >
      <div className="pointer-events-auto flex max-w-full items-center gap-1.5 overflow-x-auto rounded-lg bg-black/40 px-2 py-1.5 backdrop-blur-sm">
        {rounds.map((round, i) => {
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
        })}
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
        className={`flex h-7 min-w-[1.75rem] shrink-0 items-center justify-center rounded-md border-2 px-1.5 text-xs font-semibold tabular-nums transition-colors ${pillClasses(round.winner_side, isActive)}`}
      >
        {round.round_number}
      </button>
    </div>
  )
})
