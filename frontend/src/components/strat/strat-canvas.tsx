"use client"

import { useEffect, useRef } from "react"
import { createViewerApp, type ViewerApp } from "@/lib/pixi/app"
import { Camera } from "@/lib/pixi/camera"
import { MapLayer } from "@/lib/pixi/layers/map-layer"
import { StratRenderer } from "@/lib/strat/renderer"
import { useStratStore } from "@/stores/strat"

export function StratCanvas() {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    let destroyed = false
    let viewerApp: ViewerApp | null = null
    let camera: Camera | null = null
    let mapLayer: MapLayer | null = null
    let stratRenderer: StratRenderer | null = null
    let mapUnsub: (() => void) | null = null
    let resizeObserver: ResizeObserver | null = null

    createViewerApp({ container }).then((app) => {
      if (destroyed) {
        app.destroy()
        return
      }

      viewerApp = app

      // Create camera and add its container to stage
      camera = new Camera(app.canvas, {
        onViewportChange: () => {},
      })
      app.stage.addChild(camera.container)

      // Set initial screen size
      const { width, height } = container.getBoundingClientRect()
      camera.setScreenSize(width, height)

      // Observe container resize
      resizeObserver = new ResizeObserver((entries) => {
        const entry = entries[0]
        if (!entry || !camera) return
        const { width: w, height: h } = entry.contentRect
        camera.setScreenSize(w, h)
      })
      resizeObserver.observe(container)

      // Layers: map (bottom) -> drawings (top)
      const mapContainer = app.addLayer("map", camera.container)
      mapLayer = new MapLayer(mapContainer)

      const drawingsContainer = app.addLayer("drawings", camera.container)
      stratRenderer = new StratRenderer(drawingsContainer)

      // Subscribe to mapName changes
      mapUnsub = useStratStore.subscribe(
        (s) => s.mapName,
        (mapName) => {
          if (destroyed) return
          if (mapName) {
            mapLayer
              ?.setMap(mapName)
              .then(() => {
                if (mapLayer?.calibration) {
                  camera?.setMapSize(
                    mapLayer.calibration.width,
                    mapLayer.calibration.height,
                  )
                }
              })
              .catch(console.error)
          } else {
            mapLayer?.clear()
          }
        },
        { fireImmediately: true },
      )

      // TODO: Subscribe to boardId changes and load strat data via Wails bindings
      // when the desktop strat board is implemented (P5 tasks).
      // Previously this used Yjs collaborative docs over WebSocket.
    })

    return () => {
      destroyed = true
      resizeObserver?.disconnect()
      mapUnsub?.()
      stratRenderer?.destroy()
      mapLayer?.destroy()
      camera?.destroy()
      viewerApp?.destroy()
    }
  }, [])

  return (
    <div
      ref={containerRef}
      className="relative h-full w-full overflow-hidden"
      data-testid="strat-canvas-container"
    />
  )
}
