import { useMemo } from "react"
import { cn } from "@/lib/utils"
import { Sparkline, type SparklinePoint } from "@/components/ui/sparkline"
import type { CoachingHabitRow, HabitStatus } from "@/types/analysis"

// Status color set — same swatches as the in-app habit checklist (plan §2.6).
const STATUS_COLOR: Record<HabitStatus, string> = {
  good: "#9bbc5a",
  warn: "#ffc233",
  bad: "#f87171",
}

function formatValue(row: CoachingHabitRow): {
  value: string
  unit: string
} {
  const v = row.value
  if (row.unit === "%") {
    return { value: `${Math.round(v)}`, unit: "%" }
  }
  if (row.unit === "ms") {
    // ms-scale habits surface as seconds at the card scale (e.g. "0.12 s")
    // — the analysis page renders ms, but at a coaching glance the human
    // language is seconds. See plan §7.3.
    return { value: (v / 1000).toFixed(2), unit: "s" }
  }
  if (!row.unit) {
    return { value: `${Math.round(v)}`, unit: "" }
  }
  return { value: `${Math.round(v)}`, unit: row.unit }
}

function formatNorm(row: CoachingHabitRow): string {
  if (row.direction === "lower") {
    return `norm ≤ ${formatThreshold(row.good_threshold, row.unit)}`
  }
  if (row.direction === "higher") {
    return `norm ≥ ${formatThreshold(row.good_threshold, row.unit)}`
  }
  // Balanced: render the range once with a trailing unit so we don't repeat
  // the "%" / "ms" on both sides.
  const lo = formatNumber(row.good_min)
  const hi = formatNumber(row.good_max)
  const unitSuffix = row.unit === "ms" ? " s" : row.unit ? ` ${row.unit}` : ""
  if (row.unit === "ms") {
    return `norm ${(row.good_min / 1000).toFixed(2)}–${(row.good_max / 1000).toFixed(2)}${unitSuffix}`
  }
  return `norm ${lo}–${hi}${unitSuffix}`
}

function formatThreshold(n: number, unit: string): string {
  if (unit === "ms") {
    // Match the value formatter: ms surfaces as seconds at the card scale.
    return `${(n / 1000).toFixed(2)} s`
  }
  if (unit) {
    return `${formatNumber(n)} ${unit}`
  }
  return formatNumber(n)
}

function formatNumber(n: number): string {
  if (Number.isInteger(n)) return `${n}`
  return `${Number(n.toFixed(2))}`
}

interface MicroCardProps {
  row: CoachingHabitRow
  // Optional click handler — slide-in detail panel will plug in here once
  // the contributing-mistakes panel ships. Acceptance for P5-3 only requires
  // the card to be a button so onClick can be wired without re-render.
  onClick?: () => void
}

export function MicroCard({ row, onClick }: MicroCardProps) {
  const color = STATUS_COLOR[row.status]
  const formatted = formatValue(row)
  const norm = formatNorm(row)
  const hasTrend = row.trend && row.trend.length >= 2

  // Sparkline points come newest-first off the wire; the primitive reverses
  // so the line reads left-to-right (oldest → newest).
  const trendPoints = useMemo<SparklinePoint[]>(
    () =>
      (row.trend ?? []).map((p) => ({
        value: p.value,
        label: p.match_date,
      })),
    [row.trend],
  )

  const Tag = onClick ? "button" : "div"

  return (
    <Tag
      data-testid={`micro-card-${row.key}`}
      data-status={row.status}
      type={onClick ? "button" : undefined}
      onClick={onClick}
      className={cn(
        "group relative flex flex-col gap-3 rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-5 py-5 text-left transition-colors",
        onClick && "hover:border-[var(--border-strong)] hover:bg-white/[0.04]",
      )}
    >
      <span
        aria-hidden="true"
        className="absolute left-0 top-4 bottom-4 w-1 rounded-r-full"
        style={{ backgroundColor: color }}
      />
      <header className="flex flex-col gap-0.5 pl-2">
        <span className="text-[10.5px] font-semibold uppercase tracking-[0.18em] text-[var(--text-subtle)]">
          {row.label}
        </span>
        <span className="truncate text-[11.5px] text-[var(--text-muted)]">
          {row.description}
        </span>
      </header>

      <div className="flex items-baseline gap-1.5 pl-2">
        <span
          className="font-[Antonio] text-[34px] font-semibold leading-none text-[var(--text)] tabular-nums"
          data-testid={`micro-card-value-${row.key}`}
        >
          {formatted.value}
        </span>
        {formatted.unit ? (
          <span className="font-mono text-[12px] uppercase tracking-wide text-[var(--text-faint)]">
            {formatted.unit}
          </span>
        ) : null}
      </div>

      <footer className="flex items-end justify-between gap-3 pl-2">
        <div className="flex flex-col gap-0.5">
          <span
            data-testid={`micro-card-norm-${row.key}`}
            className="font-mono text-[10.5px] uppercase tracking-wide text-[var(--text-faint)]"
          >
            {norm}
          </span>
          <span
            data-testid={`micro-card-status-${row.key}`}
            className="font-mono text-[10.5px] uppercase tracking-wider"
            style={{ color }}
          >
            {row.status}
          </span>
        </div>
        {hasTrend ? (
          <Sparkline
            points={trendPoints}
            color={color}
            ariaLabel={`${row.label} trend`}
            width={80}
            height={18}
            className="shrink-0"
          />
        ) : (
          <span
            data-testid={`micro-card-trend-empty-${row.key}`}
            className="font-mono text-[9.5px] uppercase tracking-wider text-[var(--text-faint)]"
          >
            first demo
          </span>
        )}
      </footer>
    </Tag>
  )
}
