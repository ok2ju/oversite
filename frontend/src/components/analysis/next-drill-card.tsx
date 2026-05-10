import { useViewerStore } from "@/stores/viewer"
import { useNextDrill } from "@/hooks/use-next-drill"
import { Skeleton } from "@/components/ui/skeleton"

// NextDrillCard — closes the analysis page with a single prescribed drill
// (P1-3). Backend computes the worst-status habit, picks the matching
// drill, and ships title/why/duration/chips. An empty key means the
// maintenance fallback (no bad/warn habit found) — render a softer "keep
// your routine" framing.
export function NextDrillCard() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const { data, isLoading } = useNextDrill(demoId, steamId)

  if (!steamId) {
    return (
      <section
        data-testid="next-drill-card-empty"
        className="rounded-xl border border-dashed border-[var(--border-strong)] bg-[var(--bg-elevated)] px-6 py-8 text-center text-sm text-[var(--text-muted)]"
      >
        Pick a player above to see your next drill.
      </section>
    )
  }

  if (isLoading) {
    return (
      <section
        data-testid="next-drill-card-loading"
        className="rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6"
      >
        <Skeleton className="mb-3 h-5 w-40 bg-white/5" />
        <Skeleton className="h-20 w-full bg-white/5" />
      </section>
    )
  }

  if (!data) return null

  const isMaintenance = data.key === ""

  return (
    <section
      data-testid="next-drill-card"
      data-maintenance={isMaintenance ? "true" : undefined}
      className="relative overflow-hidden rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6"
    >
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0"
        style={{
          background: isMaintenance
            ? "radial-gradient(360px 140px at 12% -10%, rgba(155,188,90,0.10), transparent 65%)"
            : "radial-gradient(440px 160px at 12% -10%, rgba(255,122,26,0.14), transparent 65%)",
        }}
      />

      <header className="relative mb-3 flex items-end justify-between gap-4">
        <div className="flex items-center gap-3 text-[10.5px] font-semibold uppercase tracking-[0.18em] text-[var(--text-subtle)]">
          <span
            aria-hidden="true"
            className="inline-block h-px w-8 bg-[var(--border-strong)]"
          />
          <span>D · Next drill</span>
        </div>
        <span
          data-testid="next-drill-card-duration"
          className="font-mono text-[11px] uppercase tracking-wide text-[var(--text-faint)]"
        >
          ▶ {data.duration}
        </span>
      </header>

      <h3
        data-testid="next-drill-card-title"
        className="relative text-[20px] font-semibold leading-tight tracking-tight text-[var(--text)]"
        style={{ fontFamily: "'Inter Tight', Inter, sans-serif" }}
      >
        {data.title}
      </h3>

      {data.why ? (
        <p
          data-testid="next-drill-card-why"
          className="relative mt-2 max-w-[60ch] text-[13.5px] leading-snug text-[var(--text-muted)]"
        >
          {data.why}
        </p>
      ) : null}

      {data.chips.length > 0 ? (
        <ul
          data-testid="next-drill-card-chips"
          className="relative mt-4 flex flex-wrap items-center gap-1.5"
        >
          {data.chips.map((chip) => (
            <li
              key={chip}
              className="rounded-full border border-[var(--border-strong)] bg-white/[0.03] px-2.5 py-1 text-[10.5px] font-semibold uppercase tracking-wide text-[var(--text-muted)]"
            >
              {chip}
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  )
}
