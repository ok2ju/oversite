import { Card } from "@/components/ui/card"
import type { Round } from "@/types/round"

export type RoundClass = "win-t" | "win-ct" | "loss-t" | "loss-ct"

export function roundClass(
  round: Pick<Round, "winner_side">,
  mySide: "CT" | "T" = "CT",
): RoundClass {
  const won = round.winner_side === mySide
  const sideCharged = won ? mySide : round.winner_side
  const prefix = won ? "win" : "loss"
  const sideLower = sideCharged === "CT" ? "ct" : "t"
  return `${prefix}-${sideLower}` as RoundClass
}

const COLOR: Record<RoundClass, { bg: string; fg: string }> = {
  "win-t": { bg: "var(--round-win-t)", fg: "#1e5a3a" },
  "win-ct": { bg: "var(--round-win-ct)", fg: "var(--accent-ink)" },
  "loss-t": { bg: "var(--round-loss-t)", fg: "#9a1d2d" },
  "loss-ct": { bg: "var(--round-loss-ct)", fg: "#9a1d2d" },
}

const LEGEND: Array<{ key: RoundClass; label: string }> = [
  { key: "win-t", label: "Win T" },
  { key: "win-ct", label: "Win CT" },
  { key: "loss-t", label: "Loss T" },
  { key: "loss-ct", label: "Loss CT" },
]

interface RoundTimelineProps {
  rounds: Round[]
  mySide?: "CT" | "T"
}

export function RoundTimeline({ rounds, mySide = "CT" }: RoundTimelineProps) {
  return (
    <Card className="border border-[var(--border)] bg-[var(--bg-elevated)] p-4">
      <div className="flex items-center justify-between">
        <div className="text-[13.5px] font-bold text-[var(--text)]">
          Round timeline
        </div>
        <div className="flex items-center gap-3">
          {LEGEND.map(({ key, label }) => (
            <div
              key={key}
              className="flex items-center gap-1.5 text-[11px] text-[var(--text-muted)]"
            >
              <span
                className="h-2.5 w-2.5 rounded-sm"
                style={{ background: COLOR[key].bg }}
              />
              {label}
            </div>
          ))}
        </div>
      </div>

      <div
        className="mt-3 grid gap-1"
        style={{ gridTemplateColumns: "repeat(30, minmax(0, 1fr))" }}
      >
        {rounds.map((r) => {
          const cls = roundClass(r, mySide)
          const color = COLOR[cls]
          return (
            <div
              key={r.id}
              data-testid={`round-cell-${r.round_number}`}
              data-class={cls}
              title={`Round ${r.round_number}: ${r.win_reason}`}
              className="grid h-[26px] place-items-center rounded-sm font-mono text-[9px] font-semibold"
              style={{ background: color.bg, color: color.fg }}
            >
              {r.round_number}
            </div>
          )
        })}
      </div>
    </Card>
  )
}
