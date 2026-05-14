import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Play, Trash2 } from "lucide-react"
import { cn } from "@/lib/utils"
import { useDemoStore } from "@/stores/demo"
import type { DemoSummary } from "@/types/demo"
import { MapTile, resolveMap } from "@/components/demos/map-tile"
import { StatusPill, statusKey } from "@/components/demos/status-pill"
import type { DemosFilter } from "@/components/demos/demos-toolbar"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { buttonVariants } from "@/components/ui/button"

export function formatBytes(bytes: number): string {
  if (bytes >= 1_000_000_000) return `${(bytes / 1_000_000_000).toFixed(1)} GB`
  if (bytes >= 1_000_000) return `${(bytes / 1_000_000).toFixed(0)} MB`
  return `${(bytes / 1_000).toFixed(0)} KB`
}

export function formatDuration(secs: number): string {
  if (!secs) return "—"
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return `${m}:${s.toString().padStart(2, "0")}`
}

export function formatDate(iso: string): string {
  if (!iso) return "—"
  return new Date(iso).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  })
}

export function filterDemos(
  demos: DemoSummary[],
  search: string,
  filter: DemosFilter,
): DemoSummary[] {
  const q = search.trim().toLowerCase()
  return demos.filter((demo) => {
    if (filter === "ready" && demo.status !== "ready") return false
    if (filter === "parsing" && demo.status !== "parsing") return false
    if (filter === "error" && demo.status !== "failed") return false
    if (!q) return true
    const haystack = `${demo.map_name} ${demo.file_name}`.toLowerCase()
    return haystack.includes(q)
  })
}

interface LibraryTableProps {
  demos: DemoSummary[]
  search: string
  filter: DemosFilter
  onDelete: (id: number) => void
}

export function LibraryTable({
  demos,
  search,
  filter,
  onDelete,
}: LibraryTableProps) {
  const navigate = useNavigate()
  const importProgress = useDemoStore((s) => s.importProgress)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [waitingDemoId, setWaitingDemoId] = useState<number | null>(null)
  const [pendingDelete, setPendingDelete] = useState<DemoSummary | null>(null)
  const rows = useMemo(
    () => filterDemos(demos, search, filter),
    [demos, search, filter],
  )

  function confirmDelete() {
    if (pendingDelete) onDelete(pendingDelete.id)
    setPendingDelete(null)
  }

  useEffect(() => {
    if (waitingDemoId == null) return
    if (
      importProgress?.demoId === waitingDemoId &&
      importProgress.stage === "complete"
    ) {
      const id = waitingDemoId
      setWaitingDemoId(null)
      navigate(`/demos/${id}/overview`)
    }
  }, [waitingDemoId, importProgress, navigate])

  function toggle(id: number) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function toggleAll() {
    setSelected((prev) =>
      prev.size === rows.length ? new Set() : new Set(rows.map((r) => r.id)),
    )
  }

  function handleRowClick(demo: DemoSummary) {
    if (demo.status === "parsing") {
      setWaitingDemoId(demo.id)
      return
    }
    navigate(`/demos/${demo.id}/overview`)
  }

  if (rows.length === 0) {
    return (
      <div
        className="rounded-lg border border-dashed py-12 text-center text-[12.5px] text-[var(--text-muted)]"
        style={{ borderColor: "var(--border)" }}
      >
        No demos match the current filters.
      </div>
    )
  }

  const pendingFileName = pendingDelete?.file_name ?? "this demo"

  return (
    <div className="demos-table">
      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove this demo?</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingFileName} will be removed from your library. The demo file
              on disk will not be deleted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className={buttonVariants({ variant: "destructive" })}
              onClick={confirmDelete}
            >
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <table>
        <thead>
          <tr>
            <th style={{ width: 40 }}>
              <input
                type="checkbox"
                aria-label="Select all"
                checked={selected.size > 0 && selected.size === rows.length}
                onChange={toggleAll}
              />
            </th>
            <th>Map</th>
            <th style={{ width: 90 }}>Score</th>
            <th style={{ width: 140 }}>Date</th>
            <th style={{ width: 80 }}>Duration</th>
            <th style={{ width: 70 }}>Size</th>
            <th style={{ width: 110 }}>Status</th>
            <th style={{ width: 120 }} aria-label="Actions"></th>
          </tr>
        </thead>
        <tbody>
          {rows.map((demo) => {
            const isSelected = selected.has(demo.id)
            const meta = resolveMap(demo.map_name)
            const fileName = demo.file_name
            const isParsing = demo.status === "parsing"
            const isWaiting = waitingDemoId === demo.id
            return (
              <tr
                key={demo.id}
                data-testid={`demo-row-${demo.id}`}
                className={cn(
                  isSelected && "selected",
                  isWaiting && "cursor-wait",
                )}
                onClick={() => handleRowClick(demo)}
              >
                <td onClick={(e) => e.stopPropagation()}>
                  <input
                    type="checkbox"
                    aria-label={`Select demo ${demo.id}`}
                    checked={isSelected}
                    onChange={() => toggle(demo.id)}
                  />
                </td>
                <td>
                  <div className="mcell">
                    <MapTile mapName={demo.map_name} size={28} />
                    <div className="min-w-0">
                      <div className="m-name truncate">{meta.name}</div>
                      <div className="m-id truncate">{fileName}</div>
                    </div>
                  </div>
                </td>
                <td className="tabular text-[var(--text-muted)]">—</td>
                <td className="tabular text-[var(--text-muted)]">
                  {formatDate(demo.match_date || demo.created_at)}
                </td>
                <td className="tabular">
                  {formatDuration(demo.duration_secs)}
                </td>
                <td className="tabular text-[var(--text-muted)]">
                  {formatBytes(demo.file_size)}
                </td>
                <td>
                  {isParsing && isWaiting ? (
                    <span
                      className="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-medium"
                      style={{
                        background: "var(--warn-soft)",
                        color: "var(--warn)",
                      }}
                      data-testid={`demo-row-${demo.id}-waiting`}
                    >
                      Parsing…
                    </span>
                  ) : (
                    <StatusPill status={demo.status} />
                  )}
                </td>
                <td className="act-cell" onClick={(e) => e.stopPropagation()}>
                  <div className="row-actions">
                    <button
                      type="button"
                      className="icon-btn"
                      aria-label="Play"
                      disabled={statusKey(demo.status) !== "ready"}
                      onClick={() => navigate(`/demos/${demo.id}`)}
                    >
                      <Play className="h-3 w-3" />
                    </button>
                    <button
                      type="button"
                      className="icon-btn"
                      aria-label="Delete"
                      onClick={() => setPendingDelete(demo)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
