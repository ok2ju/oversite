import { Card } from "@/components/ui/card"
import { resolveMap } from "@/components/dashboard/map-tile"

interface TeamSummary {
  name: string
  score: number
  playerCount: number
  premade?: boolean
}

interface MatchHeroProps {
  mapName: string
  mode?: string
  roundCount?: number
  durationSecs?: number
  left: TeamSummary
  right: TeamSummary
}

function formatDuration(secs: number): string {
  if (!secs) return ""
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return `${m}:${s.toString().padStart(2, "0")}`
}

function TeamLogo({
  gradient,
  initial,
}: {
  gradient: string
  initial: string
}) {
  return (
    <div
      className="grid h-11 w-11 place-items-center rounded-[10px] text-[14px] font-bold text-white"
      style={{ background: gradient }}
    >
      {initial}
    </div>
  )
}

export function MatchHero({
  mapName,
  mode = "5v5",
  roundCount,
  durationSecs,
  left,
  right,
}: MatchHeroProps) {
  const map = resolveMap(mapName)
  const ribbon = [
    mode,
    map.name,
    roundCount ? `${roundCount} rounds` : null,
    durationSecs ? formatDuration(durationSecs) : null,
  ]
    .filter(Boolean)
    .join(" · ")

  const leftWin = left.score > right.score
  const rightWin = right.score > left.score

  return (
    <Card className="border border-[var(--border)] bg-[var(--bg-elevated)] p-5">
      <div
        className="text-center text-[10.5px] font-semibold uppercase tracking-wider text-[var(--text-subtle)]"
        data-testid="match-hero-ribbon"
      >
        {ribbon}
      </div>
      <div
        className="mt-3 grid items-center gap-6"
        style={{ gridTemplateColumns: "1fr auto 1fr" }}
      >
        <div className="flex items-center gap-3">
          <TeamLogo
            gradient="linear-gradient(135deg, var(--accent), var(--accent-gradient-end))"
            initial={left.name[0]?.toUpperCase() ?? "?"}
          />
          <div>
            <div className="text-[16px] font-bold text-[var(--text)]">
              {left.name}
            </div>
            <div className="text-[12px] text-[var(--text-muted)]">
              {left.playerCount} players{left.premade ? " · Premade" : ""}
            </div>
          </div>
        </div>

        <div className="tabular flex items-baseline gap-3 px-6 text-[56px] font-bold leading-none">
          <span style={{ color: leftWin ? "var(--win)" : "var(--text)" }}>
            {left.score}
          </span>
          <span className="text-[28px] text-[var(--text-faint)]">:</span>
          <span style={{ color: rightWin ? "var(--win)" : "var(--text)" }}>
            {right.score}
          </span>
        </div>

        <div className="flex items-center justify-end gap-3">
          <div className="text-right">
            <div className="text-[16px] font-bold text-[var(--text)]">
              {right.name}
            </div>
            <div className="text-[12px] text-[var(--text-muted)]">
              {right.playerCount} players
            </div>
          </div>
          <TeamLogo
            gradient="linear-gradient(135deg, #6b7280, #1f2937)"
            initial={right.name[0]?.toUpperCase() ?? "?"}
          />
        </div>
      </div>
    </Card>
  )
}
