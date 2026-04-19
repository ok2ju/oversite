import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Play, Link2, Trash2 } from "lucide-react"
import { cn } from "@/lib/utils"
import { useDemoStore } from "@/stores/demo"
import type { Demo } from "@/types/demo"
import { MapTile, resolveMap } from "@/components/dashboard/map-tile"
import { StatusPill, statusKey } from "@/components/demos/status-pill"
import type { DemosFilter } from "@/components/demos/demos-toolbar"

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
  demos: Demo[],
  search: string,
  filter: DemosFilter,
): Demo[] {
  const q = search.trim().toLowerCase()
  return demos.filter((demo) => {
    if (filter === "parsing" && demo.status !== "parsing") return false
    // 'wins' / 'losses' are spec labels without backing match-result data on
    // the Demo model yet — treat them as no-ops alongside 'all'.
    if (!q) return true
    const haystack = `${demo.map_name} ${demo.file_path}`.toLowerCase()
    return haystack.includes(q)
  })
}

interface LibraryTableProps {
  demos: Demo[]
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
  const rows = useMemo(
    () => filterDemos(demos, search, filter),
    [demos, search, filter],
  )

  useEffect(() => {
    if (waitingDemoId == null) return
    if (
      importProgress?.demoId === waitingDemoId &&
      importProgress.stage === "complete"
    ) {
      const id = waitingDemoId
      setWaitingDemoId(null)
      navigate(`/matches/${id}`)
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

  function handleRowClick(demo: Demo) {
    if (demo.status === "parsing") {
      setWaitingDemoId(demo.id)
      return
    }
    navigate(`/matches/${demo.id}`)
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

  return (
    <div
      className="overflow-hidden rounded-lg border"
      style={{ borderColor: "var(--border)", background: "var(--bg-elevated)" }}
    >
      <div
        className="grid items-center px-3.5 py-2 text-[10.5px] font-semibold uppercase tracking-wider text-[var(--text-subtle)]"
        style={{
          gridTemplateColumns:
            "40px minmax(0,1fr) 90px 140px 80px 70px 110px 120px",
          background: "var(--bg-sunken)",
        }}
      >
        <div>
          <input
            type="checkbox"
            aria-label="Select all"
            checked={selected.size > 0 && selected.size === rows.length}
            onChange={toggleAll}
          />
        </div>
        <div>Map</div>
        <div>Score</div>
        <div>Date</div>
        <div>Duration</div>
        <div>Size</div>
        <div>Status</div>
        <div className="text-right">Actions</div>
      </div>

      <div>
        {rows.map((demo) => {
          const isSelected = selected.has(demo.id)
          const meta = resolveMap(demo.map_name)
          const fileName = demo.file_path.split("/").pop() ?? ""
          const isParsing = demo.status === "parsing"
          const isWaiting = waitingDemoId === demo.id
          return (
            <div
              key={demo.id}
              data-testid={`demo-row-${demo.id}`}
              className={cn(
                "group grid cursor-pointer items-center border-t px-3.5 py-2.5 text-[12.5px] transition-colors",
                isWaiting && "cursor-wait",
              )}
              style={{
                gridTemplateColumns:
                  "40px minmax(0,1fr) 90px 140px 80px 70px 110px 120px",
                borderColor: "var(--divider)",
                background: isSelected
                  ? "var(--bg-selected)"
                  : "var(--bg-elevated)",
              }}
              onMouseEnter={(e) =>
                (e.currentTarget.style.background = isSelected
                  ? "var(--bg-selected)"
                  : "var(--bg-hover)")
              }
              onMouseLeave={(e) =>
                (e.currentTarget.style.background = isSelected
                  ? "var(--bg-selected)"
                  : "var(--bg-elevated)")
              }
              onClick={() => handleRowClick(demo)}
            >
              <div onClick={(e) => e.stopPropagation()}>
                <input
                  type="checkbox"
                  aria-label={`Select demo ${demo.id}`}
                  checked={isSelected}
                  onChange={() => toggle(demo.id)}
                />
              </div>
              <div className="flex min-w-0 items-center gap-3">
                <MapTile mapName={demo.map_name} size={28} />
                <div className="min-w-0">
                  <div className="truncate font-semibold text-[var(--text)]">
                    {meta.name}
                  </div>
                  <div className="truncate font-mono text-[11px] text-[var(--text-subtle)]">
                    {fileName}
                  </div>
                </div>
              </div>
              <div className="tabular text-[var(--text-muted)]">—</div>
              <div className="text-[var(--text-muted)]">
                {formatDate(demo.match_date || demo.created_at)}
              </div>
              <div className="tabular text-[var(--text-muted)]">
                {formatDuration(demo.duration_secs)}
              </div>
              <div className="tabular text-[var(--text-muted)]">
                {formatBytes(demo.file_size)}
              </div>
              <div>
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
              </div>
              <div
                className={cn(
                  "flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100",
                  isSelected && "opacity-100",
                )}
                onClick={(e) => e.stopPropagation()}
              >
                <button
                  type="button"
                  className="grid h-7 w-7 place-items-center rounded-md text-[var(--text-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--text)]"
                  aria-label="Play"
                  disabled={statusKey(demo.status) !== "ready"}
                  onClick={() => navigate(`/demos/${demo.id}`)}
                >
                  <Play className="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  className="grid h-7 w-7 place-items-center rounded-md text-[var(--text-muted)] hover:bg-[var(--bg-hover)] hover:text-[var(--text)]"
                  aria-label="Open match"
                  onClick={() => navigate(`/matches/${demo.id}`)}
                >
                  <Link2 className="h-3.5 w-3.5" />
                </button>
                <button
                  type="button"
                  className="grid h-7 w-7 place-items-center rounded-md text-[var(--text-muted)] hover:bg-[var(--loss-soft)] hover:text-[var(--loss)]"
                  aria-label="Delete"
                  onClick={() => onDelete(demo.id)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
