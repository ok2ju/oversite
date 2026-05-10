import { useState } from "react"
import { useViewerStore } from "@/stores/viewer"
import { usePlayerAnalysis } from "@/hooks/use-analysis"
import { CATEGORY_LABEL, SUGGESTIONS } from "@/lib/mistakes"

interface CategoryCardProps {
  // Slice 5 only ships "trade"; the prop is here so slice 7 can mount more
  // cards (utility, aim, …) without forking this file.
  category: "trade"
}

// Collapsible per-category card mounted in the viewer side panel header. Each
// card shows two metrics drawn from PlayerAnalysis plus a one-line suggestion
// from SUGGESTIONS. Default state is closed; the metrics row + suggestion
// only render when open. Returns null when the analysis row is unavailable
// (loading, unknown demo, or unknown player) so the header collapses cleanly.
export function CategoryCard({ category }: CategoryCardProps) {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const tickRate = useViewerStore((s) => s.tickRate)
  const { data, isLoading } = usePlayerAnalysis(demoId, steamId)
  const [open, setOpen] = useState(false)

  if (isLoading) return null
  if (!data || !data.steam_id) return null

  const tradePctText = `${Math.round((data.trade_pct ?? 0) * 100)}%`
  // Convert ticks to seconds when the demo's tick rate is available; the
  // "Trades" card reads better as "1.4s" than "90 ticks". Fall back to ticks
  // (one decimal) when tickRate is missing so unit-tests can still assert a
  // stable format without seeding the store.
  const avgTradeText =
    tickRate > 0
      ? `${(data.avg_trade_ticks / tickRate).toFixed(1)}s`
      : `${data.avg_trade_ticks.toFixed(1)} ticks`

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
          <div className="flex items-center justify-between tabular-nums">
            <span className="text-white/70">Trade %</span>
            <span data-testid={`category-card-trade-pct-${category}`}>
              {tradePctText}
            </span>
          </div>
          <div className="flex items-center justify-between tabular-nums">
            <span className="text-white/70">Avg trade</span>
            <span data-testid={`category-card-avg-trade-${category}`}>
              {avgTradeText}
            </span>
          </div>
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
