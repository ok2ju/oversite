import { useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { usePlayerRoundAnalysis } from "@/hooks/use-player-round-analysis"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { PlayerRoundEntry } from "@/types/analysis"

const BUY_LABEL: Record<string, string> = {
  pistol: "Pistol",
  eco: "Eco",
  force: "Force",
  full_buy: "Full",
}

// EconomyTable surfaces the per-(player, round) economy roll-up that the
// slice-10 backend persists. Aggregates by buy_type and renders one row per
// classification with rounds, average ADR, and average shots-on-target. The
// analysis page mounts this in the Economy section.
export function EconomyTable() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const { data, isLoading } = usePlayerRoundAnalysis(demoId, steamId)

  const rows = useMemo(() => aggregateByBuy(data ?? []), [data])

  if (isLoading) {
    return (
      <p data-testid="economy-table-loading" className="text-sm text-white/60">
        Loading economy…
      </p>
    )
  }
  if (rows.length === 0) {
    return (
      <p data-testid="economy-table-empty" className="text-sm text-white/60">
        No buy data
      </p>
    )
  }

  return (
    <Table data-testid="economy-table">
      <TableHeader>
        <TableRow>
          <TableHead>Buy</TableHead>
          <TableHead className="text-right">Rounds</TableHead>
          <TableHead className="text-right">Avg spend</TableHead>
          <TableHead className="text-right">Shots</TableHead>
          <TableHead className="text-right">Hit %</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row) => (
          <TableRow
            key={row.buy_type}
            data-testid={`economy-row-${row.buy_type}`}
          >
            <TableCell>{BUY_LABEL[row.buy_type] ?? row.buy_type}</TableCell>
            <TableCell className="text-right tabular-nums">
              {row.rounds}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              ${Math.round(row.avg_spend)}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {row.shots_fired}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {row.shots_fired === 0
                ? "—"
                : `${Math.round((row.shots_hit / row.shots_fired) * 100)}%`}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

interface AggregatedRow {
  buy_type: string
  rounds: number
  avg_spend: number
  shots_fired: number
  shots_hit: number
}

const BUY_ORDER = ["pistol", "eco", "force", "full_buy"]

function aggregateByBuy(rows: PlayerRoundEntry[]): AggregatedRow[] {
  if (rows.length === 0) return []
  const acc = new Map<string, AggregatedRow & { spend_total: number }>()
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
  const out: AggregatedRow[] = []
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
