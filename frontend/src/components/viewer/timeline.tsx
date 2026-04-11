"use client"

import { useCallback, useRef } from "react"
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

  const percent = tickToPercent(currentTick, totalTicks)
  const markers = roundBoundaryPositions(roundBoundaries, totalTicks)

  const seekFromClientX = useCallback(
    (clientX: number) => {
      const track = trackRef.current
      if (!track) return
      const rect = track.getBoundingClientRect()
      const pct = clientXToPercent(clientX, rect)
      const tick = percentToTick(pct, totalTicks)
      onSeek(tick)
    },
    [totalTicks, onSeek],
  )

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      draggingRef.current = true
      onScrubStart?.()
      seekFromClientX(e.clientX)

      const handleMouseMove = (ev: MouseEvent) => {
        if (draggingRef.current) {
          seekFromClientX(ev.clientX)
        }
      }

      const handleMouseUp = () => {
        draggingRef.current = false
        document.removeEventListener("mousemove", handleMouseMove)
        document.removeEventListener("mouseup", handleMouseUp)
      }

      document.addEventListener("mousemove", handleMouseMove)
      document.addEventListener("mouseup", handleMouseUp)
    },
    [seekFromClientX, onScrubStart],
  )

  const handleTouchStart = useCallback(
    (e: React.TouchEvent) => {
      draggingRef.current = true
      onScrubStart?.()
      seekFromClientX(e.touches[0].clientX)

      const handleTouchMove = (ev: TouchEvent) => {
        if (draggingRef.current) {
          seekFromClientX(ev.touches[0].clientX)
        }
      }

      const handleTouchEnd = () => {
        draggingRef.current = false
        document.removeEventListener("touchmove", handleTouchMove)
        document.removeEventListener("touchend", handleTouchEnd)
      }

      document.addEventListener("touchmove", handleTouchMove)
      document.addEventListener("touchend", handleTouchEnd)
    },
    [seekFromClientX, onScrubStart],
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
