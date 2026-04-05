"use client"

import { useEffect, useRef } from "react"
import { createViewerApp, type ViewerApp } from "@/lib/pixi/app"
import { MapLayer } from "@/lib/pixi/layers/map-layer"
import { useViewerStore } from "@/stores/viewer"

function setupViewerSubscriptions(app: ViewerApp): () => void {
  const unsubs: (() => void)[] = []

  unsubs.push(
    useViewerStore.subscribe(
      (s) => s.isPlaying,
      (isPlaying) => {
        if (isPlaying) {
          app.ticker.start()
        } else {
          app.ticker.stop()
        }
      },
      { fireImmediately: true }
    )
  )

  unsubs.push(
    useViewerStore.subscribe(
      (s) => s.speed,
      (speed) => {
        app.ticker.speed = speed
      },
      { fireImmediately: true }
    )
  )

  return () => unsubs.forEach((fn) => fn())
}

export function ViewerCanvas() {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    let destroyed = false
    let viewerApp: ViewerApp | null = null
    let mapLayer: MapLayer | null = null
    let unsubscribe: (() => void) | null = null
    let mapUnsub: (() => void) | null = null

    createViewerApp({ container }).then((app) => {
      if (destroyed) {
        app.destroy()
        return
      }

      viewerApp = app

      const mapContainer = app.addLayer("map")
      mapLayer = new MapLayer(mapContainer)

      unsubscribe = setupViewerSubscriptions(app)
      mapUnsub = useViewerStore.subscribe(
        (s) => s.mapName,
        (mapName) => {
          if (mapName) {
            mapLayer?.setMap(mapName).catch(console.error)
          } else {
            mapLayer?.clear()
          }
        },
        { fireImmediately: true }
      )
    })

    return () => {
      destroyed = true
      mapUnsub?.()
      unsubscribe?.()
      mapLayer?.destroy()
      viewerApp?.destroy()
    }
  }, [])

  return (
    <div
      ref={containerRef}
      className="relative h-full w-full overflow-hidden"
      data-testid="viewer-canvas-container"
    />
  )
}
