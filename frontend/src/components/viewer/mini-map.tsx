"use client"

import Image from "next/image"
import { Maximize2 } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import { computeViewportRect } from "@/lib/pixi/camera"
import { isCS2Map, getRadarImagePath } from "@/lib/maps/calibration"

const MINIMAP_SIZE = 150

export function MiniMap() {
  const mapName = useViewerStore((s) => s.mapName)
  const viewport = useViewerStore((s) => s.viewport)
  const screenWidth = useViewerStore((s) => s.screenWidth)
  const screenHeight = useViewerStore((s) => s.screenHeight)
  const resetViewport = useViewerStore((s) => s.resetViewport)

  if (!mapName || !isCS2Map(mapName)) return null

  const mapSize = 1024
  const rect = computeViewportRect(viewport, screenWidth, screenHeight)

  // Map viewport rect to minimap pixel space, clamped to 0-100%
  const left = Math.max(0, Math.min(100, (rect.x / mapSize) * 100))
  const top = Math.max(0, Math.min(100, (rect.y / mapSize) * 100))
  const width = Math.max(0, Math.min(100 - left, (rect.width / mapSize) * 100))
  const height = Math.max(0, Math.min(100 - top, (rect.height / mapSize) * 100))

  // If viewport covers the full map (or more), show 100%
  const isFullView = width >= 99.9 && height >= 99.9 && left < 0.1 && top < 0.1

  return (
    <div
      data-testid="mini-map"
      className="absolute bottom-4 right-4 overflow-hidden rounded-md border border-white/20 bg-black/60 shadow-lg"
      style={{ width: MINIMAP_SIZE, height: MINIMAP_SIZE }}
    >
      <Image
        src={getRadarImagePath(mapName)}
        alt={`${mapName} radar`}
        fill
        className="object-cover"
        draggable={false}
      />
      <div
        data-testid="mini-map-viewport-rect"
        className="pointer-events-none absolute border border-white/80"
        style={
          isFullView
            ? { left: 0, top: 0, width: "100%", height: "100%" }
            : {
                left: `${left}%`,
                top: `${top}%`,
                width: `${width}%`,
                height: `${height}%`,
              }
        }
      />
      <button
        onClick={resetViewport}
        aria-label="Reset view"
        className="absolute right-1 top-1 rounded bg-black/50 p-0.5 text-white/70 transition-colors hover:bg-black/70 hover:text-white"
      >
        <Maximize2 size={14} />
      </button>
    </div>
  )
}
