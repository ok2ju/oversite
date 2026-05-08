import { create } from "zustand"
import { subscribeWithSelector } from "zustand/middleware"
import type { TickBuffer } from "@/lib/pixi/tick-buffer"

// Hoists the active TickBuffer into a tiny store so that consumers outside
// the canvas (e.g. useLoadoutSnapshot) can share the single buffer that
// viewer-canvas already owns. The buffer is keyed by demoId so a stale
// buffer from a previous demo can never be observed by readers.
interface TickBufferState {
  demoId: string | null
  buffer: TickBuffer | null
  setBuffer: (demoId: string | null, buffer: TickBuffer | null) => void
}

export const useTickBufferStore = create<TickBufferState>()(
  subscribeWithSelector((set) => ({
    demoId: null,
    buffer: null,
    setBuffer: (demoId, buffer) => set({ demoId, buffer }),
  })),
)
