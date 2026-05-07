import { Check, Loader2, Clock } from "lucide-react"
import { Card } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"
import { useDemoStore } from "@/stores/demo"

interface QueueRow {
  demoId: number
  fileName: string
  percent: number
  stage: "importing" | "parsing" | "complete" | "error" | "pending"
  error?: string
}

function StateIcon({ stage }: { stage: QueueRow["stage"] }) {
  if (stage === "complete")
    return <Check className="h-4 w-4" style={{ color: "var(--win)" }} />
  if (stage === "importing" || stage === "parsing")
    return (
      <Loader2
        className="h-4 w-4 animate-spin"
        style={{ color: "var(--warn)" }}
      />
    )
  if (stage === "error")
    return (
      <span
        className="inline-block h-2 w-2 rounded-full"
        style={{ background: "var(--loss)" }}
      />
    )
  return <Clock className="h-4 w-4" style={{ color: "var(--text-faint)" }} />
}

function QueueRowView({ row }: { row: QueueRow }) {
  return (
    <div className="grid grid-cols-[16px_1fr_80px_26px] items-start gap-3 px-4 py-2">
      <StateIcon stage={row.stage} />
      <div className="min-w-0">
        <div className="truncate font-mono text-[12px] text-[var(--text)]">
          {row.fileName}
        </div>
        {row.stage === "error" && row.error && (
          <div
            className="mt-0.5 break-words font-mono text-[11px]"
            style={{ color: "var(--loss)" }}
            data-testid="import-queue-error"
          >
            {row.error}
          </div>
        )}
      </div>
      <Progress value={row.percent} className="h-1.5 w-[80px]" />
      <div className="tabular text-[11px] text-[var(--text-muted)] text-right">
        {row.percent}%
      </div>
    </div>
  )
}

export function ImportQueue() {
  const importProgress = useDemoStore((s) => s.importProgress)

  if (!importProgress) return null

  const rows: QueueRow[] = [
    {
      demoId: importProgress.demoId,
      fileName: importProgress.fileName,
      percent: importProgress.percent,
      stage: importProgress.stage,
      error: importProgress.error,
    },
  ]

  const activeCount = rows.filter(
    (r) => r.stage === "importing" || r.stage === "parsing",
  ).length

  return (
    <Card className="overflow-hidden border border-[var(--border)] bg-[var(--bg-elevated)] p-0">
      <div className="flex items-center justify-between border-b border-[var(--divider)] px-4 py-2.5">
        <div className="flex items-center gap-2">
          <span className="text-[13.5px] font-bold text-[var(--text)]">
            Import queue
          </span>
          <span
            className="rounded-full px-2 py-0.5 text-[10.5px] font-semibold"
            style={{
              background: "var(--bg-sunken)",
              color: "var(--text-muted)",
            }}
          >
            {activeCount} active
          </span>
        </div>
      </div>
      <div className="divide-y divide-[var(--divider)]">
        {rows.map((row) => (
          <QueueRowView key={row.demoId} row={row} />
        ))}
      </div>
    </Card>
  )
}
