import { useEffect, useState } from "react"
import { Download, Loader2 } from "lucide-react"
import { EventsOff, EventsOn } from "@wailsjs/runtime/runtime"
import { cn } from "@/lib/utils"
import { MapTile, resolveMap } from "@/components/dashboard/map-tile"
import { MATCH_ROW_GRID } from "@/components/dashboard/recent-matches"
import { useDemoDownload } from "@/hooks/use-demo-download"
import { useDemoStore } from "@/stores/demo"
import type { FaceitMatch } from "@/types/faceit"

interface DownloadProgress {
  bytesDownloaded: number
  totalBytes: number
}

function formatMatchDate(iso: string): { day: string; time: string } {
  const d = new Date(iso)
  const now = new Date()
  const time = d.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
  const sameDay = d.toDateString() === now.toDateString()
  if (sameDay) return { day: "Today", time }
  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  if (d.toDateString() === yesterday.toDateString()) {
    return { day: "Yesterday", time }
  }
  const day = d.toLocaleDateString(undefined, {
    weekday: "short",
    day: "numeric",
    month: "short",
  })
  return { day, time }
}

interface MatchRowProps {
  match: FaceitMatch
  onClick?: (match: FaceitMatch) => void
}

export function MatchRow({ match, onClick }: MatchRowProps) {
  const meta = resolveMap(match.map_name)

  const kills = match.kills ?? 0
  const deaths = match.deaths ?? 0
  const assists = match.assists ?? 0
  const totalRounds = match.score_team + match.score_opponent

  const kd = deaths > 0 ? (kills / deaths).toFixed(2) : "—"
  const kr = totalRounds > 0 ? (kills / totalRounds).toFixed(2) : "—"
  const kda =
    match.kills != null && match.deaths != null && match.assists != null
      ? `${kills} / ${deaths} / ${assists}`
      : "—"
  const adr = match.adr != null ? match.adr.toFixed(1) : "—"

  const { day, time } = formatMatchDate(match.played_at)

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
    download.mutate(match.id)
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
        MATCH_ROW_GRID,
        "px-5 py-3 text-left text-[13px] text-[var(--text)] transition-colors",
        interactive && "cursor-pointer hover:bg-[var(--bg-hover)]",
      )}
    >
      <div className="leading-tight">
        <div className="text-[13px] font-medium text-[var(--text)]">{day}</div>
        <div className="text-[11.5px] text-[var(--text-muted)]">{time}</div>
      </div>

      <ScoreCell
        result={match.result}
        team={match.score_team}
        opponent={match.score_opponent}
      />

      <div className="tabular font-medium">{kda}</div>

      <div className="tabular font-medium">{adr}</div>

      <div className="tabular font-medium">{kd}</div>

      <div className="tabular font-medium">{kr}</div>

      <div className="flex min-w-0 items-center gap-2">
        <MapTile mapName={match.map_name} size={24} />
        <span className="truncate text-[13px] font-medium text-[var(--text)]">
          {meta.name}
        </span>
      </div>

      <div className="flex items-center justify-end">
        {download.isPending || downloadProgress ? (
          <DownloadPill progress={downloadProgress} />
        ) : parsingThisDemo || waitingNav ? (
          <ParsingPill percent={importProgress?.percent} />
        ) : canImport ? (
          <button
            type="button"
            onClick={handleImport}
            data-testid={`match-row-${match.id}-import`}
            className="inline-flex items-center gap-1 rounded-full bg-[var(--accent-soft)] px-2.5 py-1 text-[11px] font-semibold text-[var(--accent-ink)] transition-colors hover:bg-[var(--accent)]/20"
          >
            <Download className="h-3 w-3" />
            Import demo
          </button>
        ) : (
          <DemoStatePill match={match} />
        )}
      </div>
    </div>
  )
}

function ScoreCell({
  result,
  team,
  opponent,
}: {
  result: "W" | "L"
  team: number
  opponent: number
}) {
  const win = result === "W"
  return (
    <div className="flex items-center gap-2">
      <span
        aria-label={win ? "Win" : "Loss"}
        className="inline-grid h-6 w-6 place-items-center rounded-[6px] text-[11px] font-bold text-white"
        style={{
          background: win ? "var(--win)" : "var(--loss)",
        }}
      >
        {result}
      </span>
      <span className="tabular flex items-baseline gap-1 text-[15px] font-bold leading-none">
        <span className="text-[var(--text)]">{team}</span>
        <span className="text-[var(--text-subtle)]">:</span>
        <span className="text-[var(--text-muted)]">{opponent}</span>
      </span>
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
      className="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-semibold"
      style={{
        background: "var(--accent-soft)",
        color: "var(--accent-ink)",
      }}
    >
      <Loader2 className="h-3 w-3 animate-spin" />
      {pct != null ? `Downloading ${pct}%` : "Downloading…"}
    </span>
  )
}

function ParsingPill({ percent }: { percent?: number }) {
  const pct =
    typeof percent === "number" ? Math.max(0, Math.min(100, percent)) : null
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-semibold"
      style={{
        background: "var(--accent-soft)",
        color: "var(--accent-ink)",
      }}
    >
      <Loader2 className="h-3 w-3 animate-spin" />
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
      label: "Imported",
      color: "var(--win)",
      bg: "var(--win-soft)",
      dot: "var(--win)",
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
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-semibold",
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
