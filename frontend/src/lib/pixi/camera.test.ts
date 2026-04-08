import { describe, it, expect } from "vitest"
import {
  clampZoom,
  zoomToPoint,
  clampPan,
  computeViewportRect,
  screenToWorld,
  MIN_ZOOM,
  MAX_ZOOM,
  DEFAULT_VIEWPORT,
  type Viewport,
} from "./camera"

describe("camera pure functions", () => {
  describe("clampZoom", () => {
    it("returns value within bounds unchanged", () => {
      expect(clampZoom(1)).toBe(1)
      expect(clampZoom(2.5)).toBe(2.5)
    })

    it("clamps below MIN_ZOOM", () => {
      expect(clampZoom(0.1)).toBe(MIN_ZOOM)
      expect(clampZoom(0)).toBe(MIN_ZOOM)
      expect(clampZoom(-1)).toBe(MIN_ZOOM)
    })

    it("clamps above MAX_ZOOM", () => {
      expect(clampZoom(5)).toBe(MAX_ZOOM)
      expect(clampZoom(100)).toBe(MAX_ZOOM)
    })

    it("returns exact boundary values", () => {
      expect(clampZoom(MIN_ZOOM)).toBe(MIN_ZOOM)
      expect(clampZoom(MAX_ZOOM)).toBe(MAX_ZOOM)
    })
  })

  describe("zoomToPoint", () => {
    it("keeps the cursor point fixed after zoom in", () => {
      const viewport: Viewport = { x: 0, y: 0, zoom: 1 }
      const result = zoomToPoint(viewport, 200, 300, 2)

      // The world point under cursor before zoom
      const worldX = (200 - viewport.x) / viewport.zoom // 200
      const worldY = (300 - viewport.y) / viewport.zoom // 300

      // After zoom, that same world point should still be at (200, 300) screen
      expect(result.x + worldX * result.zoom).toBeCloseTo(200)
      expect(result.y + worldY * result.zoom).toBeCloseTo(300)
      expect(result.zoom).toBe(2)
    })

    it("keeps the cursor point fixed after zoom out", () => {
      const viewport: Viewport = { x: -100, y: -50, zoom: 2 }
      const result = zoomToPoint(viewport, 400, 400, 1)

      const worldX = (400 - viewport.x) / viewport.zoom
      const worldY = (400 - viewport.y) / viewport.zoom

      expect(result.x + worldX * result.zoom).toBeCloseTo(400)
      expect(result.y + worldY * result.zoom).toBeCloseTo(400)
      expect(result.zoom).toBe(1)
    })

    it("clamps zoom to valid range", () => {
      const viewport: Viewport = { x: 0, y: 0, zoom: 1 }
      const result = zoomToPoint(viewport, 100, 100, 10)
      expect(result.zoom).toBe(MAX_ZOOM)
    })

    it("returns identity when zoom unchanged", () => {
      const viewport: Viewport = { x: -50, y: -30, zoom: 1.5 }
      const result = zoomToPoint(viewport, 100, 100, 1.5)
      expect(result.x).toBeCloseTo(viewport.x)
      expect(result.y).toBeCloseTo(viewport.y)
      expect(result.zoom).toBe(1.5)
    })
  })

  describe("clampPan", () => {
    const mapW = 1024
    const mapH = 1024

    it("returns viewport unchanged when fully visible at zoom 1", () => {
      const viewport: Viewport = { x: 0, y: 0, zoom: 1 }
      const result = clampPan(viewport, mapW, mapH, 1024, 1024)
      expect(result.x).toBe(0)
      expect(result.y).toBe(0)
    })

    it("clamps pan to prevent scrolling past right edge", () => {
      const viewport: Viewport = { x: -2000, y: 0, zoom: 2 }
      const result = clampPan(viewport, mapW, mapH, 800, 800)
      // minX = 800 - 1024*2 = -1248
      expect(result.x).toBe(-1248)
    })

    it("clamps pan to prevent scrolling past left edge", () => {
      const viewport: Viewport = { x: 500, y: 0, zoom: 2 }
      const result = clampPan(viewport, mapW, mapH, 800, 800)
      expect(result.x).toBe(0)
    })

    it("clamps pan to prevent scrolling past bottom edge", () => {
      const viewport: Viewport = { x: 0, y: -3000, zoom: 2 }
      const result = clampPan(viewport, mapW, mapH, 800, 800)
      // minY = 800 - 1024*2 = -1248
      expect(result.y).toBe(-1248)
    })

    it("clamps pan to prevent scrolling past top edge", () => {
      const viewport: Viewport = { x: 0, y: 200, zoom: 2 }
      const result = clampPan(viewport, mapW, mapH, 800, 800)
      expect(result.y).toBe(0)
    })

    it("centers map when zoomed out smaller than screen", () => {
      const viewport: Viewport = { x: 0, y: 0, zoom: 0.5 }
      const result = clampPan(viewport, mapW, mapH, 1024, 1024)
      // map size = 512, screen = 1024 -> centered at (256, 256)
      expect(result.x).toBe(256)
      expect(result.y).toBe(256)
    })
  })

  describe("computeViewportRect", () => {
    it("returns full map at zoom 1 with matching screen", () => {
      const viewport: Viewport = { x: 0, y: 0, zoom: 1 }
      const rect = computeViewportRect(viewport, 1024, 1024)
      expect(rect.x).toBeCloseTo(0)
      expect(rect.y).toBeCloseTo(0)
      expect(rect.width).toBeCloseTo(1024)
      expect(rect.height).toBeCloseTo(1024)
    })

    it("returns smaller rect when zoomed in", () => {
      const viewport: Viewport = { x: 0, y: 0, zoom: 2 }
      const rect = computeViewportRect(viewport, 1024, 1024)
      expect(rect.width).toBeCloseTo(512)
      expect(rect.height).toBeCloseTo(512)
    })

    it("offsets rect based on pan position", () => {
      const viewport: Viewport = { x: -200, y: -100, zoom: 2 }
      const rect = computeViewportRect(viewport, 1024, 1024)
      expect(rect.x).toBeCloseTo(100) // -(-200) / 2
      expect(rect.y).toBeCloseTo(50) // -(-100) / 2
    })

    it("handles non-square screen", () => {
      const viewport: Viewport = { x: 0, y: 0, zoom: 1 }
      const rect = computeViewportRect(viewport, 800, 600)
      expect(rect.width).toBeCloseTo(800)
      expect(rect.height).toBeCloseTo(600)
    })
  })

  describe("screenToWorld", () => {
    it("returns identity at default viewport", () => {
      const result = screenToWorld(100, 200, DEFAULT_VIEWPORT)
      expect(result.x).toBeCloseTo(100)
      expect(result.y).toBeCloseTo(200)
    })

    it("accounts for zoom", () => {
      const viewport: Viewport = { x: 0, y: 0, zoom: 2 }
      const result = screenToWorld(200, 400, viewport)
      expect(result.x).toBeCloseTo(100)
      expect(result.y).toBeCloseTo(200)
    })

    it("accounts for pan offset", () => {
      const viewport: Viewport = { x: -100, y: -50, zoom: 1 }
      const result = screenToWorld(200, 200, viewport)
      expect(result.x).toBeCloseTo(300)
      expect(result.y).toBeCloseTo(250)
    })

    it("accounts for both zoom and pan", () => {
      const viewport: Viewport = { x: -200, y: -100, zoom: 2 }
      const result = screenToWorld(400, 300, viewport)
      // worldX = (400 - (-200)) / 2 = 300
      // worldY = (300 - (-100)) / 2 = 200
      expect(result.x).toBeCloseTo(300)
      expect(result.y).toBeCloseTo(200)
    })
  })

  describe("constants", () => {
    it("MIN_ZOOM is 0.5", () => {
      expect(MIN_ZOOM).toBe(0.5)
    })

    it("MAX_ZOOM is 4.0", () => {
      expect(MAX_ZOOM).toBe(4.0)
    })

    it("DEFAULT_VIEWPORT is identity", () => {
      expect(DEFAULT_VIEWPORT).toEqual({ x: 0, y: 0, zoom: 1 })
    })
  })
})
