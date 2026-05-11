import { useCallback, useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { useDemo } from "@/hooks/use-demo"
import { useViewerKeyboard } from "@/hooks/use-viewer-keyboard"
import { useGameEvents } from "@/hooks/use-game-events"
import { useRounds } from "@/hooks/use-rounds"
import { useViewerStore } from "@/stores/viewer"
import { buildLanes } from "@/lib/timeline/build-lanes"
import { ViewerCanvas } from "@/components/viewer/viewer-canvas"
import { PlaybackDock } from "@/components/viewer/playback-dock"
import { MatchHeader } from "@/components/viewer/match-header"
import { Scoreboard } from "@/components/viewer/scoreboard"
import { TeamBars } from "@/components/viewer/team-bars"
import { KillLog } from "@/components/viewer/kill-log"
import { PlayerStatsPanel } from "@/components/viewer/player-stats-panel"

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
  const demoIdStr = id ?? null
  const { data: events } = useGameEvents(demoIdStr)
  const { data: rounds } = useRounds(demoIdStr)
  const handleNavigateEvent = useCallback(
    (direction: -1 | 1): number | null => {
      if (!events || !rounds?.length) return null
      const state = useViewerStore.getState()
      const currentTick = state.currentTick
      const filters = state.timelineFilters
      const selectedPlayerSteamId = state.selectedPlayerSteamId
      // Find the active round so we filter to its window — `,` / `.` should
      // only navigate inside the round the user is watching.
      let activeRound = rounds[0]
      for (let i = rounds.length - 1; i >= 0; i--) {
        if (currentTick >= rounds[i].start_tick) {
          activeRound = rounds[i]
          break
        }
      }
      const model = buildLanes({
        events,
        mistakes: [],
        round: activeRound,
        selectedPlayerSteamId,
        filters,
        laneWidthPx: 1000,
      })
      const ticks = [...model.topLane, ...model.bottomLane]
        .flatMap((c) => c.events.map((e) => e.tick))
        .sort((a, b) => a - b)
      if (ticks.length === 0) return null
      if (direction < 0) {
        for (let i = ticks.length - 1; i >= 0; i--) {
          if (ticks[i] < currentTick) return ticks[i]
        }
        return ticks[0]
      }
      for (let i = 0; i < ticks.length; i++) {
        if (ticks[i] > currentTick) return ticks[i]
      }
      return ticks[ticks.length - 1]
    },
    [events, rounds],
  )
  useViewerKeyboard({
    onToggleScoreboard: handleToggleScoreboard,
    onNavigateEvent: handleNavigateEvent,
  })

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
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError || !demo) {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-muted-foreground">
          Demo not found or failed to load.
        </p>
      </div>
    )
  }

  if (demo.status !== "ready") {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-muted-foreground">
          Demo is not ready for viewing (status: {demo.status}).
        </p>
      </div>
    )
  }

  return (
    <div
      className="relative h-full w-full overflow-hidden bg-black"
      data-testid="demo-viewer"
    >
      <ViewerCanvas />
      {/* Soft edge vignette so the chrome lifts off the radar without darkening the playable centre */}
      <div
        aria-hidden="true"
        className="hud-vignette pointer-events-none absolute inset-0 z-[5]"
      />
      <MatchHeader />
      <TeamBars />
      <KillLog />
      <PlaybackDock />
      <Scoreboard visible={scoreboardVisible} />
      <PlayerStatsPanel />
    </div>
  )
}
