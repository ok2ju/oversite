import { useCallback, useEffect, useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useAnalysisStore } from "@/stores/analysis"
import { useMistakeTimeline } from "@/hooks/use-mistake-timeline"
import { useRounds } from "@/hooks/use-rounds"
import { badgeVariants } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import { CATEGORY_LABEL, OTHER_CATEGORY, categoryForKind } from "@/lib/mistakes"
import { AnalysisOverallGauge } from "@/components/viewer/analysis-overall-gauge"
import { CategoryCard } from "@/components/viewer/category-card"
import type { MistakeEntry } from "@/types/mistake"
import type { Round } from "@/types/round"

// Human-readable label per mistake kind. Falls back to the raw kind string for
// kinds the frontend hasn't been taught about yet — keeps a future Go-only
// rule from rendering as a blank row.
const KIND_LABEL: Record<string, string> = {
  no_trade_death: "Untraded death",
  died_with_util_unused: "Died with utility unused",
}

type Severity = "low" | "med" | "high"

// Severity tier per kind. Lives on the frontend in slice 3 — promoting
// severity into the persisted data model is deferred until composite scoring
// (slice 5). Unknown kinds fall through to the neutral "low" tint.
const KIND_SEVERITY: Record<string, Severity> = {
  no_trade_death: "med",
  died_with_util_unused: "high",
}

const SEVERITY_BADGE_CLASS: Record<Severity, string> = {
  low: "bg-white/15 text-white/70",
  med: "bg-amber-400/20 text-amber-300",
  high: "bg-red-500/25 text-red-300",
}

function formatClock(
  tick: number,
  round: Round | undefined,
  tickRate: number,
): string {
  if (!round || tickRate <= 0) return "?:??"
  // Prefer freeze_end_tick (the round-live tick) so the clock matches what the
  // player remembers seeing in-game. Older imports may still have 0 there;
  // fall back to start_tick in that case.
  const base =
    round.freeze_end_tick > 0 ? round.freeze_end_tick : round.start_tick
  const secs = Math.max(0, Math.round((tick - base) / tickRate))
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return `${m}:${s.toString().padStart(2, "0")}`
}

function formatMistakeText(
  entry: MistakeEntry,
  rounds: Round[] | undefined,
  tickRate: number,
): string {
  const label = KIND_LABEL[entry.kind] ?? entry.kind
  const round = rounds?.find((r) => r.round_number === entry.round_number)
  return `${label} — round ${entry.round_number}, ${formatClock(
    entry.tick,
    round,
    tickRate,
  )}`
}

interface MistakeListProps {
  // Optional override for the player whose mistakes to render. When omitted
  // (the default), the list reads selectedPlayerSteamId from useViewerStore —
  // matching the player-stats-panel.tsx convention.
  steamId?: string | null
}

// Stable display order for the count strip — known categories first, then
// "other" so a future Go-only rule still surfaces without a frontend change.
const CATEGORY_ORDER = ["trade", "utility", OTHER_CATEGORY]

export function MistakeList({ steamId: steamIdProp }: MistakeListProps = {}) {
  const demoId = useViewerStore((s) => s.demoId)
  const selectedSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const tickRate = useViewerStore((s) => s.tickRate)
  const setTick = useViewerStore((s) => s.setTick)
  const steamId = steamIdProp ?? selectedSteamId
  const selectedCategory = useAnalysisStore((s) => s.selectedCategory)
  const setSelectedCategory = useAnalysisStore((s) => s.setSelectedCategory)
  const { data: rounds } = useRounds(demoId)
  const { data: mistakes, isLoading } = useMistakeTimeline(demoId, steamId)

  const handleSelect = useCallback((tick: number) => setTick(tick), [setTick])
  const handleCategoryClick = useCallback(
    (cat: string) => {
      setSelectedCategory(selectedCategory === cat ? null : cat)
    },
    [selectedCategory, setSelectedCategory],
  )

  // Clear the filter when the (demo, player) pair changes — matches the
  // heatmap-store reset-cascade convention so a stale filter from a previous
  // player doesn't bleed across context switches.
  useEffect(() => {
    setSelectedCategory(null)
  }, [demoId, steamId, setSelectedCategory])

  const items = useMemo(() => mistakes ?? [], [mistakes])

  const categoryCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const m of items) {
      const cat = categoryForKind(m.kind)
      counts.set(cat, (counts.get(cat) ?? 0) + 1)
    }
    return CATEGORY_ORDER.filter((c) => counts.has(c)).map((c) => ({
      category: c,
      count: counts.get(c) ?? 0,
    }))
  }, [items])

  const visibleItems = selectedCategory
    ? items.filter((m) => categoryForKind(m.kind) === selectedCategory)
    : items

  if (!steamId) return null

  return (
    <aside
      data-testid="mistake-list"
      className="absolute left-0 top-0 z-30 flex h-full w-72 flex-col border-r border-white/10 bg-black/85 text-white shadow-2xl backdrop-blur"
    >
      <header className="flex flex-col gap-2 border-b border-white/10 px-3 py-2">
        <AnalysisOverallGauge />
        <CategoryCard category="trade" />
        <span className="text-xs font-semibold uppercase tracking-wide text-white/70">
          Mistakes
        </span>
        {categoryCounts.length > 0 ? (
          <div
            data-testid="mistake-category-bar"
            className="flex flex-wrap items-center gap-1.5"
          >
            {categoryCounts.map(({ category, count }) => {
              const active = selectedCategory === category
              return (
                <button
                  key={category}
                  type="button"
                  data-testid={`mistake-category-badge-${category}`}
                  data-active={active ? "true" : undefined}
                  onClick={() => handleCategoryClick(category)}
                  className={cn(
                    badgeVariants({
                      variant: active ? "default" : "secondary",
                    }),
                    "cursor-pointer",
                  )}
                >
                  {CATEGORY_LABEL[category] ?? category} {count}
                </button>
              )
            })}
          </div>
        ) : null}
      </header>
      <div className="flex-1 overflow-y-auto p-2">
        {isLoading ? (
          <p
            data-testid="mistake-list-loading"
            className="text-sm text-white/60"
          >
            Loading mistakes…
          </p>
        ) : items.length === 0 ? (
          <p data-testid="mistake-list-empty" className="text-sm text-white/60">
            No mistakes
          </p>
        ) : (
          <ul className="space-y-1">
            {visibleItems.map((m, i) => {
              const severity = KIND_SEVERITY[m.kind] ?? "low"
              return (
                <li key={`${m.kind}-${m.tick}-${i}`}>
                  <button
                    type="button"
                    data-testid={`mistake-list-row-${i}`}
                    onClick={() => handleSelect(m.tick)}
                    className="flex w-full items-center gap-2 rounded border border-white/10 bg-white/5 px-2 py-1 text-left text-sm tabular-nums hover:bg-white/10 focus:outline-none focus-visible:ring-2 focus-visible:ring-white/40"
                  >
                    <span
                      data-testid={`mistake-row-severity-${m.kind}`}
                      aria-hidden="true"
                      className={`inline-block h-2 w-2 shrink-0 rounded-full ${SEVERITY_BADGE_CLASS[severity]}`}
                    />
                    <span>{formatMistakeText(m, rounds, tickRate)}</span>
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </aside>
  )
}
