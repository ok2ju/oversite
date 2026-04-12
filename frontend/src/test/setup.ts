import "@testing-library/jest-dom/vitest"
import { cleanup } from "@testing-library/react"
import { afterAll, afterEach, beforeAll } from "vitest"
import { server } from "./msw/server"

// jsdom doesn't implement ResizeObserver — stub it
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.ResizeObserver =
  ResizeObserverStub as unknown as typeof ResizeObserver

// jsdom doesn't implement matchMedia — stub it for next-themes
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
})

beforeAll(() => {
  server.listen({ onUnhandledRequest: "error" })
})

afterEach(() => {
  server.resetHandlers()
  cleanup()
})

afterAll(() => {
  server.close()
})
