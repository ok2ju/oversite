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
      className="pointer-events-auto absolute inset-x-0 top-0 z-30 flex h-[48px] items-center gap-6 border-b border-white/[0.07] bg-[#14171c] px-5"
    >
      <Link
        to="/demos"
        data-testid="match-header-back"
        aria-label="Back to demos"
        title="Back to demos"
        className="inline-flex items-center gap-2 rounded-md px-1.5 py-1 text-[12.5px] text-white/55 transition-colors hover:bg-white/[0.06] hover:text-white focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-white/30"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        <span>Demo</span>
      </Link>

      {/* Scoreboard: T (gold) — vs — CT (blue) */}
      <div className="flex items-baseline gap-2.5 whitespace-nowrap">
        <span className="flex items-baseline gap-1.5">
          <span
            aria-hidden="true"
            className="relative top-[-1px] inline-block h-1.5 w-1.5 self-center rounded-full bg-amber-400"
          />
          <span
            data-testid="match-header-team-t"
            className="text-[12.5px] text-white/70"
          >
            {tTeam}
          </span>
          <span
            data-testid="match-header-t-score"
            className="font-sans text-[14px] font-semibold leading-none tabular-nums text-white"
          >
            {header.tScore}
          </span>
        </span>
        <span className="text-[11.5px] text-white/30">vs</span>
        <span className="flex items-baseline gap-1.5">
          <span
            data-testid="match-header-ct-score"
            className="font-sans text-[14px] font-semibold leading-none tabular-nums text-white"
          >
            {header.ctScore}
          </span>
          <span
            data-testid="match-header-team-ct"
            className="text-[12.5px] text-white/70"
          >
            {ctTeam}
          </span>
          <span
            aria-hidden="true"
            className="relative top-[-1px] inline-block h-1.5 w-1.5 self-center rounded-full bg-sky-400"
          />
        </span>
      </div>

      {/* Match meta */}
      <div className="flex items-baseline gap-5 whitespace-nowrap text-[11.5px]">
        <MetaItem label="Map" value={mapName ?? "—"} />
        <MetaItem
          label="Round"
          value={`${String(header.roundNumber).padStart(2, "0")}/${String(
            header.totalRounds,
          ).padStart(2, "0")}`}
        />
        <MetaItem
          label="Clock"
          value={roundTime}
          testId="match-header-round-time"
        />
        <MetaItem label="Tick" value={currentTick.toLocaleString()} />
      </div>

      {/* Route tabs */}
      <div className="ml-auto">
        <DemoRouteTabs demoId={demoId} />
      </div>
    </div>
  )
}

function MetaItem({
  label,
  value,
  testId,
}: {
  label: string
  value: string
  testId?: string
}) {
  return (
    <span className="flex items-baseline gap-1.5">
      <span className="text-white/40">{label}</span>
      <span
        data-testid={testId}
        className="font-sans font-medium tabular-nums text-white/90"
      >
        {value}
      </span>
    </span>
  )
}
