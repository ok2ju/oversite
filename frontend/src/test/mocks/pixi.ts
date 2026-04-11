import { vi } from "vitest"

export interface MockPixiApp {
  init: ReturnType<typeof vi.fn>
  destroy: ReturnType<typeof vi.fn>
  canvas: HTMLCanvasElement
  stage: {
    addChild: ReturnType<typeof vi.fn>
    removeChild: ReturnType<typeof vi.fn>
    children: unknown[]
  }
  ticker: {
    start: ReturnType<typeof vi.fn>
    stop: ReturnType<typeof vi.fn>
    speed: number
  }
}

export function createMockPixiApp(): MockPixiApp {
  const canvas = document.createElement("canvas")
  const children: unknown[] = []

  return {
    init: vi.fn().mockResolvedValue(undefined),
    destroy: vi.fn(),
    canvas,
    stage: {
      addChild: vi.fn((child: unknown) => {
        children.push(child)
      }),
      removeChild: vi.fn((child: unknown) => {
        const idx = children.indexOf(child)
        if (idx >= 0) children.splice(idx, 1)
      }),
      children,
    },
    ticker: {
      start: vi.fn(),
      stop: vi.fn(),
      speed: 1,
    },
  }
}

export interface MockSprite {
  texture: unknown
  width: number
  height: number
  destroy: ReturnType<typeof vi.fn>
}

export function createMockSprite(options?: { texture?: unknown }): MockSprite {
  return {
    texture: options?.texture ?? null,
    width: 0,
    height: 0,
    destroy: vi.fn(),
  }
}

export interface MockTexture {
  width: number
  height: number
}

export function createMockTexture(width = 1024, height = 1024): MockTexture {
  return { width, height }
}

export function createMockAssets() {
  const mockTexture = createMockTexture()
  return {
    load: vi.fn().mockResolvedValue(mockTexture),
    _mockTexture: mockTexture,
  }
}

export interface MockGraphics {
  clear: ReturnType<typeof vi.fn>
  circle: ReturnType<typeof vi.fn>
  rect: ReturnType<typeof vi.fn>
  moveTo: ReturnType<typeof vi.fn>
  lineTo: ReturnType<typeof vi.fn>
  fill: ReturnType<typeof vi.fn>
  stroke: ReturnType<typeof vi.fn>
  destroy: ReturnType<typeof vi.fn>
  removeFromParent: ReturnType<typeof vi.fn>
  poly: ReturnType<typeof vi.fn>
}

export function createMockGraphics(): MockGraphics {
  const g: MockGraphics = {
    clear: vi.fn(),
    circle: vi.fn(),
    rect: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    fill: vi.fn(),
    stroke: vi.fn(),
    destroy: vi.fn(),
    removeFromParent: vi.fn(),
    poly: vi.fn(),
  }
  g.clear.mockReturnValue(g)
  g.circle.mockReturnValue(g)
  g.rect.mockReturnValue(g)
  g.moveTo.mockReturnValue(g)
  g.lineTo.mockReturnValue(g)
  g.fill.mockReturnValue(g)
  g.stroke.mockReturnValue(g)
  g.poly.mockReturnValue(g)
  return g
}

export interface MockText {
  text: string
  style: Record<string, unknown>
  position: { set: ReturnType<typeof vi.fn> }
  destroy: ReturnType<typeof vi.fn>
  removeFromParent: ReturnType<typeof vi.fn>
}

export function createMockText(): MockText {
  return {
    text: "",
    style: {},
    position: { set: vi.fn() },
    destroy: vi.fn(),
    removeFromParent: vi.fn(),
  }
}

export interface MockViewerApp {
  initialized: boolean
  ticker: {
    start: ReturnType<typeof vi.fn>
    stop: ReturnType<typeof vi.fn>
    add: ReturnType<typeof vi.fn>
    remove: ReturnType<typeof vi.fn>
    speed: number
  }
  destroy: ReturnType<typeof vi.fn>
  addLayer: ReturnType<typeof vi.fn>
  stage: { addChild: ReturnType<typeof vi.fn> }
  canvas: HTMLCanvasElement
}

export function createMockViewerApp(): MockViewerApp {
  return {
    initialized: true,
    ticker: {
      start: vi.fn(),
      stop: vi.fn(),
      add: vi.fn(),
      remove: vi.fn(),
      speed: 1,
    },
    destroy: vi.fn(),
    addLayer: vi.fn().mockReturnValue({ addChild: vi.fn(), removeChild: vi.fn() }),
    stage: { addChild: vi.fn() },
    canvas: document.createElement("canvas"),
  }
}
