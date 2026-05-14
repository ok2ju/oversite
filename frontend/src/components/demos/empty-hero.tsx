import { Plus } from "lucide-react"
import { useImportDemo } from "@/hooks/use-demos"

export function DemosEmptyHero() {
  const { importDemo, isImporting } = useImportDemo()

  return (
    <div
      className="empty-hero relative overflow-hidden rounded-[var(--radius-md)] border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-16 text-center"
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
      }}
    >
      <div className="relative mb-6 h-[140px] w-[220px]" aria-hidden>
        <svg
          width="220"
          height="140"
          viewBox="0 0 220 140"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <linearGradient id="empty-folder" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#1B2230" />
              <stop offset="100%" stopColor="#10141D" />
            </linearGradient>
          </defs>
          <path
            d="M30 40 h50 l10 12 h100 a6 6 0 0 1 6 6 v52 a6 6 0 0 1 -6 6 h-160 a6 6 0 0 1 -6 -6 v-64 a6 6 0 0 1 6 -6 z"
            fill="url(#empty-folder)"
            stroke="var(--border-strong)"
            strokeWidth="1.5"
          />
          <rect
            x="50"
            y="62"
            width="48"
            height="56"
            rx="4"
            fill="var(--bg-sunken)"
            stroke="var(--border-strong)"
            strokeWidth="1.5"
          />
          <rect
            x="54"
            y="68"
            width="30"
            height="3"
            rx="1.5"
            fill="var(--border-strong)"
          />
          <rect
            x="54"
            y="74"
            width="40"
            height="2"
            rx="1"
            fill="var(--border)"
          />
          <rect
            x="54"
            y="80"
            width="24"
            height="2"
            rx="1"
            fill="var(--border)"
          />
          <circle cx="74" cy="100" r="8" fill="var(--accent)" />
          <polygon points="71 96, 79 100, 71 104" fill="#0B0D10" />
          <rect
            x="108"
            y="58"
            width="48"
            height="56"
            rx="4"
            fill="var(--bg-sunken)"
            stroke="var(--border-strong)"
            strokeWidth="1.5"
          />
          <rect
            x="112"
            y="64"
            width="30"
            height="3"
            rx="1.5"
            fill="var(--border-strong)"
          />
          <rect
            x="112"
            y="70"
            width="40"
            height="2"
            rx="1"
            fill="var(--border)"
          />
          <circle cx="132" cy="94" r="8" fill="var(--accent)" opacity="0.7" />
          <polygon points="129 90, 137 94, 129 98" fill="#0B0D10" />
          <path
            d="M180 25 l3 7 l7 3 l-7 3 l-3 7 l-3-7 l-7-3 l7-3z"
            fill="var(--warn)"
          />
          <circle cx="200" cy="60" r="3" fill="var(--warn)" opacity="0.7" />
          <circle cx="25" cy="30" r="2" fill="var(--accent)" />
        </svg>
      </div>

      <div className="text-[22px] font-bold leading-tight tracking-[-0.01em] text-[var(--text)]">
        Add your first demo
      </div>
      <div className="mt-2 max-w-[460px] text-[13.5px] leading-[1.55] text-[var(--text-muted)]">
        Drop a .dem file in to start analyzing. Oversite parses the match, lays
        out the round-by-round timeline, and lets you scrub through duels and
        positioning.
      </div>

      <div className="mt-6 flex gap-2.5">
        <button
          type="button"
          className="btn-sm primary"
          onClick={() => importDemo()}
          disabled={isImporting}
        >
          <Plus className="h-3 w-3" />
          {isImporting ? "Importing…" : "Import demo"}
        </button>
        <button type="button" className="btn-sm" disabled>
          Watch a folder
        </button>
      </div>

      <div className="mt-10 grid w-full max-w-[680px] grid-cols-3 gap-4">
        <EmptyStep
          n={1}
          title="Import a demo"
          sub="Drop a .dem file, paste a HLTV/Faceit link, or watch a folder."
        />
        <EmptyStep
          n={2}
          title="Pick a demo folder"
          sub="Point at your CS2 replays directory — usually auto-detected."
        />
        <EmptyStep
          n={3}
          title="Review and watch"
          sub="Jump into any match, open the demo, and scrub to key rounds."
        />
      </div>
    </div>
  )
}

function EmptyStep({
  n,
  title,
  sub,
}: {
  n: number
  title: string
  sub: string
}) {
  return (
    <div className="rounded-lg border border-[var(--border)] bg-[var(--bg-elevated)] p-3.5 text-left">
      <div
        className="mb-2.5 inline-flex h-[22px] w-[22px] items-center justify-center rounded-[6px] text-[11px] font-bold"
        style={{ background: "var(--accent-soft)", color: "var(--accent-ink)" }}
      >
        {n}
      </div>
      <div className="text-[13px] font-semibold text-[var(--text)]">
        {title}
      </div>
      <div className="mt-0.5 text-[12px] text-[var(--text-muted)]">{sub}</div>
    </div>
  )
}
