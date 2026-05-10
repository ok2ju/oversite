import { useEffect, useMemo } from "react"
import { useParams } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { useDemo } from "@/hooks/use-demo"
import { useScoreboard } from "@/hooks/use-scoreboard"
import { useRounds } from "@/hooks/use-rounds"
import { useViewerStore } from "@/stores/viewer"
import { DemoRouteTabs } from "@/components/viewer/demo-route-tabs"
import { VerdictHero } from "@/components/analysis/verdict-hero"
import { HabitChecklist } from "@/components/analysis/habit-checklist"
import { MatchTimeline } from "@/components/analysis/match-timeline"
import { MistakesFeed } from "@/components/analysis/mistakes-feed"
import { NextDrillCard } from "@/components/analysis/next-drill-card"
import { HeadToHead } from "@/components/analysis/head-to-head"
import { EconomyStrip } from "@/components/analysis/economy-strip"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

// Standalone analysis page mounted at /demos/:id/analysis. Verdict-first
// layout: a hero with the overall score + per-category bars sets the story,
// the match timeline shows the round-by-round arc, and the mistakes feed is
// the action surface (one click → seek the viewer to the offending tick).
// Owns its own player picker because the scoreboard overlay (where players
// are picked on the viewer page) is not mounted here, and the route's reset
// effect on the viewer page wipes the store on navigation. The picker
// auto-selects the first player from the scoreboard so the cards light up
// without an extra click.
export default function DemoAnalysisPage() {
  const { id } = useParams<{ id: string }>()
  const { data: demo, isLoading, isError } = useDemo(id)
  const initDemo = useViewerStore((s) => s.initDemo)
  const reset = useViewerStore((s) => s.reset)
  const demoId = useViewerStore((s) => s.demoId)
  const selectedSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const setSelectedPlayer = useViewerStore((s) => s.setSelectedPlayer)
  const { data: scoreboard } = useScoreboard(demoId)
  const { data: rounds } = useRounds(demoId)

  const orderedPlayers = useMemo(() => {
    if (!scoreboard) return []
    return [...scoreboard].sort((a, b) => {
      if (a.team_side !== b.team_side) return a.team_side === "CT" ? -1 : 1
      return a.player_name.localeCompare(b.player_name)
    })
  }, [scoreboard])

  const selectedPlayer = useMemo(
    () => orderedPlayers.find((p) => p.steam_id === selectedSteamId),
    [orderedPlayers, selectedSteamId],
  )

  const finalScore = useMemo(() => {
    if (!rounds || rounds.length === 0) return null
    const last = rounds[rounds.length - 1]
    return { ct: last.ct_score, t: last.t_score }
  }, [rounds])

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
        <Loader2 className="h-8 w-8 animate-spin text-[var(--text-muted)]" />
      </div>
    )
  }

  if (isError || !demo) {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-[var(--text-muted)]">
          Demo not found or failed to load.
        </p>
      </div>
    )
  }

  if (demo.status !== "ready") {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-[var(--text-muted)]">
          Demo is not ready for viewing (status: {demo.status}).
        </p>
      </div>
    )
  }

  const matchDate = demo.match_date ? formatDate(demo.match_date) : null

  return (
    <main
      data-testid="demo-analysis"
      className="mx-auto flex max-w-[1200px] flex-col gap-5 px-6 pb-16 pt-6"
    >
      {/* Page header */}
      <header className="flex flex-wrap items-end justify-between gap-x-6 gap-y-3 border-b border-[var(--divider)] pb-4">
        <div className="flex flex-col gap-1.5">
          <span className="font-mono text-[10px] uppercase tracking-[0.22em] text-[var(--text-faint)]">
            Debrief / {demo.map_name.replace(/^de_/, "").toUpperCase()}
            {finalScore ? `  ·  ${finalScore.ct}–${finalScore.t}` : null}
            {matchDate ? `  ·  ${matchDate}` : null}
          </span>
          <h1
            className="text-[32px] font-bold leading-none tracking-tight text-[var(--text)]"
            style={{ fontFamily: "'Inter Tight', Inter, sans-serif" }}
          >
            Analysis
            {selectedPlayer ? (
              <span className="ml-3 font-medium text-[var(--text-muted)]">
                ·{" "}
                <span className="text-[var(--text)]">
                  {selectedPlayer.player_name}
                </span>
              </span>
            ) : null}
          </h1>
        </div>
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
                    <span
                      className={
                        p.team_side === "CT"
                          ? "text-[#5db1ff]"
                          : "text-[var(--accent)]"
                      }
                    >
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
      </header>

      {orderedPlayers.length === 0 ? (
        <p
          data-testid="analysis-no-players"
          className="text-sm text-[var(--text-muted)]"
        >
          No players found for this demo.
        </p>
      ) : (
        <>
          <VerdictHero />
          <HabitChecklist />
          <NextDrillCard />
          <MatchTimeline />
          <MistakesFeed />
          <div className="grid gap-5 lg:grid-cols-[3fr_2fr]">
            <HeadToHead demoId={String(demo.id)} />
            <EconomyStrip />
          </div>
        </>
      )}
    </main>
  )
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  })
}
