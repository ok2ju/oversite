import { Link2 } from "lucide-react"
import { Button } from "@/components/ui/button"

interface EmptyHeroProps {
  onConnect: () => void
  onSkip: () => void
  isLoading: boolean
  error?: string | null
}

const STEPS: Array<{ n: number; title: string; sub: string }> = [
  {
    n: 1,
    title: "Sign in",
    sub: "Connect your Faceit account through OAuth.",
  },
  {
    n: 2,
    title: "Pick a folder",
    sub: "Point Oversite at your CS2 replays folder.",
  },
  {
    n: 3,
    title: "Review & watch",
    sub: "Browse recent matches and scrub any demo locally.",
  },
]

function Illustration() {
  return (
    <svg
      width="140"
      height="120"
      viewBox="0 0 140 120"
      aria-hidden
      className="mx-auto mb-4"
    >
      <defs>
        <linearGradient id="folder-g" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#ff8f3d" />
          <stop offset="1" stopColor="#b44500" />
        </linearGradient>
      </defs>
      <path
        d="M18 34 h34 l10 10 h60 a6 6 0 0 1 6 6 v46 a6 6 0 0 1 -6 6 h-104 a6 6 0 0 1 -6 -6 v-56 a6 6 0 0 1 6 -6 z"
        fill="url(#folder-g)"
      />
      <rect
        x="34"
        y="52"
        width="66"
        height="36"
        rx="4"
        fill="#14171c"
        stroke="#30363f"
        strokeWidth="1"
        opacity="0.96"
      />
      <rect
        x="44"
        y="58"
        width="66"
        height="36"
        rx="4"
        fill="#1b1f25"
        stroke="#30363f"
        strokeWidth="1"
        opacity="0.96"
      />
      <polygon points="74,68 94,82 74,96" fill="#ff7a1a" />
      <circle cx="118" cy="28" r="3" fill="#ff7a1a" />
      <circle cx="10" cy="62" r="2" fill="#ffb266" />
      <path d="M128 46 l2 4 4 2 -4 2 -2 4 -2 -4 -4 -2 4 -2 z" fill="#fbbf24" />
    </svg>
  )
}

export function EmptyHero({
  onConnect,
  onSkip,
  isLoading,
  error,
}: EmptyHeroProps) {
  return (
    <div
      className="grid min-h-screen place-items-center p-8"
      style={{ background: "var(--bg)" }}
    >
      <div className="w-full max-w-[640px]">
        <div
          className="empty-hero rounded-[10px] border px-10 py-10 text-center"
          style={{ borderColor: "var(--border)" }}
        >
          <Illustration />
          <h1
            className="text-[22px] font-bold text-[var(--text)]"
            data-testid="empty-hero-title"
          >
            Connect Faceit to get started
          </h1>
          <p className="mx-auto mt-2 max-w-[440px] text-[13.5px] text-[var(--text-muted)]">
            Oversite auto-syncs your match history, watches your CS2 replays
            folder, and lets you scrub demos without leaving the app.
          </p>

          {error ? (
            <p className="mt-3 text-[12.5px] text-[var(--loss)]">{error}</p>
          ) : null}

          <div className="mt-6 flex items-center justify-center gap-2">
            <Button
              size="sm"
              className="gap-1.5"
              onClick={onConnect}
              disabled={isLoading}
            >
              <Link2 className="h-3.5 w-3.5" />
              {isLoading ? "Connecting…" : "Connect Faceit account"}
            </Button>
            <Button variant="ghost" size="sm" onClick={onSkip}>
              I'll do this later
            </Button>
          </div>
        </div>

        <div className="mt-6 grid grid-cols-3 gap-3">
          {STEPS.map((s) => (
            <div
              key={s.n}
              className="rounded-md border px-4 py-3"
              style={{
                borderColor: "var(--border)",
                background: "var(--bg-elevated)",
              }}
            >
              <div
                className="grid h-6 w-6 place-items-center rounded-full text-[11px] font-bold"
                style={{
                  background: "var(--accent-soft)",
                  color: "var(--accent-ink)",
                }}
              >
                {s.n}
              </div>
              <div className="mt-2 text-[13px] font-bold text-[var(--text)]">
                {s.title}
              </div>
              <div className="mt-0.5 text-[11.5px] text-[var(--text-muted)]">
                {s.sub}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
