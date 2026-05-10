import { useCallback, useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useAnalysisStore } from "@/stores/analysis"
import { useMistakeTimeline } from "@/hooks/use-mistake-timeline"
import { useRounds } from "@/hooks/use-rounds"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import { CATEGORY_LABEL, OTHER_CATEGORY, categoryForKind } from "@/lib/mistakes"
import type { MistakeEntry } from "@/types/mistake"
import type { Round } from "@/types/round"

const CATEGORY_ORDER = ["trade", "utility", "aim", "movement", OTHER_CATEGORY]

const SEVERITY_META: Record<
  number,
  { dot: string; label: string; ring: string; bg: string; text: string }
> = {
  1: {
    dot: "bg-white/35",
    label: "Low",
    ring: "ring-white/10",
    bg: "bg-white/[0.03]",
    text: "text-white/60",
  },
  2: {
    dot: "bg-[#ffc233]",
    label: "Med",
    ring: "ring-[#ffc233]/25",
    bg: "bg-[rgba(255,194,51,0.06)]",
    text: "text-[#ffc233]",
  },
  3: {
    dot: "bg-[#ff5050]",
    label: "High",
    ring: "ring-[#ff5050]/30",
    bg: "bg-[rgba(255,80,80,0.07)]",
    text: "text-[#ff8a8a]",
  },
}

function formatClock(
  tick: number,
  round: Round | undefined,
  tickRate: number,
): string {
  if (!round || tickRate <= 0) return "?:??"
  const base =
    round.freeze_end_tick > 0 ? round.freeze_end_tick : round.start_tick
  const secs = Math.max(0, Math.round((tick - base) / tickRate))
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return `${m}:${s.toString().padStart(2, "0")}`
}

// MistakesFeed — first-class action surface that replaces the buried
// MistakeDetail empty state. Filter by category, sort by severity (high
// first), then click any row to seek the viewer's PixiJS canvas to that
// tick. Selection is mirrored to useAnalysisStore for cross-component
// highlight (the legacy MistakeDetail card consumes the same store).
export function MistakesFeed() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const tickRate = useViewerStore((s) => s.tickRate)
  const setTick = useViewerStore((s) => s.setTick)
  const selectedCategory = useAnalysisStore((s) => s.selectedCategory)
  const setSelectedCategory = useAnalysisStore((s) => s.setSelectedCategory)
  const selectedMistakeId = useAnalysisStore((s) => s.selectedMistakeId)
  const setSelectedMistakeId = useAnalysisStore((s) => s.setSelectedMistakeId)
  const { data: mistakes, isLoading } = useMistakeTimeline(demoId, steamId)
  const { data: rounds } = useRounds(demoId)

  const items = useMemo(() => mistakes ?? [], [mistakes])

  const categoryCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const m of items) {
      const c = m.category || categoryForKind(m.kind)
      counts.set(c, (counts.get(c) ?? 0) + 1)
    }
    const known = CATEGORY_ORDER.filter((c) => counts.has(c))
    const extra = [...counts.keys()]
      .filter((c) => !CATEGORY_ORDER.includes(c))
      .sort()
    return [...known, ...extra].map((c) => ({
      category: c,
      count: counts.get(c) ?? 0,
    }))
  }, [items])

  const visible = useMemo(() => {
    const filtered = selectedCategory
      ? items.filter(
          (m) => (m.category || categoryForKind(m.kind)) === selectedCategory,
        )
      : items
    // High → Med → Low; tie-break on round number
    return [...filtered].sort((a, b) => {
      const sa = a.severity || 1
      const sb = b.severity || 1
      if (sa !== sb) return sb - sa
      return a.round_number - b.round_number
    })
  }, [items, selectedCategory])

  const onPick = useCallback(
    (m: MistakeEntry) => {
      setTick(m.tick)
      setSelectedMistakeId(m.id || null)
    },
    [setTick, setSelectedMistakeId],
  )

  const totals = useMemo(() => {
    const t = { high: 0, med: 0, low: 0 }
    for (const m of items) {
      const s = m.severity || 1
      if (s >= 3) t.high++
      else if (s === 2) t.med++
      else t.low++
    }
    return t
  }, [items])

  if (isLoading) {
    return (
      <div className="rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6">
        <Skeleton className="mb-3 h-5 w-48 bg-white/5" />
        <Skeleton className="h-32 w-full bg-white/5" />
      </div>
    )
  }

  return (
    <section
      data-testid="mistakes-feed"
      className="rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6"
    >
      <header className="mb-4 flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-3 text-[10.5px] font-semibold uppercase tracking-[0.18em] text-[var(--text-subtle)]">
            <span
              aria-hidden="true"
              className="inline-block h-px w-8 bg-[var(--border-strong)]"
            />
            <span>C · Flagged plays</span>
          </div>
          <h3
            className="text-[15px] font-semibold leading-tight text-[var(--text)]"
            style={{ fontFamily: "'Inter Tight', Inter, sans-serif" }}
          >
            Watch the moments that cost you rounds
          </h3>
        </div>

        <div className="flex items-end gap-5 font-mono text-[11px] uppercase tracking-wide text-[var(--text-faint)]">
          <span className="flex items-baseline gap-1.5">
            <span className="h-2 w-2 rounded-full bg-[#ff5050]" />
            <span className="font-[Antonio] text-2xl font-semibold leading-none text-[var(--text)] tabular-nums">
              {totals.high}
            </span>
            <span>high</span>
          </span>
          <span className="flex items-baseline gap-1.5">
            <span className="h-2 w-2 rounded-full bg-[#ffc233]" />
            <span className="font-[Antonio] text-2xl font-semibold leading-none text-[var(--text)] tabular-nums">
              {totals.med}
            </span>
            <span>med</span>
          </span>
          <span className="flex items-baseline gap-1.5">
            <span className="h-2 w-2 rounded-full bg-white/35" />
            <span className="font-[Antonio] text-2xl font-semibold leading-none text-[var(--text)] tabular-nums">
              {totals.low}
            </span>
            <span>low</span>
          </span>
        </div>
      </header>

      {/* Filter chips */}
      {categoryCounts.length > 0 ? (
        <div
          data-testid="mistakes-feed-filters"
          className="mb-4 flex flex-wrap items-center gap-1.5"
        >
          <button
            type="button"
            data-active={selectedCategory == null ? "true" : undefined}
            onClick={() => setSelectedCategory(null)}
            className={cn(
              "rounded-full border px-3 py-1 text-[11px] font-semibold uppercase tracking-wide transition-colors",
              selectedCategory == null
                ? "border-[var(--accent)] bg-[rgba(255,122,26,0.12)] text-[var(--accent-ink)]"
                : "border-[var(--border-strong)] bg-transparent text-[var(--text-muted)] hover:bg-white/[0.04] hover:text-[var(--text)]",
            )}
          >
            All <span className="font-mono tabular-nums">{items.length}</span>
          </button>
          {categoryCounts.map(({ category, count }) => {
            const active = selectedCategory === category
            return (
              <button
                key={category}
                type="button"
                data-testid={`mistakes-feed-filter-${category}`}
                data-active={active ? "true" : undefined}
                onClick={() => setSelectedCategory(active ? null : category)}
                className={cn(
                  "rounded-full border px-3 py-1 text-[11px] font-semibold uppercase tracking-wide transition-colors",
                  active
                    ? "border-[var(--accent)] bg-[rgba(255,122,26,0.12)] text-[var(--accent-ink)]"
                    : "border-[var(--border-strong)] bg-transparent text-[var(--text-muted)] hover:bg-white/[0.04] hover:text-[var(--text)]",
                )}
              >
                {CATEGORY_LABEL[category] ?? category}{" "}
                <span className="font-mono tabular-nums">{count}</span>
              </button>
            )
          })}
        </div>
      ) : null}

      {visible.length === 0 ? (
        <p
          data-testid="mistakes-feed-empty"
          className="rounded-lg border border-dashed border-[var(--border-strong)] py-8 text-center text-sm text-[var(--text-muted)]"
        >
          No flagged plays in this slice. Clean half.
        </p>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {visible.map((m, i) => {
            const sev = m.severity || 1
            const meta = SEVERITY_META[sev] ?? SEVERITY_META[1]
            const round = rounds?.find((r) => r.round_number === m.round_number)
            const clock = formatClock(m.tick, round, tickRate)
            const cat = m.category || categoryForKind(m.kind)
            const active =
              selectedMistakeId !== null && selectedMistakeId === m.id
            return (
              <li
                key={`${m.id}-${i}`}
                data-testid={`mistakes-feed-row-${i}`}
                data-active={active ? "true" : undefined}
                className={cn(
                  "group flex items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors",
                  active
                    ? "border-[var(--accent)]/60 bg-[rgba(255,122,26,0.07)]"
                    : cn(
                        "border-[var(--border)]",
                        meta.bg,
                        "hover:border-[var(--border-strong)]",
                      ),
                )}
              >
                <span
                  aria-hidden="true"
                  className={cn(
                    "h-2.5 w-2.5 shrink-0 rounded-full ring-2",
                    meta.dot,
                    meta.ring,
                  )}
                />
                <span
                  className={cn(
                    "w-12 shrink-0 font-mono text-[10px] font-semibold uppercase tracking-wider",
                    meta.text,
                  )}
                >
                  {meta.label}
                </span>
                <span className="w-20 shrink-0 font-[Antonio] text-lg leading-none text-[var(--text)] tabular-nums">
                  R{m.round_number}
                  <span className="ml-1.5 align-middle font-mono text-[10px] uppercase tracking-wide text-[var(--text-faint)]">
                    {clock}
                  </span>
                </span>
                <span className="min-w-0 flex-1 truncate text-[13px] text-[var(--text)]">
                  {m.title || m.kind}
                  {m.suggestion ? (
                    <span className="ml-2 text-[11.5px] text-[var(--text-muted)]">
                      — {m.suggestion}
                    </span>
                  ) : null}
                </span>
                <span className="hidden shrink-0 font-mono text-[10px] uppercase tracking-wide text-[var(--text-faint)] sm:block">
                  {CATEGORY_LABEL[cat] ?? cat}
                </span>
                <button
                  type="button"
                  data-testid={`mistakes-feed-watch-${i}`}
                  onClick={() => onPick(m)}
                  className="flex shrink-0 items-center gap-1.5 rounded-md border border-[var(--border-strong)] bg-[var(--bg-sunken)] px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wide text-[var(--text)] transition-colors hover:border-[var(--accent)]/60 hover:bg-[rgba(255,122,26,0.08)] hover:text-[var(--accent-ink)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--accent)]"
                >
                  <span aria-hidden="true" className="-mt-px">
                    ▶
                  </span>
                  Watch
                </button>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}
