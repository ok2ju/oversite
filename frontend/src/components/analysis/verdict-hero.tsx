import { useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useAnalysisStore } from "@/stores/analysis"
import { usePlayerAnalysis } from "@/hooks/use-analysis"
import { useMistakeTimeline } from "@/hooks/use-mistake-timeline"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import { CATEGORY_LABEL, categoryForKind } from "@/lib/mistakes"
import type { PlayerAnalysis } from "@/types/analysis"
import type { MistakeEntry } from "@/types/mistake"

type Tier = {
  letter: string
  label: string
  ringColor: string
  textColor: string
}

const TIERS: Tier[] = [
  {
    letter: "S",
    label: "Match-defining",
    ringColor: "#ff7a1a",
    textColor: "#ff7a1a",
  },
  { letter: "A", label: "Strong", ringColor: "#9bbc5a", textColor: "#9bbc5a" },
  { letter: "B", label: "Solid", ringColor: "#9bbc5a", textColor: "#9bbc5a" },
  { letter: "C", label: "Average", ringColor: "#ffc233", textColor: "#ffc233" },
  {
    letter: "D",
    label: "Below par",
    ringColor: "#ff8a3d",
    textColor: "#ff8a3d",
  },
  { letter: "F", label: "Carried", ringColor: "#f87171", textColor: "#f87171" },
]

function tierFor(score: number): Tier {
  if (score >= 90) return TIERS[0]
  if (score >= 80) return TIERS[1]
  if (score >= 70) return TIERS[2]
  if (score >= 55) return TIERS[3]
  if (score >= 40) return TIERS[4]
  return TIERS[5]
}

interface CategoryScore {
  key: "trade" | "aim" | "movement" | "utility"
  label: string
  pct: number
  primary: string
  secondary: string
}

function categoryScores(a: PlayerAnalysis | undefined): CategoryScore[] {
  if (!a) return []
  // Prefer the slice-10 first-class column. Older rows / mocked test
  // fixtures stash aim_pct in the extras blob — fall back to that so the
  // bar lights up instead of flat-lining at zero.
  const aim =
    num(a.first_shot_acc_pct) ?? readExtraNumber(a.extras, "aim_pct") ?? 0
  const standing =
    num(a.standing_shot_pct) ??
    readExtraNumber(a.extras, "standing_shot_pct") ??
    0
  const flash = num(a.flash_assists) ?? 0
  const smokes = num(a.smokes_kill_assist) ?? 0
  const trade = num(a.trade_pct) ?? 0
  return [
    {
      key: "trade",
      label: "Trade",
      pct: clamp01(trade),
      primary: `${Math.round(clamp01(trade) * 100)}%`,
      secondary: "trade rate",
    },
    {
      key: "aim",
      label: "Aim",
      pct: clamp01(aim),
      primary: `${Math.round(clamp01(aim) * 100)}%`,
      secondary: "first-shot",
    },
    {
      key: "movement",
      label: "Movement",
      pct: clamp01(standing),
      primary: `${Math.round(clamp01(standing) * 100)}%`,
      secondary: "standing shots",
    },
    {
      key: "utility",
      label: "Utility",
      pct: clamp01(utilityFraction(a)),
      primary: `${flash + smokes}`,
      secondary: "assists landed",
    },
  ]
}

function utilityFraction(a: PlayerAnalysis): number {
  const flash = num(a.flash_assists) ?? 0
  const smokes = num(a.smokes_kill_assist) ?? 0
  const unused = num(a.nades_unused) ?? 0
  const used = flash + smokes
  const total = used + unused
  if (total <= 0) return 0
  return used / total
}

function num(v: unknown): number | undefined {
  if (typeof v !== "number" || !Number.isFinite(v)) return undefined
  return v
}

function readExtraNumber(
  extras: Record<string, unknown> | null | undefined,
  key: string,
): number | undefined {
  if (!extras) return undefined
  return num(extras[key])
}

function clamp01(v: number): number {
  if (!Number.isFinite(v)) return 0
  if (v < 0) return 0
  if (v > 1) return 1
  return v
}

function buildVerdict(
  a: PlayerAnalysis,
  mistakes: MistakeEntry[] | undefined,
): string {
  const score = a.overall_score
  const items = mistakes ?? []
  const counts = new Map<string, number>()
  for (const m of items) {
    const c = m.category || categoryForKind(m.kind)
    counts.set(c, (counts.get(c) ?? 0) + 1)
  }
  const worst = [...counts.entries()].sort((a, b) => b[1] - a[1])[0]
  const tradePct = Math.round(clamp01(a.trade_pct) * 100)
  const standPct = Math.round(clamp01(a.standing_shot_pct) * 100)

  if (score >= 80) {
    return `Clinical match. Trades stayed at ${tradePct}% and standing shots held at ${standPct}% — keep this template.`
  }
  if (score < 55 && worst) {
    const label = CATEGORY_LABEL[worst[0]] ?? worst[0]
    return `${label.toLowerCase()} held you back — ${worst[1]} flagged ${worst[1] === 1 ? "play" : "plays"}, ${tradePct}% trade rate, ${standPct}% standing shots.`
  }
  if (worst) {
    const label = CATEGORY_LABEL[worst[0]] ?? worst[0]
    return `Solid baseline, but ${label.toLowerCase()} was the weak link — ${worst[1]} flagged ${worst[1] === 1 ? "play" : "plays"} cost rounds you could have closed.`
  }
  return `No flagged plays this round set — ${tradePct}% trade rate and ${standPct}% standing shots.`
}

export function VerdictHero() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const { data, isLoading } = usePlayerAnalysis(demoId, steamId)
  const { data: mistakes } = useMistakeTimeline(demoId, steamId)
  const selectedCategory = useAnalysisStore((s) => s.selectedCategory)
  const setSelectedCategory = useAnalysisStore((s) => s.setSelectedCategory)

  const cats = useMemo(() => categoryScores(data), [data])

  if (isLoading) {
    return (
      <div
        data-testid="verdict-hero-loading"
        className="grid gap-6 rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] p-8 lg:grid-cols-[minmax(0,420px)_1fr]"
      >
        <Skeleton className="h-48 w-full bg-white/5" />
        <Skeleton className="h-48 w-full bg-white/5" />
      </div>
    )
  }
  if (!data || !data.steam_id) {
    return (
      <div
        data-testid="verdict-hero-empty"
        className="rounded-xl border border-dashed border-[var(--border-strong)] bg-[var(--bg-elevated)] px-8 py-10 text-center text-sm text-[var(--text-muted)]"
      >
        Pick a player above to see their debrief.
      </div>
    )
  }

  const tier = tierFor(data.overall_score)
  const verdict = buildVerdict(data, mistakes)

  return (
    <section
      data-testid="verdict-hero"
      className="relative overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)]"
    >
      {/* Graph-paper backdrop */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 opacity-[0.18]"
        style={{
          backgroundImage:
            "linear-gradient(rgba(255,255,255,0.05) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.05) 1px, transparent 1px)",
          backgroundSize: "32px 32px",
          maskImage:
            "radial-gradient(110% 90% at 80% 0%, black 0%, transparent 75%)",
        }}
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            "radial-gradient(640px 220px at 88% -10%, rgba(255,122,26,0.16), transparent 60%), radial-gradient(420px 160px at 6% 110%, rgba(255,122,26,0.06), transparent 70%)",
        }}
      />

      <div className="relative grid gap-8 px-6 py-7 md:px-8 md:py-8 lg:grid-cols-[minmax(0,440px)_1fr] lg:gap-12">
        {/* LEFT: Score + verdict */}
        <div className="flex flex-col gap-5">
          <div className="flex items-center gap-3 text-[10.5px] font-semibold uppercase tracking-[0.18em] text-[var(--text-subtle)]">
            <span
              aria-hidden="true"
              className="inline-block h-px w-8 bg-[var(--border-strong)]"
            />
            <span>A1 · Overall</span>
          </div>

          <div className="flex items-end gap-5">
            <div className="relative">
              <div
                data-testid="verdict-hero-score"
                className="font-[Antonio] font-bold leading-[0.85] tracking-tight text-[var(--text)] tabular-nums"
                style={{ fontSize: "108px" }}
              >
                {data.overall_score}
              </div>
              <span className="absolute -right-7 top-1 text-xs font-medium uppercase tracking-widest text-[var(--text-faint)]">
                /100
              </span>
            </div>
            <div className="flex flex-col gap-1.5 pb-2">
              <div
                data-testid="verdict-hero-tier"
                className="flex h-12 w-12 items-center justify-center rounded-md border-2 font-[Antonio] text-3xl font-bold leading-none"
                style={{
                  borderColor: tier.ringColor,
                  color: tier.textColor,
                }}
              >
                {tier.letter}
              </div>
              <div
                className="text-[10px] font-semibold uppercase tracking-[0.16em]"
                style={{ color: tier.textColor }}
              >
                {tier.label}
              </div>
            </div>
          </div>

          <p
            data-testid="verdict-hero-verdict"
            className="max-w-[44ch] text-[15px] font-medium leading-snug text-[var(--text)]"
            style={{ fontFamily: "'Inter Tight', Inter, sans-serif" }}
          >
            {verdict}
          </p>

          <div className="mt-1 flex flex-wrap items-center gap-x-5 gap-y-1.5 text-[11px] font-mono uppercase tracking-wide text-[var(--text-subtle)]">
            <span>
              <span className="text-[var(--text-faint)]">flagged plays </span>
              <span className="text-[var(--text)]">
                {(mistakes ?? []).length}
              </span>
            </span>
            <span aria-hidden="true" className="text-[var(--text-faint)]">
              ·
            </span>
            <span>
              <span className="text-[var(--text-faint)]">version </span>
              <span className="text-[var(--text)]">v{data.version}</span>
            </span>
          </div>
        </div>

        {/* RIGHT: Category bars */}
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-3 text-[10.5px] font-semibold uppercase tracking-[0.18em] text-[var(--text-subtle)]">
            <span
              aria-hidden="true"
              className="inline-block h-px w-8 bg-[var(--border-strong)]"
            />
            <span>A2 · Categories</span>
          </div>
          <ul className="flex flex-col gap-2.5">
            {cats.map((cat, i) => {
              const active = selectedCategory === cat.key
              const widthPct = Math.round(cat.pct * 100)
              return (
                <li key={cat.key}>
                  <button
                    type="button"
                    data-testid={`verdict-hero-cat-${cat.key}`}
                    data-active={active ? "true" : undefined}
                    onClick={() => setSelectedCategory(active ? null : cat.key)}
                    className={cn(
                      "group flex w-full items-center gap-4 rounded-lg border px-4 py-3 text-left transition-colors",
                      active
                        ? "border-[var(--accent)]/60 bg-[rgba(255,122,26,0.08)]"
                        : "border-[var(--border)] bg-white/[0.02] hover:border-[var(--border-strong)] hover:bg-white/[0.04]",
                    )}
                  >
                    <span
                      className="w-6 shrink-0 font-mono text-[10px] uppercase tracking-widest text-[var(--text-faint)] tabular-nums"
                      aria-hidden="true"
                    >
                      0{i + 1}
                    </span>
                    <span className="w-20 shrink-0 text-[12px] font-semibold uppercase tracking-wide text-[var(--text)]">
                      {cat.label}
                    </span>
                    <span className="relative h-1.5 flex-1 overflow-hidden rounded-full bg-white/[0.06]">
                      <span
                        className={cn(
                          "absolute inset-y-0 left-0 rounded-full transition-[width] duration-700 ease-out",
                          active
                            ? "bg-[var(--accent)]"
                            : "bg-gradient-to-r from-[var(--accent)]/80 to-[var(--accent)]/40 group-hover:from-[var(--accent)] group-hover:to-[var(--accent)]/70",
                        )}
                        style={{ width: `${widthPct}%` }}
                      />
                    </span>
                    <span className="w-16 shrink-0 text-right font-[Antonio] text-2xl font-semibold leading-none text-[var(--text)] tabular-nums">
                      {cat.primary}
                    </span>
                    <span className="hidden w-24 shrink-0 text-right text-[10.5px] uppercase tracking-wide text-[var(--text-faint)] sm:block">
                      {cat.secondary}
                    </span>
                  </button>
                </li>
              )
            })}
          </ul>
        </div>
      </div>
    </section>
  )
}
