import { useEffect } from "react"
import { FilterPanel } from "@/components/heatmap/filter-panel"
import { HeatmapCanvas } from "@/components/heatmap/heatmap-canvas"
import { useHeatmapStore } from "@/stores/heatmap"

export default function HeatmapsPage() {
  const reset = useHeatmapStore((s) => s.reset)

  useEffect(() => {
    return () => {
      reset()
    }
  }, [reset])

  return (
    <div
      className="relative -m-6 flex h-[calc(100%+3rem)] overflow-hidden"
      data-testid="heatmaps-page"
    >
      <FilterPanel />
      <div className="flex-1">
        <HeatmapCanvas />
      </div>
    </div>
  )
}
