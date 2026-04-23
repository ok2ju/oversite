import { Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"
import type { DemoStatus } from "@/types/demo"

interface StatusConfig {
  label: string
  color: string
  dot: string
  bg: string
  spin?: boolean
}

const CONFIG: Record<"ready" | "parsing" | "error" | "pending", StatusConfig> =
  {
    ready: {
      label: "Ready",
      color: "var(--win)",
      dot: "var(--win)",
      bg: "var(--win-soft)",
    },
    parsing: {
      label: "Parsing",
      color: "var(--warn)",
      dot: "var(--warn)",
      bg: "var(--warn-soft)",
      spin: true,
    },
    error: {
      label: "Error",
      color: "var(--loss)",
      dot: "var(--loss)",
      bg: "var(--loss-soft)",
    },
    pending: {
      label: "Pending",
      color: "var(--text-muted)",
      dot: "var(--text-faint)",
      bg: "var(--bg-sunken)",
    },
  }

export function statusKey(
  status: DemoStatus,
): "ready" | "parsing" | "error" | "pending" {
  if (status === "ready") return "ready"
  if (status === "parsing") return "parsing"
  if (status === "failed") return "error"
  return "pending"
}

interface StatusPillProps {
  status: DemoStatus
  className?: string
}

export function StatusPill({ status, className }: StatusPillProps) {
  const cfg = CONFIG[statusKey(status)]
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-medium",
        className,
      )}
      style={{ background: cfg.bg, color: cfg.color }}
      data-testid={`status-pill-${statusKey(status)}`}
    >
      {cfg.spin ? (
        <Loader2 className="h-3 w-3 animate-spin" />
      ) : (
        <span
          className="h-1.5 w-1.5 rounded-full"
          style={{ background: cfg.dot }}
        />
      )}
      {cfg.label}
    </span>
  )
}
