import { Container } from "pixi.js"

// --- Types ---

export interface Viewport {
  x: number
  y: number
  zoom: number
}

export interface ViewportRect {
  x: number
  y: number
  width: number
  height: number
}

// --- Constants ---

export const MIN_ZOOM = 0.5
export const MAX_ZOOM = 4.0
export const DEFAULT_VIEWPORT: Viewport = { x: 0, y: 0, zoom: 1 }

// --- Pure functions ---

export function clampZoom(zoom: number): number {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom))
}

export function zoomToPoint(
  viewport: Viewport,
  cursorX: number,
  cursorY: number,
  newZoom: number
): Viewport {
  const clamped = clampZoom(newZoom)
  const worldX = (cursorX - viewport.x) / viewport.zoom
  const worldY = (cursorY - viewport.y) / viewport.zoom
  return {
    x: cursorX - worldX * clamped,
    y: cursorY - worldY * clamped,
    zoom: clamped,
  }
}

export function clampPan(
  viewport: Viewport,
  mapW: number,
  mapH: number,
  screenW: number,
  screenH: number
): Viewport {
  const scaledW = mapW * viewport.zoom
  const scaledH = mapH * viewport.zoom

  let x: number
  let y: number

  if (scaledW <= screenW) {
    // Map smaller than screen — center it
    x = (screenW - scaledW) / 2
  } else {
    const minX = screenW - scaledW
    x = Math.min(0, Math.max(minX, viewport.x))
  }

  if (scaledH <= screenH) {
    y = (screenH - scaledH) / 2
  } else {
    const minY = screenH - scaledH
    y = Math.min(0, Math.max(minY, viewport.y))
  }

  return { x, y, zoom: viewport.zoom }
}

export function computeViewportRect(
  viewport: Viewport,
  screenW: number,
  screenH: number
): ViewportRect {
  return {
    x: -viewport.x / viewport.zoom,
    y: -viewport.y / viewport.zoom,
    width: screenW / viewport.zoom,
    height: screenH / viewport.zoom,
  }
}

export function screenToWorld(
  screenX: number,
  screenY: number,
  viewport: Viewport
): { x: number; y: number } {
  return {
    x: (screenX - viewport.x) / viewport.zoom,
    y: (screenY - viewport.y) / viewport.zoom,
  }
}

// --- Camera class ---

const DRAG_THRESHOLD = 3
const ZOOM_SENSITIVITY = 0.001

export interface CameraOptions {
  onViewportChange?: (viewport: Viewport) => void
}

export class Camera {
  readonly container: Container
  private canvas: HTMLCanvasElement
  private viewport: Viewport = { ...DEFAULT_VIEWPORT }
  private mapWidth = 1024
  private mapHeight = 1024
  private screenWidth = 0
  private screenHeight = 0
  private isDragging = false
  private dragStartX = 0
  private dragStartY = 0
  private dragStartViewportX = 0
  private dragStartViewportY = 0
  private hasDragged = false
  private onViewportChangeCb?: (viewport: Viewport) => void

  private boundOnWheel: (e: WheelEvent) => void
  private boundOnPointerDown: (e: PointerEvent) => void
  private boundOnPointerMove: (e: PointerEvent) => void
  private boundOnPointerUp: (e: PointerEvent) => void

  constructor(canvas: HTMLCanvasElement, options?: CameraOptions) {
    this.canvas = canvas
    this.container = new Container()
    this.container.label = "camera-viewport"
    this.onViewportChangeCb = options?.onViewportChange

    this.boundOnWheel = this.onWheel.bind(this)
    this.boundOnPointerDown = this.onPointerDown.bind(this)
    this.boundOnPointerMove = this.onPointerMove.bind(this)
    this.boundOnPointerUp = this.onPointerUp.bind(this)

    this.canvas.addEventListener("wheel", this.boundOnWheel, { passive: false })
    this.canvas.addEventListener("pointerdown", this.boundOnPointerDown)
    this.canvas.addEventListener("pointermove", this.boundOnPointerMove)
    this.canvas.addEventListener("pointerup", this.boundOnPointerUp)
    this.canvas.addEventListener("pointercancel", this.boundOnPointerUp)
  }

  setScreenSize(width: number, height: number): void {
    this.screenWidth = width
    this.screenHeight = height
    this.applyAndPublish()
  }

  setMapSize(width: number, height: number): void {
    this.mapWidth = width
    this.mapHeight = height
    this.applyAndPublish()
  }

  resetView(): void {
    this.viewport = { ...DEFAULT_VIEWPORT }
    this.applyAndPublish()
  }

  destroy(): void {
    this.canvas.removeEventListener("wheel", this.boundOnWheel)
    this.canvas.removeEventListener("pointerdown", this.boundOnPointerDown)
    this.canvas.removeEventListener("pointermove", this.boundOnPointerMove)
    this.canvas.removeEventListener("pointerup", this.boundOnPointerUp)
    this.canvas.removeEventListener("pointercancel", this.boundOnPointerUp)
  }

  private onWheel(e: WheelEvent): void {
    e.preventDefault()
    let deltaY = e.deltaY
    if (e.deltaMode === 1) deltaY *= 33
    else if (e.deltaMode === 2) deltaY *= 800
    const zoomFactor = 1 - deltaY * ZOOM_SENSITIVITY
    const newZoom = this.viewport.zoom * zoomFactor

    const rect = this.canvas.getBoundingClientRect()
    const cursorX = e.clientX - rect.left
    const cursorY = e.clientY - rect.top

    this.viewport = zoomToPoint(this.viewport, cursorX, cursorY, newZoom)
    this.viewport = clampPan(
      this.viewport,
      this.mapWidth,
      this.mapHeight,
      this.screenWidth,
      this.screenHeight
    )
    this.applyAndPublish()
  }

  private onPointerDown(e: PointerEvent): void {
    if (e.button !== 0) return
    this.isDragging = true
    this.hasDragged = false
    this.dragStartX = e.clientX
    this.dragStartY = e.clientY
    this.dragStartViewportX = this.viewport.x
    this.dragStartViewportY = this.viewport.y
    this.canvas.setPointerCapture(e.pointerId)
  }

  private onPointerMove(e: PointerEvent): void {
    if (!this.isDragging) return

    const dx = e.clientX - this.dragStartX
    const dy = e.clientY - this.dragStartY

    if (!this.hasDragged && Math.abs(dx) < DRAG_THRESHOLD && Math.abs(dy) < DRAG_THRESHOLD) {
      return
    }

    this.hasDragged = true
    const newViewport: Viewport = {
      x: this.dragStartViewportX + dx,
      y: this.dragStartViewportY + dy,
      zoom: this.viewport.zoom,
    }

    this.viewport = clampPan(
      newViewport,
      this.mapWidth,
      this.mapHeight,
      this.screenWidth,
      this.screenHeight
    )
    this.applyAndPublish()
  }

  private onPointerUp(e: PointerEvent): void {
    this.isDragging = false
    this.canvas.releasePointerCapture(e.pointerId)
  }

  private applyAndPublish(): void {
    this.container.position.set(this.viewport.x, this.viewport.y)
    this.container.scale.set(this.viewport.zoom)
    this.onViewportChangeCb?.({ ...this.viewport })
  }
}
