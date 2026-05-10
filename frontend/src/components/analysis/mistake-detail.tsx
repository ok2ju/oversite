import { useMistakeContext } from "@/hooks/use-mistake-context"
import { useViewerStore } from "@/stores/viewer"
import { Skeleton } from "@/components/ui/skeleton"
import { badgeVariants } from "@/components/ui/badge"
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

// MistakeDetail renders a single analyzer mistake's full context — title,
// severity badge, suggestion, round window, and the rule-specific extras
// blob in a key-value list. Click "Seek to play" to drive the viewer's
// PixiJS canvas to the offending tick (mirrors mistake-list.tsx's
// useViewerStore.setTick wiring).
export function MistakeDetail({ id }: MistakeDetailProps) {
  const setTick = useViewerStore((s) => s.setTick)
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
  const extras = entry.extras ?? {}

  return (
    <article
      data-testid="mistake-detail"
      className="flex flex-col gap-3 rounded border border-white/10 bg-white/5 p-3 text-white"
    >
      <header className="flex items-center justify-between gap-2">
        <h3 className="text-sm font-semibold tracking-tight">
          {entry.title || entry.kind}
        </h3>
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
      {entry.suggestion ? (
        <p
          data-testid="mistake-detail-suggestion"
          className="text-sm leading-snug text-white/80"
        >
          {entry.suggestion}
        </p>
      ) : null}
      {Object.keys(extras).length > 0 ? (
        <dl
          data-testid="mistake-detail-extras"
          className="grid grid-cols-2 gap-x-3 gap-y-1 text-xs tabular-nums"
        >
          {Object.entries(extras).map(([k, v]) => (
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
