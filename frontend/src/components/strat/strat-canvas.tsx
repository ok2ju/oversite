"use client"

import { useEffect, useRef } from "react"
import { createViewerApp, type ViewerApp } from "@/lib/pixi/app"
import { Camera } from "@/lib/pixi/camera"
import { MapLayer } from "@/lib/pixi/layers/map-layer"
import { StratRenderer } from "@/lib/strat/renderer"
import { createStratDoc } from "@/lib/yjs/doc"
import { createStratProvider, type StratProvider } from "@/lib/yjs/provider"
import { useStratStore } from "@/stores/strat"
import type * as Y from "yjs"

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
    let boardUnsub: (() => void) | null = null
    let resizeObserver: ResizeObserver | null = null
    let currentDoc: Y.Doc | null = null
    let currentProvider: StratProvider | null = null

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

      // Subscribe to boardId changes -> create/swap Yjs doc + provider
      boardUnsub = useStratStore.subscribe(
        (s) => s.boardId,
        (boardId) => {
          if (destroyed) return
          // Clean up previous
          stratRenderer?.detach()
          currentProvider?.destroy()
          currentDoc?.destroy()
          currentProvider = null
          currentDoc = null

          if (boardId) {
            currentDoc = createStratDoc()
            currentProvider = createStratProvider({
              stratId: boardId,
              doc: currentDoc,
            })
            stratRenderer?.attach(currentDoc)
          }
        },
        { fireImmediately: true },
      )
    })

    return () => {
      destroyed = true
      resizeObserver?.disconnect()
      boardUnsub?.()
      mapUnsub?.()
      stratRenderer?.destroy()
      currentProvider?.destroy()
      currentProvider = null
      currentDoc?.destroy()
      currentDoc = null
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
