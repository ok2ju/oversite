import { useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import type { Round } from "@/types/round"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

function getCurrentRound(
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

function winnerColor(side: string): string {
  if (side === "CT") return "text-sky-400"
  if (side === "T") return "text-amber-400"
  return "text-muted-foreground"
}

export function RoundSelector() {
  const demoId = useViewerStore((s) => s.demoId)
  const currentTick = useViewerStore((s) => s.currentTick)
  const setRound = useViewerStore((s) => s.setRound)
  const setTick = useViewerStore((s) => s.setTick)

  const { data: rounds, isLoading } = useRounds(demoId)

  const activeRound = useMemo(
    () => (rounds?.length ? getCurrentRound(rounds, currentTick) : undefined),
    [rounds, currentTick],
  )

  if (!demoId || isLoading || !rounds?.length) return null

  const handleSelect = (value: string) => {
    const roundNumber = Number(value)
    const round = rounds.find((r) => r.round_number === roundNumber)
    if (!round) return
    setRound(roundNumber)
    setTick(round.start_tick)
  }

  return (
    <div
      data-testid="round-selector"
      className="absolute right-4 top-4 z-10 w-52"
    >
      <Select
        value={activeRound != null ? String(activeRound) : undefined}
        onValueChange={handleSelect}
      >
        <SelectTrigger className="border-white/20 bg-black/60 text-sm text-white backdrop-blur-sm hover:bg-black/70">
          <SelectValue placeholder="Select round" />
        </SelectTrigger>
        <SelectContent>
          {rounds.map((round) => (
            <SelectItem
              key={round.id}
              value={String(round.round_number)}
              data-testid={`round-option-${round.round_number}`}
            >
              <span className="flex items-center gap-2">
                <span
                  className={`inline-block h-2 w-2 rounded-full ${
                    round.winner_side === "CT"
                      ? "bg-sky-400"
                      : round.winner_side === "T"
                        ? "bg-amber-400"
                        : "bg-gray-400"
                  }`}
                />
                <span>
                  Round {round.round_number}:{" "}
                  <span className={winnerColor(round.winner_side)}>
                    {round.ct_score}
                  </span>
                  -
                  <span className={winnerColor(round.winner_side)}>
                    {round.t_score}
                  </span>
                </span>
                {round.is_overtime && (
                  <span className="text-xs text-purple-400">OT</span>
                )}
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
