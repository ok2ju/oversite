"use client"

import { useEffect, useRef } from "react"
import { createViewerApp, type ViewerApp } from "@/lib/pixi/app"
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
    let unsubscribe: (() => void) | null = null

    createViewerApp({ container }).then((app) => {
      if (destroyed) {
        app.destroy()
        return
      }

      viewerApp = app
      unsubscribe = setupViewerSubscriptions(app)
    })

    return () => {
      destroyed = true
      unsubscribe?.()
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
