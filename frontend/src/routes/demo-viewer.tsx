import { useCallback, useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { useDemo } from "@/hooks/use-demo"
import { useViewerKeyboard } from "@/hooks/use-viewer-keyboard"
import { useViewerStore } from "@/stores/viewer"
import { ViewerCanvas } from "@/components/viewer/viewer-canvas"
import { PlaybackControls } from "@/components/viewer/playback-controls"
import { MatchHeader } from "@/components/viewer/match-header"
import { RoundSelector } from "@/components/viewer/round-selector"
import { Scoreboard } from "@/components/viewer/scoreboard"
import { TeamBars } from "@/components/viewer/team-bars"

export default function DemoViewerPage() {
  const { id } = useParams<{ id: string }>()
  const { data: demo, isLoading, isError } = useDemo(id)
  const initDemo = useViewerStore((s) => s.initDemo)
  const reset = useViewerStore((s) => s.reset)
  const [scoreboardVisible, setScoreboardVisible] = useState(false)
  const handleToggleScoreboard = useCallback(
    () => setScoreboardVisible((v) => !v),
    [],
  )
  useViewerKeyboard({ onToggleScoreboard: handleToggleScoreboard })

  useEffect(() => {
    if (!demo) return
    initDemo({
      id: String(demo.id),
      mapName: demo.map_name,
      totalTicks: demo.total_ticks,
      tickRate: demo.tick_rate,
    })
  }, [demo, initDemo])

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
      <MatchHeader />
      <TeamBars />
      <PlaybackControls />
      <RoundSelector />
      <Scoreboard visible={scoreboardVisible} />
    </div>
  )
}
