import { useEffect, useState } from "react"
import { EventsOn, EventsOff } from "@wailsjs/runtime/runtime"

export interface SyncProgress {
  current: number
  total: number
}

/**
 * Subscribes to `faceit:sync:progress` Wails events emitted during
 * SyncFaceitMatches. Returns the latest progress or null when idle.
 */
export function useFaceitSyncProgress() {
  const [progress, setProgress] = useState<SyncProgress | null>(null)

  useEffect(() => {
    const cancel = EventsOn("faceit:sync:progress", (data: SyncProgress) => {
      setProgress(data)
    })

    return () => {
      EventsOff("faceit:sync:progress")
      if (typeof cancel === "function") cancel()
    }
  }, [])

  const reset = () => setProgress(null)

  return { progress, reset }
}
