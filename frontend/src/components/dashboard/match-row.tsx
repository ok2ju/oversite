import { useEffect, useState } from "react"
import { Download } from "lucide-react"
import { EventsOff, EventsOn } from "@wailsjs/runtime/runtime"
import { cn } from "@/lib/utils"
import { MapTile, resolveMap } from "@/components/dashboard/map-tile"
import { useDemoDownload } from "@/hooks/use-demo-download"
import { useDemoStore } from "@/stores/demo"
import type { FaceitMatch } from "@/types/faceit"

interface DownloadProgress {
  bytesDownloaded: number
  totalBytes: number
}

function relativeTime(iso: string): string {
  const now = Date.now()
  const ts = new Date(iso).getTime()
  const diffMs = now - ts
  const mins = Math.round(diffMs / 60_000)
  if (mins < 60) return `${Math.max(1, mins)}m ago`
  const hrs = Math.round(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  const days = Math.round(hrs / 24)
  if (days === 1) return "Yesterday"
  if (days < 7) return `${days}d ago`
  return new Date(iso).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  })
}

interface MatchRowProps {
  match: FaceitMatch
  onClick?: (match: FaceitMatch) => void
}

function StatCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col items-center px-2">
      <div className="text-[9.5px] font-semibold uppercase tracking-wider text-[var(--text-subtle)]">
        {label}
      </div>
      <div className="tabular text-[12.5px] font-semibold text-[var(--text)] leading-tight">
        {value}
      </div>
    </div>
  )
}

export function MatchRow({ match, onClick }: MatchRowProps) {
  const win = match.result === "W"
  const meta = resolveMap(match.map_name)

  const kills = match.kills ?? 0
  const deaths = match.deaths ?? 0
  const assists = match.assists ?? 0
  const kd = deaths > 0 ? (kills / deaths).toFixed(2) : "—"
  const kMinusD = kills - deaths
  const diff = kMinusD > 0 ? `+${kMinusD}` : `${kMinusD}`

  const canOpen = match.has_demo && match.demo_id != null
  const canImport = !match.has_demo && !!match.demo_url

  const download = useDemoDownload()
  const [downloadProgress, setDownloadProgress] =
    useState<DownloadProgress | null>(null)

  useEffect(() => {
    if (!download.isPending) {
      setDownloadProgress(null)
      return
    }
    const cancel = EventsOn(
      "faceit:demo:download:progress",
      (data: DownloadProgress) => setDownloadProgress(data),
    )
    return () => {
      EventsOff("faceit:demo:download:progress")
      if (typeof cancel === "function") cancel()
    }
  }, [download.isPending])

  const importProgress = useDemoStore((s) => s.importProgress)
  const parsingThisDemo =
    canOpen &&
    importProgress != null &&
    String(importProgress.demoId) === match.demo_id &&
    (importProgress.stage === "parsing" || importProgress.stage === "importing")

  const [waitingNav, setWaitingNav] = useState(false)

  useEffect(() => {
    if (!waitingNav) return
    if (
      importProgress?.demoId != null &&
      String(importProgress.demoId) === match.demo_id &&
      importProgress.stage === "complete"
    ) {
      setWaitingNav(false)
      onClick?.(match)
    }
  }, [waitingNav, importProgress, match, onClick])

  function handleRowClick() {
    if (!canOpen) return
    if (parsingThisDemo) {
      setWaitingNav(true)
      return
    }
    onClick?.(match)
  }

  function handleImport(e: React.MouseEvent) {
    e.stopPropagation()
    download.mutate(match.faceit_match_id)
  }

  const interactive = canOpen

  return (
    <div
      role={interactive ? "button" : undefined}
      tabIndex={interactive ? 0 : undefined}
      onClick={handleRowClick}
      onKeyDown={(e) => {
        if (!interactive) return
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault()
          handleRowClick()
        }
      }}
      data-testid={`match-row-${match.id}`}
      className={cn(
        "grid w-full grid-cols-[16px_1fr_auto] items-center gap-[18px] px-4 py-[14px] text-left transition-colors",
        interactive && "cursor-pointer hover:bg-[var(--bg-hover)]",
      )}
    >
      <span
        aria-label={win ? "Win" : "Loss"}
        className="h-2 w-2 rounded-full"
        style={{
          background: win ? "var(--win)" : "var(--loss)",
          boxShadow: `0 0 0 3px ${win ? "var(--win-soft)" : "var(--loss-soft)"}`,
        }}
      />

      <div className="flex min-w-0 items-center gap-3">
        <MapTile mapName={match.map_name} size={36} />
        <div className="min-w-0">
          <div className="truncate text-[13.5px] font-semibold text-[var(--text)]">
            {meta.name}
          </div>
          <div className="text-[11.5px] text-[var(--text-muted)]">5v5</div>
        </div>

        <div className="tabular ml-5 flex items-baseline gap-1.5">
          <span
            className="text-[22px] font-bold leading-none"
            style={{ color: win ? "var(--win)" : "var(--loss)" }}
          >
            {match.score_team}
          </span>
          <span className="text-[13px] text-[var(--text-subtle)]">:</span>
          <span className="text-[22px] font-bold leading-none text-[var(--text-faint)]">
            {match.score_opponent}
          </span>
        </div>

        <div className="ml-6 hidden items-center divide-x divide-[var(--divider)] md:flex">
          <StatCell label="K/D" value={kd} />
          <StatCell label="K-D" value={diff} />
          <StatCell label="A" value={String(assists)} />
        </div>
      </div>

      <div className="flex flex-col items-end gap-1.5">
        {download.isPending || downloadProgress ? (
          <DownloadPill progress={downloadProgress} />
        ) : parsingThisDemo || waitingNav ? (
          <ParsingPill percent={importProgress?.percent} />
        ) : canImport ? (
          <button
            type="button"
            onClick={handleImport}
            data-testid={`match-row-${match.id}-import`}
            className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-soft)] px-2.5 py-0.5 text-[11px] font-semibold text-[var(--accent-ink)] transition-colors hover:bg-[var(--accent)]/20"
          >
            <Download className="h-3 w-3" />
            Import demo
          </button>
        ) : (
          <DemoStatePill match={match} />
        )}
        <span className="text-[11.5px] text-[var(--text-subtle)]">
          {relativeTime(match.played_at)}
        </span>
      </div>
    </div>
  )
}

function DownloadPill({ progress }: { progress: DownloadProgress | null }) {
  const pct =
    progress && progress.totalBytes > 0
      ? Math.min(
          100,
          Math.round((progress.bytesDownloaded / progress.totalBytes) * 100),
        )
      : null
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-medium"
      style={{
        background: "var(--accent-soft)",
        color: "var(--accent-ink)",
      }}
    >
      <span
        className="h-1.5 w-1.5 animate-pulse rounded-full"
        style={{ background: "var(--accent)" }}
      />
      {pct != null ? `Downloading ${pct}%` : "Downloading…"}
    </span>
  )
}

function ParsingPill({ percent }: { percent?: number }) {
  const pct =
    typeof percent === "number" ? Math.max(0, Math.min(100, percent)) : null
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-medium"
      style={{
        background: "var(--accent-soft)",
        color: "var(--accent-ink)",
      }}
    >
      <span
        className="h-1.5 w-1.5 animate-pulse rounded-full"
        style={{ background: "var(--accent)" }}
      />
      {pct != null ? `Parsing ${pct}%` : "Parsing…"}
    </span>
  )
}

export function DemoStatePill({ match }: { match: FaceitMatch }) {
  const { has_demo, demo_url } = match
  const state: "ready" | "available" | "none" = has_demo
    ? "ready"
    : demo_url
      ? "available"
      : "none"

  const config = {
    ready: {
      label: "Demo ready",
      color: "var(--accent-ink)",
      bg: "var(--accent-soft)",
      dot: "var(--accent)",
    },
    available: {
      label: "Demo available",
      color: "var(--text-muted)",
      bg: "var(--bg-sunken)",
      dot: "var(--text-subtle)",
    },
    none: {
      label: "No demo",
      color: "var(--text-faint)",
      bg: "var(--bg-sunken)",
      dot: "var(--text-faint)",
    },
  }[state]

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-medium",
      )}
      style={{ background: config.bg, color: config.color }}
    >
      <span
        className="h-1.5 w-1.5 rounded-full"
        style={{ background: config.dot }}
      />
      {config.label}
    </span>
  )
}
