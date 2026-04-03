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
