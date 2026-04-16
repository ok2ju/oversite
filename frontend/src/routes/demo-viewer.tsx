import { useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { useDemo } from "@/hooks/use-demo"
import { useViewerStore } from "@/stores/viewer"
import { ViewerCanvas } from "@/components/viewer/viewer-canvas"
import { PlaybackControls } from "@/components/viewer/playback-controls"
import { MiniMap } from "@/components/viewer/mini-map"
import { RoundSelector } from "@/components/viewer/round-selector"
import { Scoreboard } from "@/components/viewer/scoreboard"

export default function DemoViewerPage() {
  const { id } = useParams<{ id: string }>()
  const { data: demo, isLoading, isError } = useDemo(id)
  const setDemoId = useViewerStore((s) => s.setDemoId)
  const setMapName = useViewerStore((s) => s.setMapName)
  const setTotalTicks = useViewerStore((s) => s.setTotalTicks)
  const reset = useViewerStore((s) => s.reset)
  const [scoreboardVisible, setScoreboardVisible] = useState(false)

  useEffect(() => {
    if (!demo) return
    setDemoId(String(demo.id))
    setMapName(demo.map_name)
    setTotalTicks(demo.total_ticks)
  }, [demo, setDemoId, setMapName, setTotalTicks])

  useEffect(() => {
    return () => {
      reset()
    }
  }, [reset])

  if (isLoading) {
    return (
      <div className="-m-6 flex h-[calc(100%+3rem)] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError || !demo) {
    return (
      <div className="-m-6 flex h-[calc(100%+3rem)] items-center justify-center">
        <p className="text-muted-foreground">
          Demo not found or failed to load.
        </p>
      </div>
    )
  }

  if (demo.status !== "ready") {
    return (
      <div className="-m-6 flex h-[calc(100%+3rem)] items-center justify-center">
        <p className="text-muted-foreground">
          Demo is not ready for viewing (status: {demo.status}).
        </p>
      </div>
    )
  }

  return (
    <div
      className="relative -m-6 h-[calc(100%+3rem)] overflow-hidden bg-black"
      data-testid="demo-viewer"
    >
      <ViewerCanvas />
      <PlaybackControls />
      <MiniMap />
      <RoundSelector />
      <Scoreboard visible={scoreboardVisible} />
    </div>
  )
}
