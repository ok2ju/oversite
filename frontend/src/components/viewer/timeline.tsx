"use client"

import { useCallback, useEffect, useRef } from "react"
import {
  tickToPercent,
  percentToTick,
  clientXToPercent,
  roundBoundaryPositions,
} from "@/lib/viewer/timeline-utils"

interface TimelineProps {
  currentTick: number
  totalTicks: number
  roundBoundaries: Array<{
    roundNumber: number
    startTick: number
    endTick: number
  }>
  onSeek: (tick: number) => void
  onScrubStart?: () => void
}

export function Timeline({
  currentTick,
  totalTicks,
  roundBoundaries,
  onSeek,
  onScrubStart,
}: TimelineProps) {
  const trackRef = useRef<HTMLDivElement>(null)
  const draggingRef = useRef(false)
  const seekRef = useRef<(clientX: number) => void>(null)
  // Track the active document-level listener pair so we can detach them
  // even if the component unmounts mid-drag (otherwise the listeners leak
  // and keep firing into a destroyed React tree).
  const activeMouseListenersRef = useRef<{
    move: (ev: MouseEvent) => void
    up: () => void
  } | null>(null)
  const activeTouchListenersRef = useRef<{
    move: (ev: TouchEvent) => void
    end: () => void
  } | null>(null)

  const percent = tickToPercent(currentTick, totalTicks)
  const markers = roundBoundaryPositions(roundBoundaries, totalTicks)

  // Keep a ref to the latest seek function so document-level listeners
  // never hold a stale closure over totalTicks or onSeek.
  useEffect(() => {
    seekRef.current = (clientX: number) => {
      const track = trackRef.current
      if (!track) return
      const rect = track.getBoundingClientRect()
      const pct = clientXToPercent(clientX, rect)
      const tick = percentToTick(pct, totalTicks)
      onSeek(tick)
    }
  }, [totalTicks, onSeek])

  // Detach any in-flight document listeners on unmount so a drag that's
  // still active when the component goes away doesn't leak handlers.
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

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      draggingRef.current = true
      onScrubStart?.()
      seekRef.current?.(e.clientX)

      const handleMouseMove = (ev: MouseEvent) => {
        if (draggingRef.current) {
          seekRef.current?.(ev.clientX)
        }
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
    },
    [onScrubStart],
  )

  const handleTouchStart = useCallback(
    (e: React.TouchEvent) => {
      draggingRef.current = true
      onScrubStart?.()
      seekRef.current?.(e.touches[0].clientX)

      const handleTouchMove = (ev: TouchEvent) => {
        ev.preventDefault()
        if (draggingRef.current) {
          seekRef.current?.(ev.touches[0].clientX)
        }
      }

      const handleTouchEnd = () => {
        draggingRef.current = false
        document.removeEventListener("touchmove", handleTouchMove)
        document.removeEventListener("touchend", handleTouchEnd)
        activeTouchListenersRef.current = null
      }

      document.addEventListener("touchmove", handleTouchMove, {
        passive: false,
      })
      document.addEventListener("touchend", handleTouchEnd)
      activeTouchListenersRef.current = {
        move: handleTouchMove,
        end: handleTouchEnd,
      }
    },
    [onScrubStart],
  )

  return (
    <div
      data-testid="timeline-track"
      ref={trackRef}
      role="slider"
      aria-valuemin={0}
      aria-valuemax={totalTicks}
      aria-valuenow={currentTick}
      aria-label="Playback timeline"
      tabIndex={0}
      className="relative flex h-5 cursor-pointer items-center"
      onMouseDown={handleMouseDown}
      onTouchStart={handleTouchStart}
    >
      {/* Track background */}
      <div className="h-2 w-full rounded-full bg-white/20">
        {/* Progress fill */}
        <div
          data-testid="timeline-progress"
          className="h-full rounded-full bg-primary"
          style={{ width: `${percent}%` }}
        />
      </div>

      {/* Round boundary markers */}
      {markers.map((m) => (
        <div
          key={m.roundNumber}
          data-testid="round-marker"
          className="absolute top-1 h-3 w-px bg-white/40"
          style={{ left: `${m.percent}%` }}
        />
      ))}

      {/* Thumb */}
      <div
        data-testid="timeline-thumb"
        className="absolute h-3 w-3 rounded-full bg-white shadow"
        style={{ left: `${percent}%`, transform: "translateX(-50%)" }}
      />
    </div>
  )
}
