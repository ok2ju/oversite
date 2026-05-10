import { useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { usePlayerRoundAnalysis } from "@/hooks/use-player-round-analysis"

// Hand-rolled per-round trade-percentage bar strip mounted on the standalone
// analysis page. One column per round in the demo's full round list (so the
// chart shows the entire match cadence even for rounds the player didn't die
// in); columns without an analysis row render flat at zero so a missing row
// reads visually identical to "no eligible deaths".
//
// No charting library is added — slice 7 explicitly defers a real chart
// component until follow-up work needs axis labels / tooltips beyond the
// per-bar `title` attribute.
export function RoundTradeBars() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const { data: rounds } = useRounds(demoId)
  const { data: roundRows } = usePlayerRoundAnalysis(demoId, steamId)

  // Build a Map<round_number, trade_pct> once per (rounds, rows) change so the
  // render loop is O(rounds) instead of O(rounds × rows).
  const tradeByRound = useMemo(() => {
    const m = new Map<number, number>()
    for (const r of roundRows ?? []) {
      m.set(r.round_number, r.trade_pct)
    }
    return m
  }, [roundRows])

  if (!rounds || rounds.length === 0) {
    return (
      <p
        data-testid="round-trade-bars-empty"
        className="text-sm text-muted-foreground"
      >
        No round data
      </p>
    )
  }

  return (
    <div
      data-testid="round-trade-bars"
      className="flex h-32 items-end gap-1 rounded border border-border bg-muted/30 p-3"
    >
      {rounds.map((round) => {
        const pct = tradeByRound.get(round.round_number) ?? 0
        const heightPct = Math.max(0, Math.min(1, pct)) * 100
        const titleLabel = `Round ${round.round_number} — ${Math.round(pct * 100)}%`
        return (
          <div
            key={round.round_number}
            data-testid={`round-trade-bar-${round.round_number}`}
            data-trade-pct={pct}
            title={titleLabel}
            className="flex-1 self-end rounded-sm bg-primary/70"
            style={{ height: `${heightPct}%` }}
          />
        )
      })}
    </div>
  )
}
