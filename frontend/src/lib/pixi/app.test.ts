import { describe, it, expect, vi, beforeEach } from "vitest"
import { createMockPixiApp, type MockPixiApp } from "@/test/mocks/pixi"

let mockApp: MockPixiApp

vi.mock("pixi.js", () => {
  return {
    Application: vi.fn().mockImplementation(function () {
      return mockApp
    }),
    Container: vi.fn().mockImplementation(function () {
      return { label: "" }
    }),
  }
})

import { createViewerApp, ViewerApp } from "./app"

describe("ViewerApp", () => {
  let container: HTMLDivElement

  beforeEach(() => {
    mockApp = createMockPixiApp()
    container = document.createElement("div")
    document.body.appendChild(container)
  })

  describe("createViewerApp", () => {
    it("calls Application.init with correct options", async () => {
      await createViewerApp({ container })

      expect(mockApp.init).toHaveBeenCalledWith({
        background: 0x1a1a2e,
        resizeTo: container,
        antialias: true,
        resolution: window.devicePixelRatio,
        autoDensity: true,
      })
    })

    it("calls Application.init with custom background", async () => {
      await createViewerApp({ container, background: 0xff0000 })

      expect(mockApp.init).toHaveBeenCalledWith(
        expect.objectContaining({ background: 0xff0000 })
      )
    })

    it("appends canvas to container", async () => {
      await createViewerApp({ container })

      expect(container.contains(mockApp.canvas)).toBe(true)
    })

    it("returns initialized ViewerApp", async () => {
      const app = await createViewerApp({ container })

      expect(app).toBeInstanceOf(ViewerApp)
      expect(app.initialized).toBe(true)
    })
  })

  describe("layer management", () => {
    let app: ViewerApp

    beforeEach(async () => {
      app = await createViewerApp({ container })
    })

    it("addLayer adds named container to stage", () => {
      const layer = app.addLayer("map")

      expect(mockApp.stage.addChild).toHaveBeenCalledWith(layer)
      expect(layer.label).toBe("map")
    })

    it("addLayer throws on duplicate name", () => {
      app.addLayer("map")

      expect(() => app.addLayer("map")).toThrow('Layer "map" already exists')
    })

    it("getLayer returns registered layer", () => {
      const layer = app.addLayer("players")

      expect(app.getLayer("players")).toBe(layer)
    })

    it("getLayer returns undefined for unknown layer", () => {
      expect(app.getLayer("nonexistent")).toBeUndefined()
    })

    it("removeLayer removes from stage and registry", () => {
      const layer = app.addLayer("events")
      app.removeLayer("events")

      expect(mockApp.stage.removeChild).toHaveBeenCalledWith(layer)
      expect(app.getLayer("events")).toBeUndefined()
    })

    it("removeLayer is no-op for unknown layer", () => {
      app.removeLayer("nonexistent")

      expect(mockApp.stage.removeChild).not.toHaveBeenCalled()
    })
  })

  describe("destroy", () => {
    it("calls PixiJS destroy with correct arguments and clears state", async () => {
      const app = await createViewerApp({ container })
      app.addLayer("map")

      app.destroy()

      expect(mockApp.destroy).toHaveBeenCalledWith(
        { removeView: true },
        { children: true, texture: true, textureSource: true }
      )
      expect(app.initialized).toBe(false)
      expect(app.getLayer("map")).toBeUndefined()
    })
  })

  describe("accessors", () => {
    it("exposes stage, canvas, and ticker", async () => {
      const app = await createViewerApp({ container })

      expect(app.stage).toBe(mockApp.stage)
      expect(app.canvas).toBe(mockApp.canvas)
      expect(app.ticker).toBe(mockApp.ticker)
    })
  })
})
