import { useState } from "react"
import { useViewerStore } from "@/stores/viewer"
import { usePlayerAnalysis } from "@/hooks/use-analysis"
import { CATEGORY_LABEL, SUGGESTIONS } from "@/lib/mistakes"

type Category = "trade" | "aim" | "movement"

interface CategoryCardProps {
  // Slice 8 widens the prop union to include the two tick-driven categories
  // (aim, movement). Future slices add utility / positioning / economy under
  // the same shape; each new category just adds a branch in renderMetrics.
  category: Category
}

// Collapsible per-category card mounted in the viewer side panel header (and
// the standalone analysis page). Each card shows two metrics drawn from
// PlayerAnalysis plus a one-line suggestion from SUGGESTIONS. Default state
// is closed; the metrics row + suggestion only render when open. Returns null
// when the analysis row is unavailable (loading, unknown demo, unknown
// player) so the header collapses cleanly.
//
// Slice 8 reads the new aim/standing-shot percentages from the extras blob
// rather than as top-level fields — the slice intentionally avoids a schema
// migration. Future slices may promote them to columns once a third metric
// per category arrives; the prop contract is unchanged.
export function CategoryCard({ category }: CategoryCardProps) {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const tickRate = useViewerStore((s) => s.tickRate)
  const { data, isLoading } = usePlayerAnalysis(demoId, steamId)
  const [open, setOpen] = useState(false)

  if (isLoading) return null
  if (!data || !data.steam_id) return null

  const metrics = renderMetrics(category, data, tickRate)

  return (
    <div
      data-testid={`category-card-${category}`}
      data-state={open ? "open" : "closed"}
      className="rounded border border-white/10 bg-white/5 text-white"
    >
      <button
        type="button"
        data-testid={`category-card-header-${category}`}
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between px-2 py-1 text-left text-xs font-semibold uppercase tracking-wide"
      >
        <span>{CATEGORY_LABEL[category] ?? category}</span>
        <span aria-hidden="true">{open ? "▾" : "▸"}</span>
      </button>
      {open ? (
        <div
          data-testid={`category-card-body-${category}`}
          className="space-y-1 border-t border-white/10 px-2 py-1.5 text-sm"
        >
          {metrics.map((m) => (
            <div
              key={m.testId}
              className="flex items-center justify-between tabular-nums"
            >
              <span className="text-white/70">{m.label}</span>
              <span data-testid={m.testId}>{m.value}</span>
            </div>
          ))}
          <p
            data-testid={`category-card-suggestion-${category}`}
            className="pt-1 text-xs leading-snug text-white/60"
          >
            {SUGGESTIONS[category]}
          </p>
        </div>
      ) : null}
    </div>
  )
}

// PlayerAnalysis shape we read here. Kept inline so this file doesn't depend
// on the generated Wails types — the analysis hook already returns the shape.
type PlayerAnalysisLike = {
  trade_pct?: number
  avg_trade_ticks?: number
  extras?: Record<string, unknown> | null
}

type Metric = { label: string; value: string; testId: string }

function renderMetrics(
  category: Category,
  data: PlayerAnalysisLike,
  tickRate: number,
): Metric[] {
  switch (category) {
    case "trade": {
      const tradePctText = `${Math.round((data.trade_pct ?? 0) * 100)}%`
      // Convert ticks to seconds when the demo's tick rate is available; the
      // "Trades" card reads better as "1.4s" than "90 ticks". Fall back to
      // ticks (one decimal) when tickRate is missing so unit-tests can still
      // assert a stable format without seeding the store.
      const avg = data.avg_trade_ticks ?? 0
      const avgTradeText =
        tickRate > 0
          ? `${(avg / tickRate).toFixed(1)}s`
          : `${avg.toFixed(1)} ticks`
      return [
        {
          label: "Trade %",
          value: tradePctText,
          testId: "category-card-trade-pct-trade",
        },
        {
          label: "Avg trade",
          value: avgTradeText,
          testId: "category-card-avg-trade-trade",
        },
      ]
    }
    case "aim": {
      const aimPct = readNumberFromExtras(data.extras, "aim_pct")
      const engagements = readNumberFromExtras(data.extras, "engagements")
      return [
        {
          label: "Aim %",
          value: aimPct === undefined ? "—" : `${Math.round(aimPct * 100)}%`,
          testId: "category-card-aim-pct-aim",
        },
        {
          label: "Engagements",
          value: engagements === undefined ? "—" : `${engagements}`,
          testId: "category-card-engagements-aim",
        },
      ]
    }
    case "movement": {
      const standPct = readNumberFromExtras(data.extras, "standing_shot_pct")
      const avgSpeed = readNumberFromExtras(data.extras, "avg_fire_speed")
      return [
        {
          label: "Standing shot %",
          value:
            standPct === undefined ? "—" : `${Math.round(standPct * 100)}%`,
          testId: "category-card-standing-shot-pct-movement",
        },
        {
          label: "Avg speed at fire",
          value: avgSpeed === undefined ? "—" : `${Math.round(avgSpeed)} u/s`,
          testId: "category-card-avg-fire-speed-movement",
        },
      ]
    }
  }
}

function readNumberFromExtras(
  extras: Record<string, unknown> | null | undefined,
  key: string,
): number | undefined {
  if (!extras) return undefined
  const v = extras[key]
  if (typeof v !== "number" || Number.isNaN(v)) return undefined
  return v
}
