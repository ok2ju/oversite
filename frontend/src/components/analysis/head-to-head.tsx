import { useMatchInsights } from "@/hooks/use-match-insights"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import type { TeamSummary } from "@/types/analysis"

interface HeadToHeadProps {
  demoId: string | null
}

interface MetricRow {
  label: string
  ct: number
  t: number
  display: (n: number) => string
  // Direction "higher" means a larger number wins; "lower" inverts the
  // bar so the leaner side is highlighted (e.g. crosshair_height_avg_off
  // could use this in a future row).
  direction: "higher"
}

// Head-to-head fight-card. CT on the left, T on the right, the metric
// label centered, and a bidirectional bar showing which side leads. The
// shape of the data matches `team-comparison.tsx` but the visual is
// completely different: bars push outward from the center toward the
// dominant side, the larger number is set in the Antonio display face,
// and the loser is dimmed instead of just untinted.
export function HeadToHead({ demoId }: HeadToHeadProps) {
  const { data, isLoading } = useMatchInsights(demoId)

  if (isLoading) {
    return (
      <div className="rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6">
        <Skeleton className="mb-3 h-5 w-48 bg-white/5" />
        <Skeleton className="h-40 w-full bg-white/5" />
      </div>
    )
  }
  if (!data || data.ct_summary.players + data.t_summary.players === 0) {
    return (
      <p
        data-testid="head-to-head-empty"
        className="rounded-xl border border-dashed border-[var(--border-strong)] px-6 py-8 text-center text-sm text-[var(--text-muted)]"
      >
        Team-level analysis not available for this demo.
      </p>
    )
  }

  const rows = buildRows(data.ct_summary, data.t_summary)
  const ctWins = rows.filter((r) => r.ct > r.t).length

  return (
    <section
      data-testid="head-to-head"
      className="rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6"
    >
      <header className="mb-5 flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-3 text-[10.5px] font-semibold uppercase tracking-[0.18em] text-[var(--text-subtle)]">
            <span
              aria-hidden="true"
              className="inline-block h-px w-8 bg-[var(--border-strong)]"
            />
            <span>D · Head to head</span>
          </div>
          <h3
            className="text-[15px] font-semibold leading-tight text-[var(--text)]"
            style={{ fontFamily: "'Inter Tight', Inter, sans-serif" }}
          >
            CT vs T at a glance
          </h3>
        </div>
        <div
          className="flex items-center gap-3 font-mono text-[11px] uppercase tracking-wide text-[var(--text-faint)]"
          aria-label="metric wins"
        >
          <span className="flex flex-col items-end leading-none">
            <span className="font-[Antonio] text-2xl text-[#5db1ff] tabular-nums">
              {ctWins}
            </span>
            <span>CT leads</span>
          </span>
          <span aria-hidden="true" className="text-[var(--text-faint)]">
            ·
          </span>
          <span className="flex flex-col items-start leading-none">
            <span className="font-[Antonio] text-2xl text-[var(--accent)] tabular-nums">
              {rows.length - ctWins}
            </span>
            <span>T leads</span>
          </span>
        </div>
      </header>

      {/* Side legend */}
      <div className="mb-3 grid grid-cols-[1fr_auto_1fr] items-center gap-4 text-[11px] font-semibold uppercase tracking-[0.16em]">
        <span
          data-testid="head-to-head-ct"
          className="text-right text-[#5db1ff]"
        >
          CT · {data.ct_summary.players}
        </span>
        <span className="text-center text-[var(--text-faint)]">vs</span>
        <span data-testid="head-to-head-t" className="text-[var(--accent)]">
          T · {data.t_summary.players}
        </span>
      </div>

      <ul className="flex flex-col gap-2">
        {rows.map((r) => {
          const total = r.ct + r.t
          const ctRatio = total > 0 ? r.ct / total : 0.5
          const tRatio = total > 0 ? r.t / total : 0.5
          const ctLeads = r.ct > r.t
          const tLeads = r.t > r.ct
          return (
            <li
              key={r.label}
              data-testid={`head-to-head-row-${slug(r.label)}`}
              className="grid grid-cols-[1fr_auto_1fr] items-center gap-3"
            >
              {/* CT side */}
              <div className="flex items-center justify-end gap-3">
                <span
                  className={cn(
                    "font-[Antonio] text-2xl font-semibold leading-none tabular-nums",
                    ctLeads ? "text-[#5db1ff]" : "text-white/45",
                  )}
                >
                  {r.display(r.ct)}
                </span>
                <span className="relative h-1.5 w-full max-w-[180px] overflow-hidden rounded-full bg-white/[0.04]">
                  <span
                    className={cn(
                      "absolute right-0 top-0 h-full rounded-full transition-[width] duration-700 ease-out",
                      ctLeads ? "bg-[#5db1ff]" : "bg-white/15",
                    )}
                    style={{ width: `${Math.max(4, ctRatio * 100)}%` }}
                  />
                </span>
              </div>

              {/* Center label */}
              <span className="min-w-[100px] text-center text-[10.5px] font-semibold uppercase tracking-[0.16em] text-[var(--text-muted)]">
                {r.label}
              </span>

              {/* T side */}
              <div className="flex items-center gap-3">
                <span className="relative h-1.5 w-full max-w-[180px] overflow-hidden rounded-full bg-white/[0.04]">
                  <span
                    className={cn(
                      "absolute left-0 top-0 h-full rounded-full transition-[width] duration-700 ease-out",
                      tLeads ? "bg-[var(--accent)]" : "bg-white/15",
                    )}
                    style={{ width: `${Math.max(4, tRatio * 100)}%` }}
                  />
                </span>
                <span
                  className={cn(
                    "font-[Antonio] text-2xl font-semibold leading-none tabular-nums",
                    tLeads ? "text-[var(--accent)]" : "text-white/45",
                  )}
                >
                  {r.display(r.t)}
                </span>
              </div>
            </li>
          )
        })}
      </ul>
    </section>
  )
}

function slug(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, "-")
}

function intDisplay(n: number): string {
  return `${Math.round(n)}`
}

function pctDisplay(n: number): string {
  return `${Math.round(n * 100)}%`
}

function buildRows(ct: TeamSummary, t: TeamSummary): MetricRow[] {
  return [
    {
      label: "Avg score",
      ct: ct.avg_overall_score,
      t: t.avg_overall_score,
      display: intDisplay,
      direction: "higher",
    },
    {
      label: "Trade %",
      ct: ct.avg_trade_pct,
      t: t.avg_trade_pct,
      display: pctDisplay,
      direction: "higher",
    },
    {
      label: "Standing %",
      ct: ct.avg_standing_shot_pct,
      t: t.avg_standing_shot_pct,
      display: pctDisplay,
      direction: "higher",
    },
    {
      label: "First-shot %",
      ct: ct.avg_first_shot_acc_pct,
      t: t.avg_first_shot_acc_pct,
      display: pctDisplay,
      direction: "higher",
    },
    {
      label: "Flash assists",
      ct: ct.total_flash_assists,
      t: t.total_flash_assists,
      display: intDisplay,
      direction: "higher",
    },
    {
      label: "Smoke kills",
      ct: ct.total_smokes_kill_assist,
      t: t.total_smokes_kill_assist,
      display: intDisplay,
      direction: "higher",
    },
    {
      label: "HE damage",
      ct: ct.total_he_damage,
      t: t.total_he_damage,
      display: intDisplay,
      direction: "higher",
    },
    {
      label: "Eco kills",
      ct: ct.total_eco_kills,
      t: t.total_eco_kills,
      display: intDisplay,
      direction: "higher",
    },
  ]
}
