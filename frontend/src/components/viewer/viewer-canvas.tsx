"use client"

import { useEffect, useRef } from "react"
import { createViewerApp, type ViewerApp } from "@/lib/pixi/app"
import { PlaybackEngine } from "@/lib/pixi/playback-engine"
import { Camera } from "@/lib/pixi/camera"
import { MapLayer } from "@/lib/pixi/layers/map-layer"
import { PlayerLayer } from "@/lib/pixi/layers/player-layer"
import { EventLayer } from "@/lib/pixi/layers/event-layer"
import { TickBuffer } from "@/lib/pixi/tick-buffer"
import { fetchRoster } from "@/hooks/use-roster"
import { useViewerStore } from "@/stores/viewer"
import { useGameEvents } from "@/hooks/use-game-events"
import { useRounds } from "@/hooks/use-rounds"
import { shallow } from "zustand/shallow"
import type { Round } from "@/types/round"

function toFreezeWindows(rounds: Round[] | undefined) {
  return (rounds ?? []).map((r) => ({
    startTick: r.start_tick,
    freezeEndTick: r.freeze_end_tick,
  }))
}

export function ViewerCanvas() {
  const containerRef = useRef<HTMLDivElement>(null)
  const eventLayerRef = useRef<EventLayer | null>(null)
  const engineRef = useRef<PlaybackEngine | null>(null)
  const roundsRef = useRef<ReturnType<typeof useRounds>["data"]>(undefined)

  const demoId = useViewerStore((s) => s.demoId)
  const { data: gameEventsData } = useGameEvents(demoId)
  const { data: roundsData } = useRounds(demoId)
  roundsRef.current = roundsData

  // Feed event data into the EventLayer whenever events or rounds change.
  // Rounds are needed so per-effect durations can be capped at round-end
  // (smokes / fires / trajectories shouldn't bleed into the next round).
  useEffect(() => {
    if (gameEventsData && eventLayerRef.current) {
      eventLayerRef.current.setEvents(gameEventsData, roundsData)
    }
  }, [gameEventsData, roundsData])

  // Push freeze windows into the engine so it auto-skips freeze time on
  // initial load, seek, and round transitions during live playback.
  useEffect(() => {
    if (!engineRef.current) return
    engineRef.current.setFreezeWindows(toFreezeWindows(roundsData))
  }, [roundsData])

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    let destroyed = false
    let viewerApp: ViewerApp | null = null
    let camera: Camera | null = null
    let mapLayer: MapLayer | null = null
    let playerLayer: PlayerLayer | null = null
    let tickBuffer: TickBuffer | null = null
    let eventLayer: EventLayer | null = null
    let mapUnsub: (() => void) | null = null
    let roundUnsub: (() => void) | null = null
    let demoUnsub: (() => void) | null = null
    let resetUnsub: (() => void) | null = null
    let rosterAbortController: AbortController | null = null
    let tickerFn: (() => void) | null = null
    let engine: PlaybackEngine | null = null
    let seekUnsub: (() => void) | null = null
    let playingUnsub: (() => void) | null = null
    let resizeObserver: ResizeObserver | null = null

    createViewerApp({ container }).then((app) => {
      if (destroyed) {
        app.destroy()
        return
      }

      viewerApp = app

      // Create camera and add its container to stage
      camera = new Camera(app.canvas, {
        onViewportChange: (v) => useViewerStore.getState().setViewport(v),
      })
      app.stage.addChild(camera.container)

      // Set initial screen size
      const { width, height } = container.getBoundingClientRect()
      camera.setScreenSize(width, height)
      useViewerStore.getState().setScreenSize(width, height)

      // Observe container resize
      resizeObserver = new ResizeObserver((entries) => {
        const entry = entries[0]
        if (!entry || !camera) return
        const { width: w, height: h } = entry.contentRect
        camera.setScreenSize(w, h)
        useViewerStore.getState().setScreenSize(w, h)
      })
      resizeObserver.observe(container)

      // All layers are children of camera.container
      const mapContainer = app.addLayer("map", camera.container)
      mapLayer = new MapLayer(mapContainer)

      const playerContainer = app.addLayer("players", camera.container)
      playerLayer = new PlayerLayer(playerContainer)

      playerLayer.onPlayerClick((steamId) => {
        const { selectedPlayerSteamId, setSelectedPlayer } =
          useViewerStore.getState()
        setSelectedPlayer(selectedPlayerSteamId === steamId ? null : steamId)
      })

      const eventContainer = app.addLayer("events", camera.container)
      eventLayer = new EventLayer(eventContainer)
      eventLayerRef.current = eventLayer

      // Track ticks written by the engine to avoid seek feedback loops
      let engineSetTick = -1

      engine = new PlaybackEngine({
        tickRate: useViewerStore.getState().tickRate,
        getState: () => {
          const s = useViewerStore.getState()
          return {
            currentTick: s.currentTick,
            totalTicks: s.totalTicks,
            isPlaying: s.isPlaying,
            speed: s.speed,
          }
        },
        setTick: (tick) => {
          engineSetTick = tick
          useViewerStore.getState().setTick(tick)
        },
        pause: () => useViewerStore.getState().pause(),
      })
      engineRef.current = engine
      // If rounds already loaded before the engine was ready, push now so
      // the initial seek(0) gets snapped out of round 1's freeze window.
      engine.setFreezeWindows(toFreezeWindows(roundsRef.current))

      // Sync external seek changes (timeline UI) to the engine
      seekUnsub = useViewerStore.subscribe(
        (s) => s.currentTick,
        (currentTick) => {
          if (engine && currentTick !== engineSetTick) {
            engine.seek(currentTick)
            tickBuffer?.seek(currentTick)
            // While paused, the ticker is stopped, so the scrub's new frame
            // won't render unless we paint one. engine.update short-circuits
            // on !isPlaying so tickerFn is safe to call manually.
            if (!useViewerStore.getState().isPlaying && tickerFn) {
              tickerFn()
            }
          }
        },
      )

      // Drive the PixiJS ticker off of isPlaying so we don't burn CPU
      // rendering interpolated frames while the demo is paused.
      playingUnsub = useViewerStore.subscribe(
        (s) => s.isPlaying,
        (isPlaying) => {
          if (isPlaying) {
            app.ticker.start()
          } else {
            app.ticker.stop()
          }
        },
        { fireImmediately: true },
      )

      resetUnsub = useViewerStore.subscribe(
        (s) => s.resetViewportCounter,
        () => {
          camera?.resetView()
        },
      )

      mapUnsub = useViewerStore.subscribe(
        (s) => s.mapName,
        (mapName) => {
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
                // Map loaded asynchronously — paint a frame even if paused so
                // the initial view renders without waiting for play.
                if (!useViewerStore.getState().isPlaying) {
                  tickerFn?.()
                }
              })
              .catch(console.error)
          } else {
            mapLayer?.clear()
          }
        },
        { fireImmediately: true },
      )

      demoUnsub = useViewerStore.subscribe(
        (s) => s.demoId,
        (demoId) => {
          tickBuffer?.dispose()
          tickBuffer = demoId ? new TickBuffer(demoId) : null
          // Reset engine state so fractionalTick doesn't carry over from previous demo
          engine?.seek(0)
          engineSetTick = 0
          camera?.resetView()
        },
        { fireImmediately: true },
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
        { fireImmediately: true, equalityFn: shallow },
      )

      tickerFn = () => {
        engine?.update(app.ticker.deltaMS)

        const calibration = mapLayer?.calibration
        if (!calibration || !engine) return

        const { currentTick, selectedPlayerSteamId } = useViewerStore.getState()

        if (eventLayer) {
          eventLayer.update(currentTick, calibration)
        }

        if (playerLayer && tickBuffer) {
          const fractionalTick = currentTick + engine.interpolationFactor
          const { current, next } = tickBuffer.getFramePair(fractionalTick)
          if (current) {
            const alpha =
              next && next.tick > current.tick
                ? (fractionalTick - current.tick) / (next.tick - current.tick)
                : 0
            playerLayer.update(
              current.data,
              next?.data ?? null,
              alpha,
              calibration,
              selectedPlayerSteamId,
            )
          }
        }
      }
      app.ticker.add(tickerFn)
    })

    return () => {
      destroyed = true
      resizeObserver?.disconnect()
      rosterAbortController?.abort()
      roundUnsub?.()
      demoUnsub?.()
      seekUnsub?.()
      resetUnsub?.()
      playingUnsub?.()
      if (tickerFn && viewerApp) {
        viewerApp.ticker.remove(tickerFn)
      }
      engine?.dispose()
      engineRef.current = null
      mapUnsub?.()
      tickBuffer?.dispose()
      playerLayer?.destroy()
      eventLayer?.destroy()
      eventLayerRef.current = null
      mapLayer?.destroy()
      camera?.destroy()
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
