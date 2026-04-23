import { Folder } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useImportFolder } from "@/hooks/use-demos"

interface WatchBannerProps {
  folderPath?: string
  queuedCount?: number
}

export function WatchBanner({
  folderPath = "~/Documents/CS2/replays",
  queuedCount = 0,
}: WatchBannerProps) {
  const { importFolder, isImporting } = useImportFolder()

  return (
    <div
      className="flex items-center gap-4 rounded-lg border px-4 py-3"
      style={{
        background: "var(--accent-soft)",
        borderColor: "#c7d7f7",
      }}
    >
      <div
        className="grid h-10 w-10 place-items-center rounded-md"
        style={{ background: "#fff" }}
      >
        <Folder className="h-5 w-5" style={{ color: "var(--accent)" }} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-[13.5px] font-bold text-[var(--text)]">
          Watching folder · {queuedCount} demos queued
        </div>
        <div className="mt-0.5 flex items-center gap-2 text-[12px] text-[var(--text-muted)]">
          <span
            className="font-mono rounded px-1.5 py-0.5 text-[11.5px]"
            style={{
              background: "#fff",
              border: "1px solid #c7d7f7",
              color: "var(--accent-ink)",
            }}
          >
            {folderPath}
          </span>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="sm" disabled>
          Change
        </Button>
        <Button size="sm" onClick={() => importFolder()} disabled={isImporting}>
          {isImporting ? "Scanning…" : "Re-scan"}
        </Button>
      </div>
    </div>
  )
}
