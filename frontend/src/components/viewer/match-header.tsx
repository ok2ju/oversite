import { useMemo } from "react"
import { Link } from "react-router-dom"
import { ArrowLeft } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { useRoundRoster } from "@/hooks/use-roster"
import { formatRoundTime } from "@/lib/viewer/timeline-utils"
import type { Round } from "@/types/round"
import type { PlayerRosterEntry } from "@/types/roster"
import { DemoRouteTabs } from "./demo-route-tabs"

function getActiveRoundIndex(rounds: Round[], currentTick: number): number {
  for (let i = rounds.length - 1; i >= 0; i--) {
    if (currentTick >= rounds[i].start_tick) return i
  }
  return 0
}

// Per-round clan name when the demo carries one; otherwise the per-round
// roster's first member; otherwise the side letter. Pro / FACEIT / ESEA demos
// populate ct_team_name; matchmaking demos leave it empty.
function teamLabel(
  clanName: string,
  entries: PlayerRosterEntry[] | undefined,
  side: "CT" | "T",
): string {
  if (clanName) return clanName
  const first = entries?.find((e) => e.team_side === side)
  if (first) return `team_${first.player_name}`
  return side
}

export function MatchHeader() {
  const demoId = useViewerStore((s) => s.demoId)
  const currentTick = useViewerStore((s) => s.currentTick)
  const tickRate = useViewerStore((s) => s.tickRate)
  const mapName = useViewerStore((s) => s.mapName)

  const { data: rounds } = useRounds(demoId)

  const header = useMemo(() => {
    if (!rounds?.length) return null
    const idx = getActiveRoundIndex(rounds, currentTick)
    const active = rounds[idx]
    const prev = idx > 0 ? rounds[idx - 1] : null
    const freezeDurationTicks = Math.max(
      0,
      active.freeze_end_tick - active.start_tick,
    )
    return {
      roundNumber: active.round_number,
      totalRounds: rounds.length,
      ctScore: prev?.ct_score ?? 0,
      tScore: prev?.t_score ?? 0,
      ctTeamName: active.ct_team_name,
      tTeamName: active.t_team_name,
      roundTicks: Math.max(0, currentTick - active.start_tick),
      freezeDurationTicks,
    }
  }, [rounds, currentTick])

  const { data: roster } = useRoundRoster(demoId, header?.roundNumber ?? null)

  if (!demoId || !header) return null

  const ctTeam = teamLabel(header.ctTeamName, roster, "CT")
  const tTeam = teamLabel(header.tTeamName, roster, "T")

  const roundTime = formatRoundTime(
    header.roundTicks,
    tickRate,
    header.freezeDurationTicks,
  )

  return (
    <div
      data-testid="match-header"
      className="pointer-events-auto absolute inset-x-0 top-0 z-30 flex h-[46px] items-center gap-4 border-b border-white/[0.06] bg-[#0b0d10]/95 px-4 backdrop-blur-md"
    >
      <Link
        to="/demos"
        data-testid="match-header-back"
        aria-label="Back to demos"
        title="Back to demos"
        className="inline-flex h-7 w-7 items-center justify-center rounded-md text-white/65 transition-colors hover:bg-white/10 hover:text-white focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-white/30"
      >
        <ArrowLeft className="h-4 w-4" />
      </Link>

      <span aria-hidden="true" className="h-[22px] w-px bg-white/10" />

      {/* Brand lockup */}
      <span className="hud-callsign whitespace-nowrap text-[11px] text-white/45">
        OVERSITE · DEMO
      </span>

      <span aria-hidden="true" className="h-[22px] w-px bg-white/10" />

      {/* Scoreboard: T (gold) — vs — CT (blue) */}
      <div className="flex items-center gap-2.5 whitespace-nowrap">
        <span className="flex items-center gap-1.5">
          <span
            aria-hidden="true"
            className="h-2 w-2 rounded-sm bg-amber-400 shadow-[0_0_6px_1px_rgba(251,191,36,0.55)]"
          />
          <span
            data-testid="match-header-team-t"
            className="font-mono text-[12px] text-white/65"
          >
            {tTeam}
          </span>
          <span
            data-testid="match-header-t-score"
            className="ml-1 font-mono text-[18px] font-bold leading-none tabular-nums text-white"
          >
            {header.tScore}
          </span>
        </span>
        <span className="font-mono text-[12px] text-white/35">vs</span>
        <span className="flex items-center gap-1.5">
          <span
            data-testid="match-header-ct-score"
            className="font-mono text-[18px] font-bold leading-none tabular-nums text-white"
          >
            {header.ctScore}
          </span>
          <span
            data-testid="match-header-team-ct"
            className="ml-1 font-mono text-[12px] text-white/65"
          >
            {ctTeam}
          </span>
          <span
            aria-hidden="true"
            className="h-2 w-2 rounded-sm bg-sky-400 shadow-[0_0_6px_1px_rgba(56,189,248,0.55)]"
          />
        </span>
      </div>

      <span aria-hidden="true" className="h-[22px] w-px bg-white/10" />

      {/* Match meta */}
      <div className="flex items-center gap-2 whitespace-nowrap font-mono text-[12px] text-white/55">
        <span>MAP</span>
        <span className="text-white/85">{mapName ?? "—"}</span>
        <span className="ml-3">RND</span>
        <span className="text-white/85 tabular-nums">
          {String(header.roundNumber).padStart(2, "0")}/
          {String(header.totalRounds).padStart(2, "0")}
        </span>
        <span className="ml-3">CLOCK</span>
        <span
          data-testid="match-header-round-time"
          className="text-white/85 tabular-nums"
        >
          {roundTime}
        </span>
        <span className="ml-3">TICK</span>
        <span className="text-white/85 tabular-nums">
          {currentTick.toLocaleString()}
        </span>
      </div>

      {/* Route tabs */}
      <div className="ml-auto">
        <DemoRouteTabs demoId={demoId} />
      </div>
    </div>
  )
}
