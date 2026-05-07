import { useEffect, useState } from "react"
import { FolderOpen, Copy, Check } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { LogsDir, OpenLogsFolder } from "@wailsjs/go/main/App"
import { ClipboardSetText } from "@wailsjs/runtime/runtime"

export default function SettingsPage() {
  const [logsDir, setLogsDir] = useState<string>("")
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    LogsDir()
      .then(setLogsDir)
      .catch((e: unknown) => setError(String(e)))
  }, [])

  async function handleOpen() {
    setError(null)
    try {
      await OpenLogsFolder()
    } catch (e) {
      setError(String(e))
    }
  }

  async function handleCopy() {
    if (!logsDir) return
    try {
      await ClipboardSetText(logsDir)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch (e) {
      setError(String(e))
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Settings</h1>
        <p className="mt-2 text-muted-foreground">
          Manage your account and preferences
        </p>
      </div>

      <Card className="space-y-3 p-4">
        <div>
          <h2 className="text-sm font-semibold">Diagnostics</h2>
          <p className="text-xs text-muted-foreground">
            Persistent error log written to{" "}
            <code className="font-mono">errors.txt</code>. Share this file when
            reporting bugs.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <code
            className="block flex-1 truncate rounded border px-2 py-1 font-mono text-[12px]"
            title={logsDir}
            data-testid="settings-logs-path"
          >
            {logsDir || "—"}
          </code>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCopy}
            disabled={!logsDir}
          >
            {copied ? (
              <Check className="mr-1 h-3 w-3" />
            ) : (
              <Copy className="mr-1 h-3 w-3" />
            )}
            {copied ? "Copied" : "Copy path"}
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={handleOpen}
            disabled={!logsDir}
          >
            <FolderOpen className="mr-1 h-3 w-3" />
            Open logs folder
          </Button>
        </div>

        {error && (
          <p className="text-[11px]" style={{ color: "var(--loss)" }}>
            {error}
          </p>
        )}
      </Card>
    </div>
  )
}
