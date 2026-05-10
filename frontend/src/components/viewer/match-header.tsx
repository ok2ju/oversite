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

  const roundTime = formatRoundTime(
    header.roundTicks,
    tickRate,
    header.freezeDurationTicks,
  )

  return (
    <div
      data-testid="match-header"
      className="pointer-events-none absolute left-1/2 top-3 z-10 flex -translate-x-1/2 items-stretch whitespace-nowrap"
    >
      {/* CT plate */}
      <div className="hud-panel hud-stripe-ct flex items-center gap-3 rounded-l-md py-1.5 pl-4 pr-5">
        <span
          aria-hidden="true"
          className="h-1.5 w-1.5 rotate-45 bg-sky-400 shadow-[0_0_8px_2px_rgba(56,189,248,0.65)]"
        />
        <span
          data-testid="match-header-team-ct"
          className="hud-callsign text-[11px] font-semibold text-sky-300"
        >
          {ctTeam}
        </span>
        <span
          data-testid="match-header-ct-score"
          className="hud-display text-[28px] font-semibold leading-none tabular-nums text-sky-200"
          style={{ textShadow: "0 0 18px rgba(56,189,248,0.45)" }}
        >
          {String(header.ctScore).padStart(2, "0")}
        </span>
      </div>

      {/* Center clock */}
      <div className="relative flex flex-col items-center justify-center bg-black/85 px-5 py-1 ring-1 ring-inset ring-white/10">
        <span className="hud-callsign text-[9px] text-white/45">
          ROUND {String(header.roundNumber).padStart(2, "0")}
        </span>
        <span
          data-testid="match-header-round-time"
          className="font-mono text-[15px] font-semibold leading-tight tabular-nums text-white"
        >
          {roundTime}
        </span>
        {/* tiny score divider dot */}
        <span
          aria-hidden="true"
          className="absolute -top-1 left-1/2 h-1 w-px -translate-x-1/2 bg-white/40"
        />
      </div>

      {/* T plate */}
      <div className="hud-panel hud-stripe-t flex items-center gap-3 rounded-r-md py-1.5 pl-5 pr-4">
        <span
          data-testid="match-header-t-score"
          className="hud-display text-[28px] font-semibold leading-none tabular-nums text-orange-200"
          style={{ textShadow: "0 0 18px rgba(251,146,60,0.45)" }}
        >
          {String(header.tScore).padStart(2, "0")}
        </span>
        <span
          data-testid="match-header-team-t"
          className="hud-callsign text-[11px] font-semibold text-orange-300"
        >
          {tTeam}
        </span>
        <span
          aria-hidden="true"
          className="h-1.5 w-1.5 rotate-45 bg-orange-400 shadow-[0_0_8px_2px_rgba(251,146,60,0.65)]"
        />
      </div>
    </div>
  )
}
