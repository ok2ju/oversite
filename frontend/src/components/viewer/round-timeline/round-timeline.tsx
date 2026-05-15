"use client"

import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react"
import {
  MAX_TIMELINE_ZOOM,
  MIN_TIMELINE_ZOOM,
  useViewerStore,
} from "@/stores/viewer"
import { useRoundTimelineModel } from "@/hooks/use-round-timeline-model"
import { TooltipProvider } from "@/components/ui/tooltip"
import { clientXToPercent } from "@/lib/viewer/timeline-utils"
import { findActiveContact } from "@/lib/timeline/contacts"
import { EventsTrack } from "./events-track"
import { ContactsLane } from "./contacts-lane"
import { DuelsLane } from "./duels-lane"
import { Playhead } from "./playhead"
import { useDuelTimeline } from "@/hooks/use-duel-timeline"
import { useMistakeTimeline } from "@/hooks/use-mistake-timeline"
import type { Round } from "@/types/round"

interface RoundTimelineProps {
  round: Round
}

// Top-level orchestrator. Builds the model, measures its own width so the
// clustering pass can decide what overlaps, paints the lanes + spine + mistakes
// + playhead, and handles scrub-drag input across the full track.
export function RoundTimeline({ round }: RoundTimelineProps) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const trackRef = useRef<HTMLDivElement>(null)
  const draggingRef = useRef(false)
  const seekRef = useRef<(clientX: number) => void>(null)
  const activeMouseListenersRef = useRef<{
    move: (ev: MouseEvent) => void
    up: () => void
  } | null>(null)
  const activeTouchListenersRef = useRef<{
    move: (ev: TouchEvent) => void
    end: () => void
  } | null>(null)
  // Captured by the wheel/pinch handler before each zoom step so the
  // upcoming re-render can hold the point under the cursor in place.
  // { pct: 0..1 along the track, cursorX: pixels from scroll-container left }
  const zoomAnchorRef = useRef<{ pct: number; cursorX: number } | null>(null)
  // Accumulator for trackpad wheel deltas — single physical scroll often
  // fires many small events; we only step zoom once a threshold is crossed.
  const wheelDeltaRef = useRef(0)
  const [containerWidth, setContainerWidth] = useState(800)
  const timelineZoom = useViewerStore((s) => s.timelineZoom)
  // The inner track stretches to containerWidth * zoom; the scroll container
  // shows a viewport of the dock's width. Clustering sees the inflated width,
  // so dense windows un-cluster as the user zooms in.
  const trackWidth = Math.max(1, Math.round(containerWidth * timelineZoom))

  useEffect(() => {
    const node = scrollRef.current
    if (!node) return
    const update = () => setContainerWidth(node.clientWidth)
    update()
    const ro = new ResizeObserver(update)
    ro.observe(node)
    return () => ro.disconnect()
  }, [])

  const { model } = useRoundTimelineModel(round, trackWidth)
  const demoId = useViewerStore((s) => s.demoId)
  const selectedPlayerSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const currentTick = useViewerStore((s) => s.currentTick)
  const { data: duels } = useDuelTimeline(demoId, selectedPlayerSteamId)
  const { data: mistakes } = useMistakeTimeline(demoId, selectedPlayerSteamId)

  // Re-anchor the scroll viewport whenever the zoom changes. If the wheel
  // handler captured a cursor point we hold *that* tick under the cursor
  // (so pinch-zoom feels like zooming around the cursor); otherwise we
  // center on the playhead, which is the right behaviour for button clicks.
  useLayoutEffect(() => {
    const scroll = scrollRef.current
    if (!scroll) return
    const maxScroll = Math.max(0, scroll.scrollWidth - scroll.clientWidth)
    let target: number
    const anchor = zoomAnchorRef.current
    if (anchor) {
      target = anchor.pct * trackWidth - anchor.cursorX
      zoomAnchorRef.current = null
    } else {
      const start =
        round.freeze_end_tick > 0 ? round.freeze_end_tick : round.start_tick
      const end = round.end_tick
      const span = Math.max(1, end - start)
      const tick = useViewerStore.getState().currentTick
      const pct = Math.max(0, Math.min(1, (tick - start) / span))
      target = pct * trackWidth - scroll.clientWidth / 2
    }
    scroll.scrollLeft = Math.max(0, Math.min(target, maxScroll))
    // currentTick handled by the follow-the-playhead effect below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [timelineZoom, trackWidth])

  // Wheel + trackpad pinch zoom. Pinch on macOS trackpads fires a wheel event
  // with `ctrlKey: true`; we also accept plain vertical wheel deltas (the
  // timeline doesn't scroll vertically, so the gesture is unambiguous).
  // Horizontal wheel deltas pass through to native overflow-x scroll so
  // two-finger pans still work.
  useEffect(() => {
    const node = scrollRef.current
    if (!node) return
    const WHEEL_STEP_THRESHOLD = 40
    const onWheel = (e: WheelEvent) => {
      const horizontal = Math.abs(e.deltaX) > Math.abs(e.deltaY)
      // Let the browser handle two-finger horizontal pans natively.
      if (horizontal && !e.ctrlKey) return

      e.preventDefault()
      wheelDeltaRef.current += e.deltaY
      if (Math.abs(wheelDeltaRef.current) < WHEEL_STEP_THRESHOLD) return

      const direction: 1 | -1 = wheelDeltaRef.current < 0 ? 1 : -1
      wheelDeltaRef.current = 0

      const state = useViewerStore.getState()
      const atMax = state.timelineZoom >= MAX_TIMELINE_ZOOM - 1e-6
      const atMin = state.timelineZoom <= MIN_TIMELINE_ZOOM + 1e-6
      if ((direction === 1 && atMax) || (direction === -1 && atMin)) return

      // Capture the cursor anchor so the upcoming re-render holds the
      // point under the cursor in place after the track inflates.
      const track = trackRef.current
      if (track) {
        const containerRect = node.getBoundingClientRect()
        const trackRect = track.getBoundingClientRect()
        const cursorX = e.clientX - containerRect.left
        const pct = Math.max(
          0,
          Math.min(1, (e.clientX - trackRect.left) / trackRect.width),
        )
        zoomAnchorRef.current = { pct, cursorX }
      }

      if (direction === 1) state.zoomTimelineIn()
      else state.zoomTimelineOut()
    }
    node.addEventListener("wheel", onWheel, { passive: false })
    return () => node.removeEventListener("wheel", onWheel)
  }, [])

  // Once zoomed in, keep the playhead in view as the tick advances. Only nudge
  // when it drifts past the visible margins so we don't fight manual scroll.
  useEffect(() => {
    const scroll = scrollRef.current
    if (!scroll) return
    if (timelineZoom <= 1) return
    const start =
      round.freeze_end_tick > 0 ? round.freeze_end_tick : round.start_tick
    const end = round.end_tick
    const span = Math.max(1, end - start)
    const pct = Math.max(0, Math.min(1, (currentTick - start) / span))
    const playheadX = pct * trackWidth
    const visibleStart = scroll.scrollLeft
    const visibleEnd = visibleStart + scroll.clientWidth
    const margin = scroll.clientWidth * 0.1
    if (playheadX < visibleStart + margin) {
      scroll.scrollTo({
        left: Math.max(0, playheadX - margin),
        behavior: "smooth",
      })
    } else if (playheadX > visibleEnd - margin) {
      scroll.scrollTo({
        left: Math.max(0, playheadX - scroll.clientWidth + margin),
        behavior: "smooth",
      })
    }
  }, [
    currentTick,
    timelineZoom,
    trackWidth,
    round.start_tick,
    round.freeze_end_tick,
    round.end_tick,
  ])

  // Derive the active-contact id from the playhead. `currentTick` is the
  // single source of truth — every seek (click / scrub / round switch)
  // updates this without store coupling. useMemo short-circuits the
  // marker re-render when the playhead stays inside the same window.
  const contacts = model?.contacts
  const activeContactId = useMemo(
    () =>
      contacts ? (findActiveContact(contacts, currentTick)?.id ?? null) : null,
    [contacts, currentTick],
  )

  // Detach any in-flight document listeners on unmount so a drag that's still
  // active when the component goes away doesn't leak handlers.
  useEffect(() => {
    return () => {
      const mouse = activeMouseListenersRef.current
      if (mouse) {
        document.removeEventListener("mousemove", mouse.move)
        document.removeEventListener("mouseup", mouse.up)
        activeMouseListenersRef.current = null
      }
      const touch = activeTouchListenersRef.current
      if (touch) {
        document.removeEventListener("touchmove", touch.move)
        document.removeEventListener("touchend", touch.end)
        activeTouchListenersRef.current = null
      }
      draggingRef.current = false
    }
  }, [])

  // Keep the seek closure fresh whenever the round window changes — the
  // document listeners hold a ref to it, so swapping rounds mid-drag stays
  // accurate. The track spans the live phase only, so scrubbing maps to
  // [freeze_end_tick, end_tick].
  useEffect(() => {
    const start =
      round.freeze_end_tick > 0 ? round.freeze_end_tick : round.start_tick
    const end = round.end_tick
    const span = Math.max(1, end - start)
    seekRef.current = (clientX: number) => {
      const node = trackRef.current
      if (!node) return
      const rect = node.getBoundingClientRect()
      const pct = clientXToPercent(clientX, rect)
      const tick = start + Math.round((pct / 100) * span)
      const state = useViewerStore.getState()
      state.pause()
      state.setTick(tick)
    }
  }, [round.start_tick, round.freeze_end_tick, round.end_tick])

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    // Ignore clicks on interactive children (event icons, cluster popouts);
    // they handle their own seek.
    if ((e.target as HTMLElement).closest("button, [role='menu']")) return
    draggingRef.current = true
    seekRef.current?.(e.clientX)
    const handleMouseMove = (ev: MouseEvent) => {
      if (draggingRef.current) seekRef.current?.(ev.clientX)
    }
    const handleMouseUp = () => {
      draggingRef.current = false
      document.removeEventListener("mousemove", handleMouseMove)
      document.removeEventListener("mouseup", handleMouseUp)
      activeMouseListenersRef.current = null
    }
    document.addEventListener("mousemove", handleMouseMove)
    document.addEventListener("mouseup", handleMouseUp)
    activeMouseListenersRef.current = {
      move: handleMouseMove,
      up: handleMouseUp,
    }
  }, [])

  const handleTouchStart = useCallback((e: React.TouchEvent) => {
    if ((e.target as HTMLElement).closest("button, [role='menu']")) return
    draggingRef.current = true
    seekRef.current?.(e.touches[0].clientX)
    const handleTouchMove = (ev: TouchEvent) => {
      ev.preventDefault()
      if (draggingRef.current) seekRef.current?.(ev.touches[0].clientX)
    }
    const handleTouchEnd = () => {
      draggingRef.current = false
      document.removeEventListener("touchmove", handleTouchMove)
      document.removeEventListener("touchend", handleTouchEnd)
      activeTouchListenersRef.current = null
    }
    document.addEventListener("touchmove", handleTouchMove, { passive: false })
    document.addEventListener("touchend", handleTouchEnd)
    activeTouchListenersRef.current = {
      move: handleTouchMove,
      end: handleTouchEnd,
    }
  }, [])

  if (!model) {
    return (
      <div
        ref={scrollRef}
        data-testid="round-timeline"
        className="flex h-[88px] items-center justify-center text-[10px] text-white/40"
      >
        Loading round timeline…
      </div>
    )
  }

  const isPlayerMode = !!model.selectedPlayerSteamId

  return (
    <TooltipProvider delayDuration={150}>
      <div
        ref={scrollRef}
        data-testid="round-timeline-scroll"
        className="round-timeline-scroll w-full overflow-x-auto overflow-y-hidden"
      >
        <div
          ref={trackRef}
          data-testid="round-timeline"
          role="slider"
          aria-valuemin={model.roundStartTick}
          aria-valuemax={model.roundEndTick}
          aria-valuenow={useViewerStore.getState().currentTick}
          aria-label="Round timeline"
          tabIndex={0}
          onMouseDown={handleMouseDown}
          onTouchStart={handleTouchStart}
          style={{ width: trackWidth }}
          className="relative flex cursor-pointer flex-col gap-1 select-none"
        >
          <EventsTrack
            clusters={model.events}
            spine={model.spine}
            roundStartTick={model.roundStartTick}
            roundEndTick={model.roundEndTick}
          />
          {isPlayerMode ? (
            <>
              <ContactsLane
                contacts={model.contacts}
                roundStartTick={model.roundStartTick}
                roundEndTick={model.roundEndTick}
                hasPlayer
                activeContactId={activeContactId}
              />
              <DuelsLane
                duels={duels ?? []}
                mistakes={mistakes ?? []}
                roundStartTick={model.roundStartTick}
                roundEndTick={model.roundEndTick}
                roundNumber={round.round_number}
                selectedPlayerSteamId={selectedPlayerSteamId}
                hasPlayer
              />
            </>
          ) : null}
          <Playhead
            roundStartTick={model.roundStartTick}
            roundEndTick={model.roundEndTick}
          />
        </div>
      </div>
    </TooltipProvider>
  )
}
