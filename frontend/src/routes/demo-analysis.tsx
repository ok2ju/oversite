import { useEffect, useMemo } from "react"
import { useParams } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { useDemo } from "@/hooks/use-demo"
import { useScoreboard } from "@/hooks/use-scoreboard"
import { useViewerStore } from "@/stores/viewer"
import { AnalysisOverallGauge } from "@/components/viewer/analysis-overall-gauge"
import { CategoryCard } from "@/components/viewer/category-card"
import { RoundTradeBars } from "@/components/viewer/round-trade-bars"
import { DemoRouteTabs } from "@/components/viewer/demo-route-tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

// Standalone analysis page mounted at /demos/:id/analysis. Reuses the slice-5
// gauge and Trades card so the per-round bar chart, the overall score, and
// the category card stay rendered against the same store-backed
// (demoId, selectedPlayerSteamId) pair without component duplication.
//
// Owns its own player picker because the scoreboard overlay (where players
// are picked on the viewer page) is not mounted here, and the route's reset
// effect on the viewer page wipes the store on navigation. The picker
// auto-selects the first player from the scoreboard so the cards light up
// without an extra click — users can then switch via the dropdown.
export default function DemoAnalysisPage() {
  const { id } = useParams<{ id: string }>()
  const { data: demo, isLoading, isError } = useDemo(id)
  const initDemo = useViewerStore((s) => s.initDemo)
  const reset = useViewerStore((s) => s.reset)
  const demoId = useViewerStore((s) => s.demoId)
  const selectedSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const setSelectedPlayer = useViewerStore((s) => s.setSelectedPlayer)
  const { data: scoreboard } = useScoreboard(demoId)

  const orderedPlayers = useMemo(() => {
    if (!scoreboard) return []
    return [...scoreboard].sort((a, b) => {
      if (a.team_side !== b.team_side) return a.team_side === "CT" ? -1 : 1
      return a.player_name.localeCompare(b.player_name)
    })
  }, [scoreboard])

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
    if (selectedSteamId) return
    if (orderedPlayers.length === 0) return
    setSelectedPlayer(orderedPlayers[0].steam_id)
  }, [selectedSteamId, orderedPlayers, setSelectedPlayer])

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
    <main data-testid="demo-analysis" className="flex flex-col gap-6 p-6">
      <div className="flex items-center justify-between gap-4">
        <h1 className="text-xl font-semibold">Analysis</h1>
        <div className="flex items-center gap-3">
          {orderedPlayers.length > 0 ? (
            <Select
              value={selectedSteamId ?? undefined}
              onValueChange={(v) => setSelectedPlayer(v)}
            >
              <SelectTrigger
                data-testid="analysis-player-picker"
                className="h-9 w-56"
              >
                <SelectValue placeholder="Select player" />
              </SelectTrigger>
              <SelectContent>
                {orderedPlayers.map((p) => (
                  <SelectItem key={p.steam_id} value={p.steam_id}>
                    <span className="text-muted-foreground">
                      [{p.team_side}]
                    </span>{" "}
                    {p.player_name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : null}
          <DemoRouteTabs demoId={String(demo.id)} />
        </div>
      </div>
      {orderedPlayers.length === 0 ? (
        <p
          data-testid="analysis-no-players"
          className="text-sm text-muted-foreground"
        >
          No players found for this demo.
        </p>
      ) : null}
      <section className="rounded-md border border-border bg-card p-4">
        <AnalysisOverallGauge />
      </section>
      <section className="rounded-md border border-border bg-card p-4">
        <CategoryCard category="trade" />
      </section>
      <section className="rounded-md border border-border bg-card p-4">
        <CategoryCard category="aim" />
      </section>
      <section className="rounded-md border border-border bg-card p-4">
        <CategoryCard category="movement" />
      </section>
      <section className="rounded-md border border-border bg-card p-4">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          Round trade %
        </h2>
        <RoundTradeBars />
      </section>
    </main>
  )
}
