import { useEffect, useRef, useCallback } from "react"
import { useViewerStore } from "@/stores/viewer"
import { TickBuffer } from "@/lib/pixi/tick-buffer"
import type { TickData } from "@/types/demo"

export function useTickData() {
  const demoId = useViewerStore((s) => s.demoId)
  const bufferRef = useRef<TickBuffer | null>(null)

  useEffect(() => {
    if (!demoId) {
      bufferRef.current = null
      return
    }

    const buffer = new TickBuffer(demoId)
    bufferRef.current = buffer

    return () => {
      buffer.dispose()
      bufferRef.current = null
    }
  }, [demoId])

  const getTickData = useCallback((tick: number): TickData[] | null => {
    return bufferRef.current?.getTickData(tick) ?? null
  }, [])

  const seek = useCallback((tick: number): void => {
    bufferRef.current?.seek(tick)
  }, [])

  return { getTickData, seek }
}
