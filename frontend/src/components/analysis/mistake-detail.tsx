import { useMistakeContext } from "@/hooks/use-mistake-context"
import { useViewerStore } from "@/stores/viewer"
import { useAnalysisStore } from "@/stores/analysis"
import { useUiStore } from "@/stores/ui"
import { Skeleton } from "@/components/ui/skeleton"
import { badgeVariants } from "@/components/ui/badge"
import { TickSpeedBar } from "@/components/analysis/tick-speed-bar"
import { MousePath } from "@/components/analysis/mouse-path"
import { cn } from "@/lib/utils"

interface MistakeDetailProps {
  // The mistake row's ID. null disables the query and renders the empty
  // state — the parent (analysis page or side-panel detail row) clears the
  // selection by passing null when no row is active.
  id: number | null
}

const SEVERITY_LABEL: Record<number, string> = {
  1: "Low",
  2: "Med",
  3: "High",
}

const SEVERITY_BADGE_VARIANT: Record<
  number,
  "secondary" | "default" | "destructive"
> = {
  1: "secondary",
  2: "default",
  3: "destructive",
}

// CAUSE_HEADLINE maps the analyzer's cause_tag (set on fire-related mistake
// extras_json by P2-1) to a short headline appended to the mistake title.
// Keeping the mapping client-side mirrors how SEVERITY_LABEL works — these
// are presentation strings, not data; a translation rotation lands here, not
// in the analyzer.
const CAUSE_HEADLINE: Record<string, string> = {
  shot_before_stop: "shot before stop",
  no_counter_strafe: "no counter-strafe",
  crouch_before_shot: "crouched mid-fight",
  over_flick: "over-flicked",
  under_flick: "under-flicked",
  late_reaction: "late reaction",
}

// Numeric extras keys that the speed-bar and forensic header consume. Keep
// them out of the generic key/value list at the bottom so we don't render
// the same data twice.
const FORENSIC_EXTRAS_KEYS = new Set([
  "fire_tick",
  "speed_at_fire",
  "weapon_speed_cap",
  "ticks_window",
  "speeds",
  "yaw_path",
  "pitch_path",
  "cause_tag",
])

// MistakeDetail renders a single analyzer mistake's full context — title,
// severity badge, "why it hurts" caption, the per-tick speed bar (P2-2),
// co-occurring chips (P2-3), and the rule-specific extras blob in a
// key-value list. Click "Seek to play" to drive the viewer's PixiJS canvas
// to the offending tick (mirrors mistake-list.tsx's useViewerStore.setTick
// wiring); click a co-occurring chip to swap the pinned mistake.
export function MistakeDetail({ id }: MistakeDetailProps) {
  const setTick = useViewerStore((s) => s.setTick)
  const setSelectedMistakeId = useAnalysisStore((s) => s.setSelectedMistakeId)
  const advancedOpen = useUiStore((s) => s.mistakeAdvancedOpen)
  const toggleAdvancedOpen = useUiStore((s) => s.toggleMistakeAdvancedOpen)
  const { data, isLoading } = useMistakeContext(id)

  if (id == null) {
    return (
      <p data-testid="mistake-detail-empty" className="text-sm text-white/60">
        Select a mistake to see details.
      </p>
    )
  }
  if (isLoading) {
    return (
      <div data-testid="mistake-detail-loading" className="flex flex-col gap-2">
        <Skeleton className="h-6 w-1/2 bg-white/10" />
        <Skeleton className="h-4 w-3/4 bg-white/10" />
        <Skeleton className="h-12 w-full bg-white/10" />
      </div>
    )
  }
  if (!data) {
    return (
      <p data-testid="mistake-detail-missing" className="text-sm text-white/60">
        Mistake not found.
      </p>
    )
  }

  const { entry } = data
  const severityLabel = SEVERITY_LABEL[entry.severity] ?? "Low"
  const severityVariant = SEVERITY_BADGE_VARIANT[entry.severity] ?? "secondary"
  const extras = (entry.extras ?? {}) as Record<string, unknown>
  const causeTag =
    typeof extras.cause_tag === "string" ? extras.cause_tag : null
  const causeHeadline = causeTag ? CAUSE_HEADLINE[causeTag] : null
  const speeds = Array.isArray(extras.speeds)
    ? (extras.speeds as unknown[]).filter(
        (v): v is number => typeof v === "number",
      )
    : []
  const yaws = Array.isArray(extras.yaw_path)
    ? (extras.yaw_path as unknown[]).filter(
        (v): v is number => typeof v === "number",
      )
    : []
  const pitches = Array.isArray(extras.pitch_path)
    ? (extras.pitch_path as unknown[]).filter(
        (v): v is number => typeof v === "number",
      )
    : null
  const ticksWindow = Array.isArray(extras.ticks_window)
    ? (extras.ticks_window as unknown[]).filter(
        (v): v is number => typeof v === "number",
      )
    : undefined
  const weaponSpeedCap =
    typeof extras.weapon_speed_cap === "number" ? extras.weapon_speed_cap : 40
  // Mouse-path expander is hidden when we have neither yaw nor pitch — the
  // SVG would render only the rings and a dot at center, which is more
  // confusing than absent. Speed alone has its own bar so we don't surface
  // a degenerate variant here.
  const hasMousePath = yaws.length > 0 && speeds.length > 0
  const coOccurring = data.co_occurring ?? []
  // Filter the extras blob to fields that aren't already rendered above.
  const visibleExtras = Object.entries(extras).filter(
    ([k]) => !FORENSIC_EXTRAS_KEYS.has(k),
  )

  return (
    <article
      data-testid="mistake-detail"
      className="flex flex-col gap-3 rounded border border-white/10 bg-white/5 p-3 text-white"
    >
      <header className="flex items-start justify-between gap-2">
        <div className="flex flex-col gap-1">
          <h3
            data-testid="mistake-detail-title"
            className="text-sm font-semibold tracking-tight"
          >
            {entry.title || entry.kind}
            {causeHeadline ? (
              <span
                data-testid="mistake-detail-cause"
                className="ml-1.5 text-white/60"
              >
                — {causeHeadline}
              </span>
            ) : null}
          </h3>
          {entry.why_it_hurts ? (
            <p
              data-testid="mistake-detail-why"
              className="text-xs leading-snug text-white/70"
            >
              {entry.why_it_hurts}
            </p>
          ) : null}
        </div>
        <span
          data-testid="mistake-detail-severity"
          className={cn(badgeVariants({ variant: severityVariant }))}
        >
          {severityLabel}
        </span>
      </header>
      <p className="text-xs uppercase tracking-wide text-white/50">
        {entry.category} • Round {entry.round_number}
      </p>
      {speeds.length > 0 ? (
        <TickSpeedBar
          speeds={speeds}
          weaponSpeedCap={weaponSpeedCap}
          ticksWindow={ticksWindow}
        />
      ) : null}
      {hasMousePath ? (
        <section
          data-testid="mistake-detail-advanced"
          className="flex flex-col gap-2 rounded border border-white/10 bg-black/20 p-2"
        >
          <button
            type="button"
            data-testid="mistake-detail-advanced-toggle"
            aria-expanded={advancedOpen}
            aria-controls="mistake-detail-advanced-panel"
            onClick={toggleAdvancedOpen}
            className="flex w-full items-center justify-between text-[11px] uppercase tracking-wide text-white/60 hover:text-white/90 focus:outline-none focus-visible:ring-2 focus-visible:ring-white/40"
          >
            <span>Advanced — mouse path</span>
            <span aria-hidden className="font-mono text-white/40">
              {advancedOpen ? "−" : "+"}
            </span>
          </button>
          {advancedOpen ? (
            <div
              id="mistake-detail-advanced-panel"
              data-testid="mistake-detail-advanced-panel"
            >
              <MousePath
                yaws={yaws}
                speeds={speeds}
                pitches={pitches}
                weaponSpeedCap={weaponSpeedCap}
              />
            </div>
          ) : null}
        </section>
      ) : null}
      {coOccurring.length > 0 ? (
        <section
          data-testid="mistake-detail-co-occurring"
          aria-label="Other mistakes in this moment"
          className="flex flex-col gap-1"
        >
          <span className="text-[10px] uppercase tracking-wide text-white/50">
            Also in this moment
          </span>
          <div className="flex flex-wrap gap-1">
            {coOccurring.map((c) => (
              <button
                key={c.id}
                type="button"
                data-testid={`mistake-detail-co-chip-${c.id}`}
                onClick={() => setSelectedMistakeId(c.id)}
                className="rounded-full border border-white/15 bg-white/10 px-2 py-0.5 text-[11px] hover:bg-white/20 focus:outline-none focus-visible:ring-2 focus-visible:ring-white/40"
              >
                {c.title || c.kind}
              </button>
            ))}
          </div>
        </section>
      ) : null}
      {entry.suggestion ? (
        <p
          data-testid="mistake-detail-suggestion"
          className="text-sm leading-snug text-white/80"
        >
          {entry.suggestion}
        </p>
      ) : null}
      {visibleExtras.length > 0 ? (
        <dl
          data-testid="mistake-detail-extras"
          className="grid grid-cols-2 gap-x-3 gap-y-1 text-xs tabular-nums"
        >
          {visibleExtras.map(([k, v]) => (
            <div key={k} className="contents">
              <dt className="text-white/50">{k}</dt>
              <dd className="text-right">{formatExtra(v)}</dd>
            </div>
          ))}
        </dl>
      ) : null}
      <button
        type="button"
        data-testid="mistake-detail-seek"
        onClick={() => setTick(entry.tick)}
        className="self-start rounded border border-white/15 bg-white/10 px-2 py-1 text-xs hover:bg-white/20 focus:outline-none focus-visible:ring-2 focus-visible:ring-white/40"
      >
        Seek to play
      </button>
    </article>
  )
}

function formatExtra(v: unknown): string {
  if (v == null) return "—"
  if (typeof v === "number") {
    return Number.isInteger(v) ? `${v}` : v.toFixed(2)
  }
  if (typeof v === "string") return v
  if (typeof v === "boolean") return v ? "true" : "false"
  if (Array.isArray(v)) return v.map(formatExtra).join(", ")
  return JSON.stringify(v)
}
