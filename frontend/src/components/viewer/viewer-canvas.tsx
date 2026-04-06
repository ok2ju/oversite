"use client"

import { useEffect, useRef } from "react"
import { createViewerApp, type ViewerApp } from "@/lib/pixi/app"
import { MapLayer } from "@/lib/pixi/layers/map-layer"
import { PlayerLayer } from "@/lib/pixi/layers/player-layer"
import { TickBuffer } from "@/lib/pixi/tick-buffer"
import { fetchRoster } from "@/hooks/use-roster"
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
    let playerLayer: PlayerLayer | null = null
    let tickBuffer: TickBuffer | null = null
    let unsubscribe: (() => void) | null = null
    let mapUnsub: (() => void) | null = null
    let tickUnsub: (() => void) | null = null
    let roundUnsub: (() => void) | null = null
    let demoUnsub: (() => void) | null = null
    let rosterAbortController: AbortController | null = null

    createViewerApp({ container }).then((app) => {
      if (destroyed) {
        app.destroy()
        return
      }

      viewerApp = app

      const mapContainer = app.addLayer("map")
      mapLayer = new MapLayer(mapContainer)

      const playerContainer = app.addLayer("players")
      playerLayer = new PlayerLayer(playerContainer)

      playerLayer.onPlayerClick((steamId) => {
        const { selectedPlayerSteamId, setSelectedPlayer } = useViewerStore.getState()
        setSelectedPlayer(selectedPlayerSteamId === steamId ? null : steamId)
      })

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

      demoUnsub = useViewerStore.subscribe(
        (s) => s.demoId,
        (demoId) => {
          tickBuffer?.dispose()
          tickBuffer = demoId ? new TickBuffer(demoId) : null
        },
        { fireImmediately: true }
      )

      tickUnsub = useViewerStore.subscribe(
        (s) => ({ tick: s.currentTick, selected: s.selectedPlayerSteamId }),
        ({ tick, selected }) => {
          if (!playerLayer || !mapLayer?.calibration || !tickBuffer) return
          const data = tickBuffer.getTickData(tick)
          if (data === null) return
          playerLayer.update(data, mapLayer.calibration, selected)
        },
        { fireImmediately: false }
      )

      roundUnsub = useViewerStore.subscribe(
        (s) => ({ round: s.currentRound, demoId: s.demoId }),
        ({ round, demoId }) => {
          if (!playerLayer || !demoId) return

          rosterAbortController?.abort()
          rosterAbortController = new AbortController()

          fetchRoster(demoId, round, rosterAbortController.signal)
            .then((entries) => {
              playerLayer?.setRoster(entries)
            })
            .catch((err: unknown) => {
              if (err instanceof Error && err.name !== "AbortError") {
                console.error("Failed to fetch roster:", err)
              }
            })
        },
        { fireImmediately: true }
      )
    })

    return () => {
      destroyed = true
      rosterAbortController?.abort()
      roundUnsub?.()
      tickUnsub?.()
      demoUnsub?.()
      mapUnsub?.()
      unsubscribe?.()
      tickBuffer?.dispose()
      playerLayer?.destroy()
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
