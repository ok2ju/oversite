import { useEffect } from "react"
import { EventsOn, EventsOff } from "@wailsjs/runtime/runtime"
import { useDemoStore, type ImportProgress } from "@/stores/demo"

/**
 * Subscribes to `demo:parse:progress` Wails events and updates the demo store.
 * The backend emits this event during demo parsing with payload:
 *   { demoId: number, fileName: string, percent: number, stage: string }
 */
export function useParseProgress() {
  const updateImportProgress = useDemoStore((s) => s.updateImportProgress)

  useEffect(() => {
    const cancel = EventsOn("demo:parse:progress", (data: ImportProgress) => {
      updateImportProgress(data)
      if (data.stage === "complete") {
        setTimeout(() => updateImportProgress(null), 2000)
      } else if (data.stage === "error") {
        // Errors stay on screen long enough to read and act on. The user can
        // also drop a new demo to replace the row.
        setTimeout(() => updateImportProgress(null), 30000)
      }
    })

    return () => {
      EventsOff("demo:parse:progress")
      if (typeof cancel === "function") cancel()
    }
  }, [updateImportProgress])
}
