import { useMatchInsights } from "@/hooks/use-match-insights"
import { Skeleton } from "@/components/ui/skeleton"
import type { TeamSummary } from "@/types/analysis"

interface TeamComparisonProps {
  demoId: string | null
}

interface MetricRow {
  label: string
  ct: string
  t: string
  emphasizeHigher: "ct" | "t" | null
}

// TeamComparison renders the head-to-head bar surfaced on the standalone
// analysis page. Reads GetMatchInsights and lays out CT vs T summaries with
// each metric's higher value highlighted. Empty / loading states match the
// existing analysis components' shimmer pattern.
export function TeamComparison({ demoId }: TeamComparisonProps) {
  const { data, isLoading } = useMatchInsights(demoId)

  if (isLoading) {
    return (
      <div
        data-testid="team-comparison-loading"
        className="flex flex-col gap-2"
      >
        <Skeleton className="h-6 w-2/3 bg-white/10" />
        <Skeleton className="h-32 w-full bg-white/10" />
      </div>
    )
  }
  if (!data || data.ct_summary.players + data.t_summary.players === 0) {
    return (
      <p data-testid="team-comparison-empty" className="text-sm text-white/60">
        Team-level analysis not available.
      </p>
    )
  }
  const rows = buildRows(data.ct_summary, data.t_summary)
  return (
    <section
      data-testid="team-comparison"
      className="rounded border border-white/10 bg-white/5 p-3 text-white"
    >
      <header className="flex items-center justify-between text-xs uppercase tracking-wide">
        <span data-testid="team-comparison-ct">
          CT ({data.ct_summary.players})
        </span>
        <span className="text-white/50">vs</span>
        <span data-testid="team-comparison-t">
          T ({data.t_summary.players})
        </span>
      </header>
      <ul className="mt-2 space-y-1.5 text-sm">
        {rows.map((r) => (
          <li
            key={r.label}
            className="grid grid-cols-[1fr_auto_1fr] items-center gap-2 tabular-nums"
            data-testid={`team-comparison-row-${slug(r.label)}`}
          >
            <span
              className={
                r.emphasizeHigher === "ct"
                  ? "text-right font-semibold text-emerald-300"
                  : "text-right text-white/80"
              }
            >
              {r.ct}
            </span>
            <span className="text-center text-xs text-white/50">{r.label}</span>
            <span
              className={
                r.emphasizeHigher === "t"
                  ? "text-left font-semibold text-emerald-300"
                  : "text-left text-white/80"
              }
            >
              {r.t}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}

function slug(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, "-")
}

function pct(n: number): string {
  return `${Math.round(n * 100)}%`
}

function buildRows(ct: TeamSummary, t: TeamSummary): MetricRow[] {
  const compareNumber = (a: number, b: number): "ct" | "t" | null => {
    if (a === b) return null
    return a > b ? "ct" : "t"
  }
  return [
    {
      label: "Avg score",
      ct: ct.avg_overall_score.toFixed(0),
      t: t.avg_overall_score.toFixed(0),
      emphasizeHigher: compareNumber(ct.avg_overall_score, t.avg_overall_score),
    },
    {
      label: "Trade %",
      ct: pct(ct.avg_trade_pct),
      t: pct(t.avg_trade_pct),
      emphasizeHigher: compareNumber(ct.avg_trade_pct, t.avg_trade_pct),
    },
    {
      label: "Standing %",
      ct: pct(ct.avg_standing_shot_pct),
      t: pct(t.avg_standing_shot_pct),
      emphasizeHigher: compareNumber(
        ct.avg_standing_shot_pct,
        t.avg_standing_shot_pct,
      ),
    },
    {
      label: "First-shot %",
      ct: pct(ct.avg_first_shot_acc_pct),
      t: pct(t.avg_first_shot_acc_pct),
      emphasizeHigher: compareNumber(
        ct.avg_first_shot_acc_pct,
        t.avg_first_shot_acc_pct,
      ),
    },
    {
      label: "Flash assists",
      ct: `${ct.total_flash_assists}`,
      t: `${t.total_flash_assists}`,
      emphasizeHigher: compareNumber(
        ct.total_flash_assists,
        t.total_flash_assists,
      ),
    },
    {
      label: "Smoke kills",
      ct: `${ct.total_smokes_kill_assist}`,
      t: `${t.total_smokes_kill_assist}`,
      emphasizeHigher: compareNumber(
        ct.total_smokes_kill_assist,
        t.total_smokes_kill_assist,
      ),
    },
    {
      label: "HE damage",
      ct: `${ct.total_he_damage}`,
      t: `${t.total_he_damage}`,
      emphasizeHigher: compareNumber(ct.total_he_damage, t.total_he_damage),
    },
    {
      label: "Eco kills",
      ct: `${ct.total_eco_kills}`,
      t: `${t.total_eco_kills}`,
      emphasizeHigher: compareNumber(ct.total_eco_kills, t.total_eco_kills),
    },
  ]
}
