import { Sprite, Texture, type Container } from "pixi.js"
import type { MapCalibration } from "@/lib/maps/calibration"
import { worldToPixel } from "@/lib/maps/calibration"
import { computeKDE, type KDEPoint } from "../kde"
import { fillImageDataFromDensity } from "../colormap"
import type { HeatmapPoint } from "@/types/heatmap"

const DEFAULT_GRID_SIZE = 512
const DEFAULT_BANDWIDTH = 18
const DEFAULT_OPACITY = 0.7

export interface HeatmapOptions {
  gridSize?: number
  bandwidth?: number
  opacity?: number
}

/**
 * Render a density grid to an offscreen canvas and return it.
 * Extracted for testability — jsdom lacks canvas 2d context.
 */
export function renderDensityToCanvas(
  densityData: Float32Array,
  maxDensity: number,
  gridSize: number,
  opacity: number,
): HTMLCanvasElement {
  const canvas = document.createElement("canvas")
  canvas.width = gridSize
  canvas.height = gridSize
  const ctx = canvas.getContext("2d")!
  const imageData = ctx.createImageData(gridSize, gridSize)

  fillImageDataFromDensity(imageData.data, densityData, maxDensity, opacity)
  ctx.putImageData(imageData, 0, 0)

  return canvas
}

export class HeatmapLayer {
  private container: Container
  private sprite: Sprite | null = null

  constructor(container: Container) {
    this.container = container
  }

  render(
    points: HeatmapPoint[],
    calibration: MapCalibration,
    options?: HeatmapOptions,
  ): void {
    this.clear()

    if (points.length === 0) return

    const gridSize = options?.gridSize ?? DEFAULT_GRID_SIZE
    const bandwidth = options?.bandwidth ?? DEFAULT_BANDWIDTH
    const opacity = options?.opacity ?? DEFAULT_OPACITY

    // Convert world coordinates to pixel-space KDE points
    const kdePoints: KDEPoint[] = points.map((p) => {
      const pixel = worldToPixel({ x: p.x, y: p.y }, calibration)
      const scaleX = gridSize / calibration.width
      const scaleY = gridSize / calibration.height
      return {
        x: pixel.x * scaleX,
        y: pixel.y * scaleY,
        weight: p.kill_count,
      }
    })

    // Compute density grid
    const grid = computeKDE(kdePoints, gridSize, gridSize, bandwidth)

    if (grid.maxDensity <= 0) return

    const canvas = renderDensityToCanvas(
      grid.data,
      grid.maxDensity,
      gridSize,
      opacity,
    )

    // Create PixiJS texture from canvas and add sprite
    const texture = Texture.from(canvas)
    const sprite = new Sprite({ texture })
    sprite.width = calibration.width
    sprite.height = calibration.height

    this.container.addChild(sprite)
    this.sprite = sprite
  }

  clear(): void {
    if (this.sprite) {
      this.container.removeChild(this.sprite)
      this.sprite.destroy({ texture: true })
      this.sprite = null
    }
  }

  destroy(): void {
    this.clear()
  }
}
