import { useCallback, useEffect, useMemo, useRef } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useAnalysisStore } from "@/stores/analysis"
import { useMistakeTimeline } from "@/hooks/use-mistake-timeline"
import { useAnalysisStatus } from "@/hooks/use-analysis-status"
import { useRecomputeAnalysis } from "@/hooks/use-recompute-analysis"
import { useRounds } from "@/hooks/use-rounds"
import { Skeleton } from "@/components/ui/skeleton"
import { badgeVariants } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import { CATEGORY_LABEL, OTHER_CATEGORY, categoryForKind } from "@/lib/mistakes"
import { AnalysisOverallGauge } from "@/components/viewer/analysis-overall-gauge"
import { CategoryCard } from "@/components/viewer/category-card"
import type { MistakeEntry } from "@/types/mistake"
import type { Round } from "@/types/round"

// SEVERITY_BADGE_CLASS keys off the integer severity persisted server-side
// (1=low, 2=med, 3=high). Slice 10 made the integer authoritative; the
// per-kind frontend map below is the legacy fallback for rows / tests that
// pre-date the column.
const SEVERITY_BADGE_CLASS: Record<number, string> = {
  1: "bg-white/15 text-white/70",
  2: "bg-amber-400/20 text-amber-300",
  3: "bg-red-500/25 text-red-300",
}

// FALLBACK_KIND_LABEL covers the case where MistakeEntry.title is empty —
// pre-slice-10 rows and test fixtures that mock the binding without the new
// fields. Every kind shipped by the backend is taught here; new rules should
// rely on the server-side title and not extend this map.
const FALLBACK_KIND_LABEL: Record<string, string> = {
  no_trade_death: "Untraded death",
  died_with_util_unused: "Died with utility unused",
  survived_with_util: "Survived with utility unused",
  crosshair_too_low: "Crosshair too low",
  shot_while_moving: "Shot while moving",
  slow_reaction: "Slow reaction",
  missed_flick: "Missed flick",
  missed_first_shot: "Missed first shot",
  spray_decay: "Spray decay",
  no_counter_strafe: "No counter-strafe",
  unused_smoke: "Unused smoke",
  isolated_peek: "Isolated peek",
  repeated_death_zone: "Repeated death zone",
  walked_into_molotov: "Walked into molotov",
  eco_misbuy: "Eco misbuy",
  caught_reloading: "Caught reloading",
  flash_assist: "Flash assist",
  he_damage: "HE damage",
}

// FALLBACK_KIND_SEVERITY mirrors the backend's templates.go severity map —
// used when MistakeEntry.severity is 0 (legacy / mocked rows).
const FALLBACK_KIND_SEVERITY: Record<string, number> = {
  no_trade_death: 2,
  died_with_util_unused: 3,
  survived_with_util: 2,
  crosshair_too_low: 1,
  shot_while_moving: 2,
  slow_reaction: 2,
  missed_flick: 1,
  missed_first_shot: 2,
  spray_decay: 2,
  no_counter_strafe: 2,
  unused_smoke: 1,
  isolated_peek: 3,
  repeated_death_zone: 2,
  walked_into_molotov: 1,
  eco_misbuy: 1,
  caught_reloading: 3,
  flash_assist: 1,
  he_damage: 1,
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
  // entry.title comes from the backend's templates.go; an empty string
  // (unknown kind, legacy / mocked row) falls back to the kind label map
  // and ultimately to the raw kind string so the panel never renders a
  // blank row.
  const label = entry.title || FALLBACK_KIND_LABEL[entry.kind] || entry.kind
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
const CATEGORY_ORDER = ["trade", "utility", "aim", "movement", OTHER_CATEGORY]

export function MistakeList({ steamId: steamIdProp }: MistakeListProps = {}) {
  const demoId = useViewerStore((s) => s.demoId)
  const selectedSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const tickRate = useViewerStore((s) => s.tickRate)
  const setTick = useViewerStore((s) => s.setTick)
  const steamId = steamIdProp ?? selectedSteamId
  const selectedCategory = useAnalysisStore((s) => s.selectedCategory)
  const setSelectedCategory = useAnalysisStore((s) => s.setSelectedCategory)
  const setSelectedMistakeId = useAnalysisStore((s) => s.setSelectedMistakeId)
  const { data: rounds } = useRounds(demoId)
  const { data: mistakes, isLoading } = useMistakeTimeline(demoId, steamId)
  const { data: analysisStatus } = useAnalysisStatus(demoId)
  const recompute = useRecomputeAnalysis()
  const status = analysisStatus?.status

  // Auto-trigger recompute when the panel sees a "missing" status. Guarded by
  // a ref keyed on (demoId, status) so React 18 strict-mode double-invokes
  // and any in-mutation re-render don't fire RecomputeAnalysis twice.
  const recomputeRef = useRef<{ demoId: string | null; status?: string }>({
    demoId: null,
  })
  useEffect(() => {
    if (!demoId || status !== "missing") return
    if (
      recomputeRef.current.demoId === demoId &&
      recomputeRef.current.status === status
    ) {
      return
    }
    recomputeRef.current = { demoId, status }
    recompute.mutate({ demoId })
  }, [demoId, status, recompute])

  const handleSelect = useCallback(
    (entry: MistakeEntry) => {
      setTick(entry.tick)
      // entry.id is 0 on legacy rows that pre-date the slice-10 ID column;
      // clear the selection in that case so the detail surface keeps its
      // existing state rather than firing a "not found" fetch.
      setSelectedMistakeId(entry.id || null)
    },
    [setTick, setSelectedMistakeId],
  )
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
      const cat = m.category || categoryForKind(m.kind)
      counts.set(cat, (counts.get(cat) ?? 0) + 1)
    }
    // Show every category that has at least one mistake, with known categories
    // ordered first and any backend-only category trailing.
    const known = CATEGORY_ORDER.filter((c) => counts.has(c))
    const extra = [...counts.keys()]
      .filter((c) => !CATEGORY_ORDER.includes(c))
      .sort()
    return [...known, ...extra].map((c) => ({
      category: c,
      count: counts.get(c) ?? 0,
    }))
  }, [items])

  const visibleItems = selectedCategory
    ? items.filter(
        (m) => (m.category || categoryForKind(m.kind)) === selectedCategory,
      )
    : items

  if (!steamId) return null

  // The demo isn't analyzable yet (lifecycle states owned by the import flow).
  // For "imported" / "failed" we render nothing so the demos list / parser
  // owns the messaging; "parsing" falls through to the existing loading text.
  if (status === "imported" || status === "failed") return null

  // Render shimmer while the recompute is in flight or the status hasn't
  // flipped to "ready" yet. Same <aside> shell as the populated state so the
  // panel doesn't visually shift when the data lands.
  const showShimmer =
    status === "missing" || status === "parsing" || recompute.isPending
  if (showShimmer) {
    return (
      <aside
        data-testid="mistake-list"
        className="hud-panel absolute left-0 top-0 z-30 flex h-full w-72 flex-col rounded-none border-l-0 border-r border-t-0 border-white/[0.07] text-white"
      >
        <div
          data-testid="mistake-list-shimmer"
          className="flex flex-col gap-3 px-3 py-3"
        >
          <Skeleton className="h-12 w-full bg-white/10" />
          <Skeleton className="h-20 w-full bg-white/10" />
          <Skeleton className="h-6 w-2/3 bg-white/10" />
        </div>
      </aside>
    )
  }

  return (
    <aside
      data-testid="mistake-list"
      className="absolute left-0 top-0 z-30 flex h-full w-72 flex-col border-r border-white/10 bg-black/85 text-white shadow-2xl backdrop-blur"
    >
      <header className="flex flex-col gap-2.5 border-b border-white/[0.07] bg-white/[0.015] px-3 py-3">
        <AnalysisOverallGauge />
        <CategoryCard category="trade" />
        <span className="hud-callsign text-[10px] font-semibold text-white/55">
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
              const severity = m.severity || FALLBACK_KIND_SEVERITY[m.kind] || 1
              return (
                <li key={`${m.kind}-${m.tick}-${i}`}>
                  <button
                    type="button"
                    data-testid={`mistake-list-row-${i}`}
                    onClick={() => handleSelect(m)}
                    className="group flex w-full items-center gap-2 rounded-md border border-white/[0.06] bg-white/[0.03] px-2 py-1.5 text-left text-[12px] tabular-nums transition-colors hover:border-white/15 hover:bg-white/[0.08] focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-400/50"
                  >
                    <span
                      data-testid={`mistake-row-severity-${m.kind}`}
                      aria-hidden="true"
                      className={`inline-block h-2 w-2 shrink-0 rounded-full ${SEVERITY_BADGE_CLASS[severity] ?? SEVERITY_BADGE_CLASS[1]}`}
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
