"use client"

import { useEffect, useRef, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { createViewerApp, type ViewerApp } from "@/lib/pixi/app"
import { PlaybackEngine } from "@/lib/pixi/playback-engine"
import { Camera } from "@/lib/pixi/camera"
import { MapLayer } from "@/lib/pixi/layers/map-layer"
import { PlayerLayer } from "@/lib/pixi/layers/player-layer"
import { EventLayer } from "@/lib/pixi/layers/event-layer"
import { TickBuffer } from "@/lib/pixi/tick-buffer"
import {
  fetchRoster,
  useAllRosters,
  type RosterByRound,
} from "@/hooks/use-roster"
import { useViewerStore } from "@/stores/viewer"
import { useTickBufferStore } from "@/stores/tick-buffer"
import { useGameEvents } from "@/hooks/use-game-events"
import { useRounds } from "@/hooks/use-rounds"
import { shallow } from "zustand/shallow"
import type { Round } from "@/types/round"
import type { GameEvent } from "@/types/demo"

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
  // The full per-round roster map is preloaded once per demo and read live by
  // the round-change subscription via this ref, so PixiJS round transitions
  // resolve roster locally instead of issuing one Wails call per round.
  const allRostersRef = useRef<RosterByRound | undefined>(undefined)

  const queryClient = useQueryClient()
  const demoId = useViewerStore((s) => s.demoId)
  // Once the EventLayer owns the events, we disable the observer for this
  // demoId and remove the cache. Without disabling, removeQueries would
  // immediately trigger a re-fetch since the observer is still active —
  // landing us with a duplicated payload again. Reset on demoId change.
  const [consumedDemoId, setConsumedDemoId] = useState<string | null>(null)
  const queryEnabled = consumedDemoId !== demoId
  const { data: gameEventsData } = useGameEvents(demoId, queryEnabled)
  const { data: roundsData } = useRounds(demoId)
  const { data: allRostersData } = useAllRosters(demoId)
  roundsRef.current = roundsData
  allRostersRef.current = allRostersData

  // Feed event data into the EventLayer whenever events or rounds change.
  // Rounds are needed so per-effect durations can be capped at round-end
  // (smokes / fires / trajectories shouldn't bleed into the next round).
  // After the EventLayer holds the parsed effect timeline (8K events ×
  // ~200 B = 1.6 MB), drop the React Query cache + disable the observer:
  // the layer is now the single owner. If the user navigates away and back,
  // the disabled-observer check flips on demoId change and a fresh fetch
  // runs.
  useEffect(() => {
    if (!gameEventsData || !eventLayerRef.current || !demoId) return
    eventLayerRef.current.setEvents(gameEventsData, roundsData)
    if (consumedDemoId !== demoId) {
      setConsumedDemoId(demoId)
      queryClient.removeQueries({
        queryKey: ["game-events", demoId],
        exact: true,
      })
    }
  }, [gameEventsData, roundsData, demoId, queryClient, consumedDemoId])

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
    let selectionUnsub: (() => void) | null = null
    let playingUnsub: (() => void) | null = null
    let resizeObserver: ResizeObserver | null = null
    // rAF handle for the viewport-write coalescer so cleanup can cancel a
    // pending write if the user unmounts mid-drag. Hoisted to outer scope so
    // the cleanup return below can see it.
    let viewportRafHandle: number | null = null

    createViewerApp({ container }).then((app) => {
      if (destroyed) {
        app.destroy()
        return
      }

      viewerApp = app

      // Create camera and add its container to stage. Pan/zoom drag emits
      // onViewportChange on every pointer-move tick (~60-120 Hz). No current
      // React subscriber re-renders on viewport changes, but a future minimap
      // would; coalescing into one write per animation frame caps store
      // updates at the display refresh rate without dropping the final value
      // (the rAF callback always reads the latest viewport).
      let pendingViewport:
        | ReturnType<typeof useViewerStore.getState>["viewport"]
        | null = null
      camera = new Camera(app.canvas, {
        onViewportChange: (v) => {
          pendingViewport = v
          if (viewportRafHandle !== null) return
          viewportRafHandle = requestAnimationFrame(() => {
            viewportRafHandle = null
            if (destroyed) return
            if (pendingViewport) {
              useViewerStore.getState().setViewport(pendingViewport)
              pendingViewport = null
            }
            // Pixi auto-renders via the ticker. When paused the ticker is
            // stopped, so pan/zoom would update container.position but never
            // paint. Manually render once per coalesced viewport change so
            // the user sees their drag while paused.
            if (!useViewerStore.getState().isPlaying) {
              app.render()
            }
          })
        },
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

      // Events may have arrived from React Query before the EventLayer existed;
      // the gameEventsData useEffect can't push them retroactively because it
      // doesn't see eventLayerRef changes. Pull directly from the cache once
      // (no extra retained copy in this component) and consume the same way
      // the useEffect would: hand off to the layer, then drop the cache and
      // disable the observer so removeQueries doesn't spawn a refetch.
      const cachedDemoId = useViewerStore.getState().demoId
      if (cachedDemoId) {
        const cached = queryClient.getQueryData<GameEvent[]>([
          "game-events",
          cachedDemoId,
        ])
        if (cached) {
          eventLayer.setEvents(cached, roundsRef.current)
          setConsumedDemoId(cachedDemoId)
          queryClient.removeQueries({
            queryKey: ["game-events", cachedDemoId],
            exact: true,
          })
        }
      }

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
            // While paused, app.ticker is stopped — calling tickerFn alone
            // would update sprite positions but skip the renderer's own
            // ticker listener, so nothing paints. ticker.update() pumps both
            // our tickerFn and the auto-render in one pass.
            if (!useViewerStore.getState().isPlaying) {
              app.ticker.update()
            }
          }
        },
      )

      // Selection changes are read inside tickerFn and forwarded to the
      // PlayerLayer, but the layer only repaints its rings on a ticker pass.
      // While paused the ticker is stopped, so a click in the scoreboard
      // would update the store without ever moving the indicator. Pump one
      // ticker update so the registered callbacks run AND the renderer
      // paints the new frame in the same pass.
      selectionUnsub = useViewerStore.subscribe(
        (s) => s.selectedPlayerSteamId,
        () => {
          if (!useViewerStore.getState().isPlaying) {
            app.ticker.update()
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
                // the initial view renders without waiting for play. Pump the
                // ticker so the renderer's auto-render listener fires alongside
                // tickerFn (calling tickerFn alone wouldn't paint).
                if (!useViewerStore.getState().isPlaying) {
                  app.ticker.update()
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
          // Publish the active buffer so consumers like useLoadoutSnapshot
          // share a single instance keyed by demoId rather than allocating
          // their own duplicate buffer + chunk cache.
          useTickBufferStore.getState().setBuffer(demoId, tickBuffer)
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

          fetchRoster(
            demoId,
            round,
            rosterAbortController.signal,
            allRostersRef.current,
          )
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
      if (viewportRafHandle !== null) {
        cancelAnimationFrame(viewportRafHandle)
        viewportRafHandle = null
      }
      roundUnsub?.()
      demoUnsub?.()
      seekUnsub?.()
      selectionUnsub?.()
      resetUnsub?.()
      playingUnsub?.()
      if (tickerFn && viewerApp) {
        viewerApp.ticker.remove(tickerFn)
      }
      engine?.dispose()
      engineRef.current = null
      mapUnsub?.()
      tickBuffer?.dispose()
      // Clear the published buffer so stale references can't be read after
      // unmount. demoUnsub.unsubscribe is already called above so the store
      // won't get re-populated by a late demoId change.
      useTickBufferStore.getState().setBuffer(null, null)
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
