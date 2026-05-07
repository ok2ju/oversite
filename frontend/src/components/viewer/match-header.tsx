import { useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { useRoundRoster } from "@/hooks/use-roster"
import { formatRoundTime } from "@/lib/viewer/timeline-utils"
import type { Round } from "@/types/round"
import type { PlayerRosterEntry } from "@/types/roster"

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

  return (
    <div
      data-testid="match-header"
      className="pointer-events-none absolute left-1/2 top-4 z-10 flex -translate-x-1/2 items-center gap-8 whitespace-nowrap rounded-lg bg-black/40 px-6 py-2 backdrop-blur-sm"
    >
      <span
        data-testid="match-header-team-ct"
        className="text-lg font-bold text-sky-400"
      >
        {ctTeam}
      </span>
      <div className="flex flex-col items-center">
        <div className="flex items-center gap-2 text-xl font-bold tabular-nums">
          <span data-testid="match-header-ct-score" className="text-sky-400">
            {header.ctScore}
          </span>
          <span className="text-white/60">-</span>
          <span data-testid="match-header-t-score" className="text-amber-400">
            {header.tScore}
          </span>
        </div>
        <span
          data-testid="match-header-round-time"
          className="text-sm tabular-nums text-white/80"
        >
          {formatRoundTime(
            header.roundTicks,
            tickRate,
            header.freezeDurationTicks,
          )}
        </span>
      </div>
      <span
        data-testid="match-header-team-t"
        className="text-lg font-bold text-amber-400"
      >
        {tTeam}
      </span>
    </div>
  )
}
