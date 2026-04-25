import { Card } from "@/components/ui/card"
import type { ScoreboardEntry } from "@/types/scoreboard"

export function ratingClass(rating: number): "hi" | "lo" | "mid" {
  if (rating >= 1.1) return "hi"
  if (rating < 0.9) return "lo"
  return "mid"
}

export function tierColor(level: number): string {
  if (level >= 10) return "var(--tier-10)"
  if (level >= 8) return "var(--tier-8)"
  if (level >= 6) return "var(--tier-6)"
  return "var(--tier-5)"
}

function approximateLevel(adr: number): number {
  // Fallback when the roster has no explicit level. ADR is a rough proxy:
  // keeps the tier badge non-empty until the backend exposes Faceit levels.
  if (adr >= 100) return 10
  if (adr >= 80) return 8
  if (adr >= 60) return 6
  return 4
}

function approximateRating(entry: ScoreboardEntry): number {
  const kpr = entry.rounds_played > 0 ? entry.kills / entry.rounds_played : 0
  const survival =
    entry.rounds_played > 0 ? 1 - entry.deaths / entry.rounds_played : 0
  return 0.3591 * kpr + 0.2372 * survival + 0.0032 * entry.adr + 0.2
}

interface TeamBlockProps {
  label: string
  players: ScoreboardEntry[]
  meSteamId?: string | null
}

function TeamBlock({ label, players, meSteamId }: TeamBlockProps) {
  return (
    <div
      className="overflow-hidden rounded-md border"
      style={{ borderColor: "var(--border)" }}
    >
      <div
        className="px-4 py-2 text-[10.5px] font-semibold uppercase tracking-wider text-[var(--text-muted)]"
        style={{ background: "var(--bg-sunken)" }}
      >
        {label}
      </div>
      <div
        className="grid items-center px-4 py-2 text-[10.5px] font-semibold uppercase tracking-wider text-[var(--text-subtle)]"
        style={{
          gridTemplateColumns: "minmax(0,1fr) repeat(6, 56px) 60px",
        }}
      >
        <div>Player</div>
        <div className="text-right">K</div>
        <div className="text-right">A</div>
        <div className="text-right">D</div>
        <div className="text-right">HS%</div>
        <div className="text-right">ADR</div>
        <div className="text-right">KAST%</div>
        <div className="text-right">Rating</div>
      </div>
      {players.map((p) => {
        const level = approximateLevel(p.adr)
        const rating = approximateRating(p)
        const cls = ratingClass(rating)
        const isMe = meSteamId === p.steam_id
        return (
          <div
            key={p.steam_id}
            data-testid={`player-${p.steam_id}`}
            className="grid items-center border-t px-4 py-2.5 text-[12.5px]"
            style={{
              gridTemplateColumns: "minmax(0,1fr) repeat(6, 56px) 60px",
              borderColor: "var(--divider)",
              background: isMe ? "var(--accent-soft)" : undefined,
            }}
          >
            <div className="flex min-w-0 items-center gap-2.5">
              <div
                className="grid h-[26px] w-[26px] place-items-center rounded-[4px] text-[11px] font-bold text-white"
                style={{
                  background: "linear-gradient(135deg, #4a5058, #1a1d22)",
                }}
                aria-hidden
              >
                {p.player_name[0]?.toUpperCase() ?? "?"}
              </div>
              <div
                className="grid h-4 w-4 place-items-center rounded-[3px] text-[9px] font-bold text-white"
                style={{ background: tierColor(level) }}
                aria-label={`Level ${level}`}
              >
                {level}
              </div>
              <span className="truncate font-semibold text-[var(--text)]">
                {p.player_name}
              </span>
              {isMe ? (
                <span
                  className="rounded-sm px-1.5 py-0.5 text-[9.5px] font-bold tracking-wider uppercase"
                  style={{
                    background: "var(--accent)",
                    color: "#0b0d10",
                  }}
                >
                  You
                </span>
              ) : null}
            </div>
            <div className="tabular text-right text-[var(--text)]">
              {p.kills}
            </div>
            <div className="tabular text-right text-[var(--text-muted)]">
              {p.assists}
            </div>
            <div className="tabular text-right text-[var(--text-muted)]">
              {p.deaths}
            </div>
            <div className="tabular text-right text-[var(--text-muted)]">
              {Math.round(p.hs_percent)}
            </div>
            <div className="tabular text-right text-[var(--text-muted)]">
              {p.adr.toFixed(1)}
            </div>
            <div className="tabular text-right text-[var(--text-muted)]">—</div>
            <div
              className="tabular text-right font-semibold"
              style={{
                color:
                  cls === "hi"
                    ? "var(--win)"
                    : cls === "lo"
                      ? "var(--loss)"
                      : "var(--text)",
              }}
              data-rating-class={cls}
            >
              {rating.toFixed(2)}
            </div>
          </div>
        )
      })}
    </div>
  )
}

interface ScoreboardTableProps {
  entries: ScoreboardEntry[]
  meSteamId?: string | null
  myTeamLabel?: string
  enemyTeamLabel?: string
}

export function ScoreboardTable({
  entries,
  meSteamId = null,
  myTeamLabel = "Your team",
  enemyTeamLabel = "Enemy team",
}: ScoreboardTableProps) {
  const me = meSteamId
    ? entries.find((e) => e.steam_id === meSteamId)
    : undefined
  const mySide = me?.team_side ?? "CT"
  const mine = entries.filter((e) => e.team_side === mySide)
  const enemies = entries.filter((e) => e.team_side !== mySide)

  return (
    <Card className="border border-[var(--border)] bg-[var(--bg-elevated)] p-0">
      <div className="divide-y divide-[var(--divider)]">
        <TeamBlock label={myTeamLabel} players={mine} meSteamId={meSteamId} />
        <TeamBlock label={enemyTeamLabel} players={enemies} />
      </div>
    </Card>
  )
}
