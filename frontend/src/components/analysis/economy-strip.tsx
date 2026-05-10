import { useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { usePlayerRoundAnalysis } from "@/hooks/use-player-round-analysis"
import { Skeleton } from "@/components/ui/skeleton"
import type { PlayerRoundEntry } from "@/types/analysis"

const BUY_LABEL: Record<string, string> = {
  pistol: "Pistol",
  eco: "Eco",
  force: "Force",
  full_buy: "Full",
}

const BUY_ORDER = ["pistol", "eco", "force", "full_buy"]

interface BuyRow {
  buy_type: string
  rounds: number
  avg_spend: number
  shots_fired: number
  shots_hit: number
}

// Compact economy strip — replaces the legacy four-column table with a
// horizontal grid where each buy-type lives in its own card. Each card
// surfaces the headline number (rounds played) up front, then average
// spend / shots / hit-% as supporting tabular data.
export function EconomyStrip() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const { data, isLoading } = usePlayerRoundAnalysis(demoId, steamId)

  const rows = useMemo(() => aggregateByBuy(data ?? []), [data])

  if (isLoading) {
    return (
      <div className="rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6">
        <Skeleton className="mb-3 h-5 w-48 bg-white/5" />
        <Skeleton className="h-24 w-full bg-white/5" />
      </div>
    )
  }

  if (rows.length === 0) {
    return (
      <p
        data-testid="economy-strip-empty"
        className="rounded-xl border border-dashed border-[var(--border-strong)] px-6 py-8 text-center text-sm text-[var(--text-muted)]"
      >
        No buy data
      </p>
    )
  }

  return (
    <section
      data-testid="economy-strip"
      className="rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6"
    >
      <header className="mb-4 flex items-center gap-3 text-[10.5px] font-semibold uppercase tracking-[0.18em] text-[var(--text-subtle)]">
        <span
          aria-hidden="true"
          className="inline-block h-px w-8 bg-[var(--border-strong)]"
        />
        <span>E · Economy</span>
      </header>
      <ul className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {rows.map((row) => {
          const hitPct =
            row.shots_fired === 0
              ? null
              : Math.round((row.shots_hit / row.shots_fired) * 100)
          return (
            <li
              key={row.buy_type}
              data-testid={`economy-strip-${row.buy_type}`}
              className="flex flex-col gap-2 rounded-lg border border-[var(--border)] bg-white/[0.02] p-4"
            >
              <div className="flex items-baseline justify-between">
                <span className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--text-muted)]">
                  {BUY_LABEL[row.buy_type] ?? row.buy_type}
                </span>
                <span className="font-[Antonio] text-3xl font-semibold leading-none text-[var(--text)] tabular-nums">
                  {row.rounds}
                  <span className="ml-1 align-baseline text-[11px] font-medium uppercase tracking-wide text-[var(--text-faint)]">
                    rd
                  </span>
                </span>
              </div>
              <dl className="mt-1 grid grid-cols-2 gap-x-3 gap-y-1 font-mono text-[11px] tabular-nums text-[var(--text-muted)]">
                <dt className="text-[var(--text-faint)]">Avg spend</dt>
                <dd className="text-right text-[var(--text)]">
                  ${formatMoney(Math.round(row.avg_spend))}
                </dd>
                <dt className="text-[var(--text-faint)]">Shots</dt>
                <dd className="text-right text-[var(--text)]">
                  {row.shots_fired}
                </dd>
                <dt className="text-[var(--text-faint)]">Hit %</dt>
                <dd className="text-right text-[var(--text)]">
                  {hitPct == null ? "—" : `${hitPct}%`}
                </dd>
              </dl>
            </li>
          )
        })}
      </ul>
    </section>
  )
}

function formatMoney(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return `${n}`
}

function aggregateByBuy(rows: PlayerRoundEntry[]): BuyRow[] {
  if (rows.length === 0) return []
  const acc = new Map<string, BuyRow & { spend_total: number }>()
  for (const r of rows) {
    if (!r.buy_type) continue
    let cur = acc.get(r.buy_type)
    if (!cur) {
      cur = {
        buy_type: r.buy_type,
        rounds: 0,
        avg_spend: 0,
        spend_total: 0,
        shots_fired: 0,
        shots_hit: 0,
      }
      acc.set(r.buy_type, cur)
    }
    cur.rounds++
    cur.spend_total += r.money_spent
    cur.shots_fired += r.shots_fired
    cur.shots_hit += r.shots_hit
  }
  const out: BuyRow[] = []
  for (const r of acc.values()) {
    out.push({
      buy_type: r.buy_type,
      rounds: r.rounds,
      avg_spend: r.rounds === 0 ? 0 : r.spend_total / r.rounds,
      shots_fired: r.shots_fired,
      shots_hit: r.shots_hit,
    })
  }
  out.sort(
    (a, b) =>
      BUY_ORDER.indexOf(a.buy_type) - BUY_ORDER.indexOf(b.buy_type) ||
      a.buy_type.localeCompare(b.buy_type),
  )
  return out
}
