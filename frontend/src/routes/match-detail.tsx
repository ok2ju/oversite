import { useMemo } from "react"
import { useParams } from "react-router-dom"
import { useQuery } from "@tanstack/react-query"
import { GetDemoByID } from "@wailsjs/go/main/App"
import { MatchToolbar } from "@/components/match/toolbar"
import { MatchHero } from "@/components/match/match-hero"
import { RoundTimeline } from "@/components/match/round-timeline"
import { ScoreboardTable } from "@/components/match/scoreboard-table"
import { Skeleton } from "@/components/ui/skeleton"
import { useRounds } from "@/hooks/use-rounds"
import { useScoreboard } from "@/hooks/use-scoreboard"
import type { Demo } from "@/types/demo"

function useDemo(id: string | undefined) {
  return useQuery({
    queryKey: ["demo", id],
    queryFn: () => GetDemoByID(id!) as Promise<Demo>,
    enabled: !!id,
  })
}

export default function MatchDetailPage() {
  const { id } = useParams<{ id: string }>()
  const demoQuery = useDemo(id)
  const roundsQuery = useRounds(id ?? null)
  const scoreboardQuery = useScoreboard(id ?? null)

  const demo = demoQuery.data
  const rounds = useMemo(() => roundsQuery.data ?? [], [roundsQuery.data])
  const scoreboard = useMemo(
    () => scoreboardQuery.data ?? [],
    [scoreboardQuery.data],
  )

  const summary = useMemo(() => {
    const ct = scoreboard.filter((p) => p.team_side === "CT")
    const t = scoreboard.filter((p) => p.team_side !== "CT")
    const lastRound = rounds[rounds.length - 1]
    return {
      leftScore: lastRound?.ct_score ?? 0,
      rightScore: lastRound?.t_score ?? 0,
      leftCount: ct.length,
      rightCount: t.length,
    }
  }, [scoreboard, rounds])

  return (
    <div className="flex flex-col gap-[18px]">
      <MatchToolbar
        matchId={id ?? ""}
        demoId={demo?.id ?? null}
        demoStatus={demo?.status ?? null}
      />

      {demoQuery.isLoading ? (
        <Skeleton className="h-[140px] w-full" />
      ) : (
        <MatchHero
          mapName={demo?.map_name ?? ""}
          mode="5v5"
          roundCount={rounds.length}
          durationSecs={demo?.duration_secs}
          left={{
            name: "Your team",
            score: summary.leftScore,
            playerCount: summary.leftCount,
            premade: true,
          }}
          right={{
            name: "Enemy team",
            score: summary.rightScore,
            playerCount: summary.rightCount,
          }}
        />
      )}

      {roundsQuery.isLoading ? (
        <Skeleton className="h-[120px] w-full" />
      ) : rounds.length > 0 ? (
        <RoundTimeline rounds={rounds} />
      ) : null}

      {scoreboardQuery.isLoading ? (
        <Skeleton className="h-[200px] w-full" />
      ) : (
        <ScoreboardTable entries={scoreboard} />
      )}
    </div>
  )
}
