import { useEffect } from "react"
import { useViewerStore } from "@/stores/viewer"

const SPEED_OPTIONS = [0.25, 0.5, 1, 2, 4] as const
const SEEK_SECONDS = 5

interface UseViewerKeyboardOptions {
  onToggleScoreboard: () => void
  // Optional: jump to the prev/next event tick on the active round timeline.
  // direction = -1 for `,` (previous), +1 for `.` (next). Return null to
  // ignore the keystroke (no events, no model, etc.).
  onNavigateEvent?: (direction: -1 | 1) => number | null
}

export function useViewerKeyboard({
  onToggleScoreboard,
  onNavigateEvent,
}: UseViewerKeyboardOptions) {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === "INPUT" || tag === "TEXTAREA") return

      const state = useViewerStore.getState()
      const seekTicks = state.tickRate * SEEK_SECONDS

      switch (e.key) {
        case " ": {
          e.preventDefault()
          state.togglePlay()
          break
        }
        case "ArrowLeft": {
          e.preventDefault()
          const newTick = Math.max(0, state.currentTick - seekTicks)
          state.setTick(newTick)
          break
        }
        case "ArrowRight": {
          e.preventDefault()
          const newTick = Math.min(
            state.totalTicks,
            state.currentTick + seekTicks,
          )
          state.setTick(newTick)
          break
        }
        case "ArrowUp": {
          e.preventDefault()
          const currentIdx = SPEED_OPTIONS.indexOf(
            state.speed as (typeof SPEED_OPTIONS)[number],
          )
          if (currentIdx < SPEED_OPTIONS.length - 1) {
            state.setSpeed(SPEED_OPTIONS[currentIdx + 1])
          }
          break
        }
        case "ArrowDown": {
          e.preventDefault()
          const currentIdx = SPEED_OPTIONS.indexOf(
            state.speed as (typeof SPEED_OPTIONS)[number],
          )
          if (currentIdx > 0) {
            state.setSpeed(SPEED_OPTIONS[currentIdx - 1])
          }
          break
        }
        case "Tab": {
          e.preventDefault()
          onToggleScoreboard()
          break
        }
        case "Escape": {
          e.preventDefault()
          state.setSelectedPlayer(null)
          break
        }
        case "r":
        case "R": {
          e.preventDefault()
          state.resetViewport()
          break
        }
        case ",": {
          if (!onNavigateEvent) break
          e.preventDefault()
          const tick = onNavigateEvent(-1)
          if (tick !== null) state.setTick(tick)
          break
        }
        case ".": {
          if (!onNavigateEvent) break
          e.preventDefault()
          const tick = onNavigateEvent(1)
          if (tick !== null) state.setTick(tick)
          break
        }
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [onToggleScoreboard, onNavigateEvent])
}
