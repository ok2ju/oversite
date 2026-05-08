import { useEffect, useRef } from "react"
import { EventsOn, EventsOff } from "@wailsjs/runtime/runtime"
import { useDemoStore, type ImportProgress } from "@/stores/demo"

/**
 * Subscribes to `demo:parse:progress` Wails events and updates the demo store.
 * The backend emits this event during demo parsing with payload:
 *   { demoId: number, fileName: string, percent: number, stage: string }
 */
export function useParseProgress() {
  const updateImportProgress = useDemoStore((s) => s.updateImportProgress)
  // Hold the pending "clear progress" timer so unmount can cancel it; otherwise
  // a 30s error timeout (or 2s complete timeout) can fire after unmount and
  // call into the store unnecessarily.
  const clearTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    const cancel = EventsOn("demo:parse:progress", (data: ImportProgress) => {
      updateImportProgress(data)
      if (data.stage === "complete") {
        if (clearTimerRef.current) clearTimeout(clearTimerRef.current)
        clearTimerRef.current = setTimeout(() => {
          clearTimerRef.current = null
          updateImportProgress(null)
        }, 2000)
      } else if (data.stage === "error") {
        // Errors stay on screen long enough to read and act on. The user can
        // also drop a new demo to replace the row.
        if (clearTimerRef.current) clearTimeout(clearTimerRef.current)
        clearTimerRef.current = setTimeout(() => {
          clearTimerRef.current = null
          updateImportProgress(null)
        }, 30000)
      }
    })

    return () => {
      EventsOff("demo:parse:progress")
      if (typeof cancel === "function") cancel()
      if (clearTimerRef.current) {
        clearTimeout(clearTimerRef.current)
        clearTimerRef.current = null
      }
    }
  }, [updateImportProgress])
}
