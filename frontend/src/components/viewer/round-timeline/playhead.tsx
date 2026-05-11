import { useEffect, useRef } from "react"
import { useViewerStore } from "@/stores/viewer"

interface PlayheadProps {
  roundStartTick: number
  roundEndTick: number
}

// Imperative playhead: subscribes to the viewer store outside React's render
// cycle and writes transform directly to a ref. Lanes never re-render on tick
// advances; only this div's `style.transform` is touched.
export function Playhead({ roundStartTick, roundEndTick }: PlayheadProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const apply = (tick: number) => {
      const node = ref.current
      if (!node) return
      const span = Math.max(1, roundEndTick - roundStartTick)
      const clamped = Math.max(roundStartTick, Math.min(tick, roundEndTick))
      const pct = ((clamped - roundStartTick) / span) * 100
      node.style.left = `${pct}%`
      node.style.opacity =
        tick < roundStartTick || tick > roundEndTick ? "0.35" : "1"
    }

    apply(useViewerStore.getState().currentTick)
    const unsubscribe = useViewerStore.subscribe(
      (s) => s.currentTick,
      (tick) => apply(tick),
    )
    return unsubscribe
  }, [roundStartTick, roundEndTick])

  return (
    <div
      ref={ref}
      data-testid="round-timeline-playhead"
      aria-hidden="true"
      className="pointer-events-none absolute inset-y-0 left-0 z-10 w-px -translate-x-1/2 bg-orange-400 shadow-[0_0_6px_0_rgba(255,122,26,0.7)]"
    />
  )
}
