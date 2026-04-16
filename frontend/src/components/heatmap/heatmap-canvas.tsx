import { useEffect, useRef } from "react"
import { createViewerApp, type ViewerApp } from "@/lib/pixi/app"
import { MapLayer } from "@/lib/pixi/layers/map-layer"
import { HeatmapLayer } from "@/lib/pixi/layers/heatmap-layer"
import { useHeatmapStore } from "@/stores/heatmap"
import { useHeatmapData } from "@/hooks/use-heatmap"

export function HeatmapCanvas() {
  const containerRef = useRef<HTMLDivElement>(null)

  const selectedMap = useHeatmapStore((s) => s.selectedMap)
  const selectedDemoIds = useHeatmapStore((s) => s.selectedDemoIds)
  const selectedWeapons = useHeatmapStore((s) => s.selectedWeapons)
  const selectedPlayer = useHeatmapStore((s) => s.selectedPlayer)
  const selectedSide = useHeatmapStore((s) => s.selectedSide)
  const bandwidth = useHeatmapStore((s) => s.bandwidth)
  const opacity = useHeatmapStore((s) => s.opacity)

  const { data: heatmapPoints } = useHeatmapData(
    selectedDemoIds,
    selectedWeapons,
    selectedPlayer,
    selectedSide,
  )

  const mapLayerRef = useRef<MapLayer | null>(null)
  const heatmapLayerRef = useRef<HeatmapLayer | null>(null)

  // Initialize PixiJS app
  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    let destroyed = false
    let viewerApp: ViewerApp | null = null
    let mapLayer: MapLayer | null = null
    let heatmapLayer: HeatmapLayer | null = null

    createViewerApp({ container }).then((app) => {
      if (destroyed) {
        app.destroy()
        return
      }

      viewerApp = app

      const mapContainer = app.addLayer("map", app.stage)
      mapLayer = new MapLayer(mapContainer)
      mapLayerRef.current = mapLayer

      const heatmapContainer = app.addLayer("heatmap", app.stage)
      heatmapLayer = new HeatmapLayer(heatmapContainer)
      heatmapLayerRef.current = heatmapLayer
    })

    return () => {
      destroyed = true
      heatmapLayerRef.current = null
      mapLayerRef.current = null
      heatmapLayer?.destroy()
      mapLayer?.destroy()
      viewerApp?.destroy()
    }
  }, [])

  // Update map when selection changes
  useEffect(() => {
    const mapLayer = mapLayerRef.current
    if (!mapLayer) return

    if (selectedMap) {
      mapLayer.setMap(selectedMap).catch(console.error)
    } else {
      mapLayer.clear()
    }
  }, [selectedMap])

  // Update heatmap overlay when data or display options change
  useEffect(() => {
    const heatmapLayer = heatmapLayerRef.current
    const mapLayer = mapLayerRef.current
    if (!heatmapLayer || !mapLayer) return

    const calibration = mapLayer.calibration
    if (!calibration || !heatmapPoints || heatmapPoints.length === 0) {
      heatmapLayer.clear()
      return
    }

    heatmapLayer.render(heatmapPoints, calibration, { bandwidth, opacity })
  }, [heatmapPoints, bandwidth, opacity])

  return (
    <div
      ref={containerRef}
      className="relative h-full w-full overflow-hidden"
      data-testid="heatmap-canvas-container"
    />
  )
}
