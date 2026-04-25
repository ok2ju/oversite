import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Play, RefreshCw } from "lucide-react"
import { useDemos } from "@/hooks/use-demos"
import { useFaceitSync } from "@/hooks/use-faceit-sync"
import { useFaceitStore } from "@/stores/faceit"
import { cn } from "@/lib/utils"

function formatRelative(ts: number | null): string {
  if (!ts) return "Not synced yet"
  const seconds = Math.max(0, Math.floor((Date.now() - ts) / 1000))
  if (seconds < 5) return "Just now"
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

export function DashboardHeaderActions() {
  const navigate = useNavigate()
  const sync = useFaceitSync()
  const lastSyncedAt = useFaceitStore((s) => s.lastSyncedAt)
  const { data: demosData } = useDemos(1, 1)
  const [, tick] = useState(0)

  useEffect(() => {
    const id = setInterval(() => tick((n) => n + 1), 30_000)
    return () => clearInterval(id)
  }, [])

  const latestDemo = demosData?.data?.find((d) => d.status === "ready")
  const canOpenLatest = Boolean(latestDemo)
  const label = sync.isPending
    ? "Syncing…"
    : `Last synced ${formatRelative(lastSyncedAt)}`

  return (
    <>
      <button
        type="button"
        className="btn-sm ghost"
        onClick={() => sync.mutate()}
        disabled={sync.isPending}
        aria-label="Sync Faceit data"
      >
        <RefreshCw
          className={cn("h-3 w-3", sync.isPending && "animate-spin")}
        />
        {label}
      </button>
      <button
        type="button"
        className="btn-sm primary"
        onClick={() => latestDemo && navigate(`/matches/${latestDemo.id}`)}
        disabled={!canOpenLatest}
      >
        <Play className="h-3 w-3" />
        Open latest demo
      </button>
    </>
  )
}
