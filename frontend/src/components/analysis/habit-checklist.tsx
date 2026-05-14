import { useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useAnalysisStore } from "@/stores/analysis"
import { useHabitReport } from "@/hooks/use-habit-report"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import type {
  HabitDirection,
  HabitKey,
  HabitRow,
  HabitStatus,
} from "@/types/analysis"

// Status pill swatches — shared with verdict-hero tier table. Color is
// decoration; the row also echoes the status word in text (no color-only
// signaling — accessibility, see plan §2.1).
const STATUS_COLOR: Record<HabitStatus, string> = {
  good: "#9bbc5a",
  warn: "#ffc233",
  bad: "#f87171",
}

// Habit → mistake-feed category. Clicking a habit row drives
// useAnalysisStore.selectedCategory which the MistakesFeed reads as its
// filter — so a "bad: counter_strafe" click narrows the feed to movement.
const HABIT_CATEGORY: Record<HabitKey, string> = {
  counter_strafe: "movement",
  shooting_in_motion: "movement",
  reaction: "aim",
  first_shot_acc: "aim",
  crouch_before_shot: "aim",
  flick_balance: "aim",
  trade_timing: "trade",
  utility_used: "utility",
  isolated_peek_deaths: "positioning",
  repeated_death_zone: "positioning",
}

function formatValue(row: HabitRow): string {
  const v = row.value
  if (row.unit === "%") {
    return `${Math.round(v)} %`
  }
  if (row.unit === "ms") {
    return `${Math.round(v)} ms`
  }
  if (!row.unit) {
    return `${Math.round(v)}`
  }
  return `${Math.round(v)} ${row.unit}`
}

function formatNorm(row: HabitRow): string {
  const unit = row.unit ? ` ${row.unit}` : ""
  if (row.direction === "lower") {
    return `norm ≤ ${formatThreshold(row.good_threshold)}${unit}`
  }
  if (row.direction === "higher") {
    return `norm ≥ ${formatThreshold(row.good_threshold)}${unit}`
  }
  return `norm ${formatThreshold(row.good_min)}–${formatThreshold(row.good_max)}${unit}`
}

function formatThreshold(n: number): string {
  if (Number.isInteger(n)) return `${n}`
  return `${Number(n.toFixed(2))}`
}

interface DeltaPresentation {
  arrow: "↑" | "↓"
  magnitude: number
  improving: boolean
  unit: string
}

function buildDelta(row: HabitRow): DeltaPresentation | null {
  if (row.delta === null) return null
  const d = row.delta
  if (d === 0) return null
  const arrow: "↑" | "↓" = d > 0 ? "↑" : "↓"
  const magnitude = Math.abs(d)
  const improving = isImproving(d, row.direction)
  // Percentages are reported as percentage points (pp) when they move.
  const unit = row.unit === "%" ? "pp" : row.unit
  return { arrow, magnitude, improving, unit }
}

function isImproving(delta: number, direction: HabitDirection): boolean {
  if (direction === "lower") return delta < 0
  if (direction === "higher") return delta > 0
  // For balanced, we don't strictly know which way is better without
  // knowing the prior side of the band — treat any movement neutrally.
  // We still color it green if it moved closer to neutral; here we
  // approximate "improvement" as no movement away from zero when balanced.
  return false
}

interface RowProps {
  row: HabitRow
  active: boolean
  onToggle: () => void
}

function HabitChecklistRow({ row, active, onToggle }: RowProps) {
  const color = STATUS_COLOR[row.status]
  const valueText = formatValue(row)
  const normText = formatNorm(row)
  const delta = buildDelta(row)

  return (
    <li>
      <button
        type="button"
        data-testid={`habit-checklist-row-${row.key}`}
        data-active={active ? "true" : undefined}
        data-status={row.status}
        onClick={onToggle}
        className={cn(
          "group flex w-full items-center gap-4 rounded-lg border px-4 py-3 text-left transition-colors",
          active
            ? "border-[var(--accent)]/60 bg-[rgba(232,155,42,0.08)]"
            : "border-[var(--border)] bg-white/[0.02] hover:border-[var(--border-strong)] hover:bg-white/[0.04]",
        )}
      >
        <span
          aria-hidden="true"
          data-testid={`habit-checklist-pill-${row.key}`}
          className="h-2 w-2 shrink-0 rounded-sm"
          style={{ backgroundColor: color }}
        />
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="flex items-center gap-2">
            <span className="text-[12px] font-semibold uppercase tracking-wide text-[var(--text)]">
              {row.label}
            </span>
            <span
              data-testid={`habit-checklist-status-${row.key}`}
              className="font-mono text-[10px] uppercase tracking-wider"
              style={{ color }}
            >
              {row.status}
            </span>
          </span>
          <span className="truncate text-[11.5px] text-[var(--text-muted)]">
            {row.description}
          </span>
        </span>
        <span className="flex shrink-0 flex-col items-end gap-0.5">
          <span className="font-[Antonio] text-2xl font-semibold leading-none text-[var(--text)] tabular-nums">
            {valueText}
          </span>
          <span className="font-mono text-[10.5px] uppercase tracking-wide text-[var(--text-faint)]">
            {normText}
          </span>
          {delta ? (
            <span
              data-testid={`habit-checklist-delta-${row.key}`}
              className="font-mono text-[11px] tabular-nums"
              style={{ color: delta.improving ? "#9bbc5a" : "#f87171" }}
            >
              {delta.arrow}{" "}
              {delta.unit
                ? `${formatThreshold(delta.magnitude)} ${delta.unit}`
                : `${formatThreshold(delta.magnitude)}`}
            </span>
          ) : null}
        </span>
      </button>
    </li>
  )
}

// HabitChecklist — slice-11 in-app debrief surface. Replaces the legacy
// 4-bar category list inside VerdictHero with one row per habit, each
// carrying its own value, norm, status, and delta vs. the player's prior
// demo. Click a row to filter MistakesFeed to that category (toggle to
// clear). See plans/analysis-overhaul.md §6.1, §7.1.
export function HabitChecklist() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const selectedCategory = useAnalysisStore((s) => s.selectedCategory)
  const setSelectedCategory = useAnalysisStore((s) => s.setSelectedCategory)
  const { data, isLoading } = useHabitReport(demoId, steamId)

  const habits = useMemo(() => data?.habits ?? [], [data])

  if (isLoading) {
    return (
      <div
        data-testid="habit-checklist-loading"
        className="rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6"
      >
        <Skeleton className="mb-3 h-5 w-48 bg-white/5" />
        <Skeleton className="h-48 w-full bg-white/5" />
      </div>
    )
  }

  if (!steamId || habits.length === 0) {
    return (
      <div
        data-testid="habit-checklist-empty"
        className="rounded-xl border border-dashed border-[var(--border-strong)] bg-[var(--bg-elevated)] px-8 py-10 text-center text-sm text-[var(--text-muted)]"
      >
        Pick a player above to see their habits.
      </div>
    )
  }

  return (
    <section
      data-testid="habit-checklist"
      className="rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6"
    >
      <header className="mb-4 flex flex-col gap-1.5">
        <div className="flex items-center gap-3 text-[10.5px] font-semibold uppercase tracking-[0.18em] text-[var(--text-subtle)]">
          <span
            aria-hidden="true"
            className="inline-block h-px w-8 bg-[var(--border-strong)]"
          />
          <span>B · Habits</span>
        </div>
        <h3
          className="text-[15px] font-semibold leading-tight text-[var(--text)]"
          style={{ fontFamily: "'Inter Tight', Inter, sans-serif" }}
        >
          The habits that decide duels
        </h3>
      </header>

      <ul className="flex flex-col gap-1.5">
        {habits.map((row) => {
          const cat = HABIT_CATEGORY[row.key]
          const active = selectedCategory === cat
          return (
            <HabitChecklistRow
              key={row.key}
              row={row}
              active={active}
              onToggle={() => setSelectedCategory(active ? null : cat)}
            />
          )
        })}
      </ul>
    </section>
  )
}
