import { Link } from "react-router-dom"
import type { MistakeKindCount } from "@/types/analysis"

// Fire-related mistake kinds + the human copy the coaching errors strip
// renders. Mirrors the in-app MistakeKind enum (analyzer.go) but only lists
// kinds that derive from a fired bullet — these are the ones the strip is
// scoped to per plan §3.2.
const FIRE_RELATED_KINDS: Array<{
  kind: string
  label: string
  whyItHurts: string
}> = [
  {
    kind: "missed_first_shot",
    label: "Missed first shot",
    whyItHurts:
      "The first bullet is your most accurate one — miss it and you're spraying into recoil to recover.",
  },
  {
    kind: "shot_while_moving",
    label: "Shot while moving",
    whyItHurts:
      "First-bullet accuracy collapses past ~25 u/s of drift, so the duel is decided before the spray.",
  },
  {
    kind: "no_counter_strafe",
    label: "No counter-strafe",
    whyItHurts:
      "Without a counter-strafe your rifle's first-bullet cone is closer to a deagle's than a tap kill.",
  },
  {
    kind: "slow_reaction",
    label: "Slow reaction",
    whyItHurts:
      "If you fire 100 ms after the enemy, you've already eaten the bullet that decides the duel.",
  },
  {
    kind: "spray_decay",
    label: "Spray decay",
    whyItHurts:
      "Past shot 5 the cone is so wide most bullets miss — you're just feeding ammo into a wall.",
  },
]

interface ErrorsStripProps {
  errors: MistakeKindCount[]
  // The most recent demo's id — used to anchor the "see duel-by-duel" CTA.
  // Empty when no analyzed demos exist; CTA is hidden in that case.
  latestDemoId: string
}

export function ErrorsStrip({ errors, latestDemoId }: ErrorsStripProps) {
  const counts = new Map(errors.map((e) => [e.kind, e.total]))
  const rows = FIRE_RELATED_KINDS.map((meta) => ({
    ...meta,
    total: counts.get(meta.kind) ?? 0,
  })).filter((row) => row.total > 0)

  if (rows.length === 0) {
    return (
      <section
        data-testid="errors-strip-empty"
        className="rounded-xl border border-dashed border-[var(--border-strong)] bg-[var(--bg-elevated)] px-6 py-6 text-center text-sm text-[var(--text-muted)]"
      >
        No flagged plays in your last few demos — clean.
      </section>
    )
  }

  return (
    <section
      data-testid="errors-strip"
      className="flex flex-col gap-4 rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6"
    >
      <header className="flex flex-col gap-1.5">
        <div className="flex items-center gap-3 text-[10.5px] font-semibold uppercase tracking-[0.18em] text-[var(--text-subtle)]">
          <span
            aria-hidden="true"
            className="inline-block h-px w-8 bg-[var(--border-strong)]"
          />
          <span>What we found</span>
        </div>
        <h3
          className="text-[15px] font-semibold leading-tight text-[var(--text)]"
          style={{ fontFamily: "'Inter Tight', Inter, sans-serif" }}
        >
          Errors flagged in your fights
        </h3>
      </header>

      <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {rows.map((row) => (
          <li
            key={row.kind}
            data-testid={`errors-strip-card-${row.kind}`}
            className="flex flex-col gap-1 rounded-lg border border-[var(--border)] bg-white/[0.02] px-4 py-3"
          >
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-[12px] font-semibold uppercase tracking-wide text-[var(--text)]">
                {row.label}
              </span>
              <span
                data-testid={`errors-strip-count-${row.kind}`}
                className="font-[Antonio] text-[20px] font-semibold leading-none text-[var(--text)] tabular-nums"
              >
                {row.total}
              </span>
            </div>
            <span className="text-[11px] leading-snug text-[var(--text-muted)]">
              {row.whyItHurts}
            </span>
          </li>
        ))}
      </ul>

      {latestDemoId ? (
        <footer className="flex justify-end pt-1">
          <Link
            to={`/demos/${latestDemoId}/analysis`}
            data-testid="errors-strip-cta"
            className="font-mono text-[11px] uppercase tracking-wider text-[var(--accent)] hover:text-[var(--accent-strong)]"
          >
            See the duel-by-duel breakdown →
          </Link>
        </footer>
      ) : null}
    </section>
  )
}
