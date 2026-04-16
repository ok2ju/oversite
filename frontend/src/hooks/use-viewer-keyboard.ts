import { useEffect } from "react"
import { useViewerStore } from "@/stores/viewer"

const SPEED_OPTIONS = [0.25, 0.5, 1, 2, 4] as const
const SEEK_TICKS = 320 // 5 seconds at 64 tick rate

interface UseViewerKeyboardOptions {
  onToggleScoreboard: () => void
}

export function useViewerKeyboard({
  onToggleScoreboard,
}: UseViewerKeyboardOptions) {
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === "INPUT" || tag === "TEXTAREA") return

      const state = useViewerStore.getState()

      switch (e.key) {
        case " ": {
          e.preventDefault()
          state.togglePlay()
          break
        }
        case "ArrowLeft": {
          e.preventDefault()
          const newTick = Math.max(0, state.currentTick - SEEK_TICKS)
          state.setTick(newTick)
          break
        }
        case "ArrowRight": {
          e.preventDefault()
          const newTick = Math.min(
            state.totalTicks,
            state.currentTick + SEEK_TICKS,
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
      }
    }

    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [onToggleScoreboard])
}
