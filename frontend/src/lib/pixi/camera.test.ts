import { describe, it, expect, vi, beforeEach } from "vitest"

vi.mock("pixi.js", () => ({
  Container: vi.fn().mockImplementation(function () {
    return {
      label: "",
      position: { set: vi.fn() },
      scale: { set: vi.fn() },
    }
  }),
}))

import {
  clampZoom,
  zoomToPoint,
  clampPan,
  computeViewportRect,
  screenToWorld,
  MIN_ZOOM,
  MAX_ZOOM,
  DEFAULT_VIEWPORT,
  Camera,
  type Viewport,
  type CameraOptions,
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

describe("Camera class", () => {
  let canvas: HTMLCanvasElement
  let onViewportChange: ReturnType<typeof vi.fn>

  beforeEach(() => {
    canvas = document.createElement("canvas")
    canvas.setPointerCapture = vi.fn()
    canvas.releasePointerCapture = vi.fn()
    canvas.getBoundingClientRect = vi.fn().mockReturnValue({
      left: 0,
      top: 0,
      width: 1024,
      height: 1024,
    })
    onViewportChange = vi.fn()
  })

  function createCamera(options?: CameraOptions) {
    return new Camera(canvas, options)
  }

  describe("constructor", () => {
    it("registers event listeners on canvas", () => {
      const addSpy = vi.spyOn(canvas, "addEventListener")
      const cam = createCamera()

      expect(addSpy).toHaveBeenCalledWith("wheel", expect.any(Function), { passive: false })
      expect(addSpy).toHaveBeenCalledWith("pointerdown", expect.any(Function))
      expect(addSpy).toHaveBeenCalledWith("pointermove", expect.any(Function))
      expect(addSpy).toHaveBeenCalledWith("pointerup", expect.any(Function))
      expect(addSpy).toHaveBeenCalledWith("pointercancel", expect.any(Function))
      cam.destroy()
    })

    it("creates a container with label", () => {
      const cam = createCamera()
      expect(cam.container.label).toBe("camera-viewport")
      cam.destroy()
    })
  })

  describe("destroy", () => {
    it("removes all event listeners", () => {
      const removeSpy = vi.spyOn(canvas, "removeEventListener")
      const cam = createCamera()
      cam.destroy()

      expect(removeSpy).toHaveBeenCalledWith("wheel", expect.any(Function))
      expect(removeSpy).toHaveBeenCalledWith("pointerdown", expect.any(Function))
      expect(removeSpy).toHaveBeenCalledWith("pointermove", expect.any(Function))
      expect(removeSpy).toHaveBeenCalledWith("pointerup", expect.any(Function))
      expect(removeSpy).toHaveBeenCalledWith("pointercancel", expect.any(Function))
    })
  })

  describe("setScreenSize", () => {
    it("calls onViewportChange callback", () => {
      const cam = createCamera({ onViewportChange })
      cam.setScreenSize(800, 600)

      expect(onViewportChange).toHaveBeenCalled()
      cam.destroy()
    })
  })

  describe("setMapSize", () => {
    it("calls onViewportChange callback", () => {
      const cam = createCamera({ onViewportChange })
      cam.setMapSize(2048, 2048)

      expect(onViewportChange).toHaveBeenCalled()
      cam.destroy()
    })
  })

  describe("resetView", () => {
    it("resets viewport to default and publishes", () => {
      const cam = createCamera({ onViewportChange })
      cam.setScreenSize(800, 600)
      onViewportChange.mockClear()

      cam.resetView()

      expect(onViewportChange).toHaveBeenCalledWith(DEFAULT_VIEWPORT)
      expect(cam.container.position.set).toHaveBeenCalledWith(0, 0)
      expect(cam.container.scale.set).toHaveBeenCalledWith(1)
      cam.destroy()
    })
  })

  describe("wheel zoom", () => {
    it("zooms in on scroll up (negative deltaY)", () => {
      const cam = createCamera({ onViewportChange })
      cam.setScreenSize(1024, 1024)
      cam.setMapSize(1024, 1024)
      onViewportChange.mockClear()

      canvas.dispatchEvent(
        new WheelEvent("wheel", {
          deltaY: -100,
          deltaMode: 0,
          clientX: 512,
          clientY: 512,
          cancelable: true,
        })
      )

      expect(onViewportChange).toHaveBeenCalled()
      const viewport = onViewportChange.mock.calls[0][0]
      expect(viewport.zoom).toBeGreaterThan(1)
      cam.destroy()
    })

    it("normalizes deltaMode 1 (line mode) for significant zoom", () => {
      const cam = createCamera({ onViewportChange })
      cam.setScreenSize(1024, 1024)
      cam.setMapSize(1024, 1024)
      onViewportChange.mockClear()

      // Line mode: deltaY=3 lines * 33 = 99 pixels equivalent
      canvas.dispatchEvent(
        new WheelEvent("wheel", {
          deltaY: -3,
          deltaMode: 1,
          clientX: 512,
          clientY: 512,
          cancelable: true,
        })
      )

      expect(onViewportChange).toHaveBeenCalled()
      const viewport = onViewportChange.mock.calls[0][0]
      // Without normalization this would be ~1.003, with normalization ~1.1
      expect(viewport.zoom).toBeGreaterThan(1.05)
      cam.destroy()
    })

    it("normalizes deltaMode 2 (page mode)", () => {
      const cam = createCamera({ onViewportChange })
      cam.setScreenSize(1024, 1024)
      cam.setMapSize(1024, 1024)
      onViewportChange.mockClear()

      canvas.dispatchEvent(
        new WheelEvent("wheel", {
          deltaY: -1,
          deltaMode: 2,
          clientX: 512,
          clientY: 512,
          cancelable: true,
        })
      )

      expect(onViewportChange).toHaveBeenCalled()
      const viewport = onViewportChange.mock.calls[0][0]
      expect(viewport.zoom).toBeGreaterThan(1.5)
      cam.destroy()
    })
  })

  describe("pointer drag", () => {
    it("pans on left-click drag past threshold", () => {
      const cam = createCamera({ onViewportChange })
      cam.setScreenSize(1024, 1024)
      cam.setMapSize(2048, 2048)
      onViewportChange.mockClear()

      canvas.dispatchEvent(new PointerEvent("pointerdown", { button: 0, clientX: 100, clientY: 100, pointerId: 1 }))
      canvas.dispatchEvent(new PointerEvent("pointermove", { clientX: 110, clientY: 115, pointerId: 1 }))

      expect(onViewportChange).toHaveBeenCalled()
      cam.destroy()
    })

    it("ignores non-left-click", () => {
      const cam = createCamera({ onViewportChange })
      cam.setScreenSize(1024, 1024)
      onViewportChange.mockClear()

      canvas.dispatchEvent(new PointerEvent("pointerdown", { button: 2, clientX: 100, clientY: 100, pointerId: 1 }))
      canvas.dispatchEvent(new PointerEvent("pointermove", { clientX: 150, clientY: 150, pointerId: 1 }))

      expect(onViewportChange).not.toHaveBeenCalled()
      cam.destroy()
    })

    it("does not pan below drag threshold", () => {
      const cam = createCamera({ onViewportChange })
      cam.setScreenSize(1024, 1024)
      onViewportChange.mockClear()

      canvas.dispatchEvent(new PointerEvent("pointerdown", { button: 0, clientX: 100, clientY: 100, pointerId: 1 }))
      canvas.dispatchEvent(new PointerEvent("pointermove", { clientX: 101, clientY: 100, pointerId: 1 }))

      expect(onViewportChange).not.toHaveBeenCalled()
      cam.destroy()
    })

    it("calls setPointerCapture on pointerdown", () => {
      const cam = createCamera()
      canvas.dispatchEvent(new PointerEvent("pointerdown", { button: 0, clientX: 100, clientY: 100, pointerId: 42 }))

      expect(canvas.setPointerCapture).toHaveBeenCalledWith(42)
      cam.destroy()
    })

    it("calls releasePointerCapture on pointerup", () => {
      const cam = createCamera()
      canvas.dispatchEvent(new PointerEvent("pointerdown", { button: 0, clientX: 100, clientY: 100, pointerId: 42 }))
      canvas.dispatchEvent(new PointerEvent("pointerup", { pointerId: 42 }))

      expect(canvas.releasePointerCapture).toHaveBeenCalledWith(42)
      cam.destroy()
    })

    it("calls releasePointerCapture on pointercancel", () => {
      const cam = createCamera()
      canvas.dispatchEvent(new PointerEvent("pointerdown", { button: 0, clientX: 100, clientY: 100, pointerId: 7 }))
      canvas.dispatchEvent(new PointerEvent("pointercancel", { pointerId: 7 }))

      expect(canvas.releasePointerCapture).toHaveBeenCalledWith(7)
      cam.destroy()
    })

    it("stops dragging after pointerup", () => {
      const cam = createCamera({ onViewportChange })
      cam.setScreenSize(1024, 1024)

      canvas.dispatchEvent(new PointerEvent("pointerdown", { button: 0, clientX: 100, clientY: 100, pointerId: 1 }))
      canvas.dispatchEvent(new PointerEvent("pointerup", { pointerId: 1 }))

      onViewportChange.mockClear()
      canvas.dispatchEvent(new PointerEvent("pointermove", { clientX: 200, clientY: 200, pointerId: 1 }))

      expect(onViewportChange).not.toHaveBeenCalled()
      cam.destroy()
    })
  })

  describe("onViewportChange callback", () => {
    it("is not required (no error without it)", () => {
      const cam = createCamera()
      cam.setScreenSize(800, 600)
      cam.resetView()
      cam.destroy()
    })

    it("receives viewport copy (not internal reference)", () => {
      const cam = createCamera({ onViewportChange })
      cam.setScreenSize(1024, 1024)

      const first = onViewportChange.mock.calls[0][0]
      cam.resetView()
      const second = onViewportChange.mock.calls[1][0]

      expect(first).not.toBe(second)
      cam.destroy()
    })
  })
})
