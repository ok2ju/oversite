import { useEffect, useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { RefreshCw } from "lucide-react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { useFaceitMatches } from "@/hooks/use-faceit-matches"
import { useFaceitSync } from "@/hooks/use-faceit-sync"
import { MatchRow } from "@/components/dashboard/match-row"
import type { FaceitMatch } from "@/types/faceit"
import { cn } from "@/lib/utils"

const PER_PAGE = 8

export const MATCH_ROW_GRID =
  "grid grid-cols-[90px_120px_120px_80px_70px_70px_1fr_130px] items-center gap-x-4"

export function RecentMatches() {
  const navigate = useNavigate()
  const [limit, setLimit] = useState(PER_PAGE)
  const { data, isLoading, isError, isFetched } = useFaceitMatches(1, limit)
  const sync = useFaceitSync()

  const matches = data?.data ?? []
  const total = data?.meta.total ?? matches.length
  const canLoadMore = matches.length < total

  const autoSynced = useRef(false)
  useEffect(() => {
    if (autoSynced.current) return
    if (!isFetched || isError) return
    if (matches.length > 0) return
    if (sync.isPending) return
    autoSynced.current = true
    sync.mutate()
  }, [isFetched, isError, matches.length, sync])

  function handleClick(match: FaceitMatch) {
    if (match.has_demo && match.demo_id) {
      navigate(`/matches/${match.demo_id}`)
    }
  }

  return (
    <Card className="overflow-hidden border border-[var(--border)] bg-[var(--bg-elevated)] p-0">
      <div className="flex items-center justify-between border-b border-[var(--border)] px-5 py-4">
        <div>
          <div className="text-[15px] font-bold text-[var(--text)]">
            Recent Matches
          </div>
          <div className="text-[11.5px] text-[var(--text-muted)]">
            Last 7 days · Pulled from Faceit API
          </div>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => sync.mutate()}
          disabled={sync.isPending}
          className="gap-1.5"
        >
          <RefreshCw
            className={cn("h-3.5 w-3.5", sync.isPending && "animate-spin")}
          />
          Sync
        </Button>
      </div>

      {isLoading ? (
        <div className="divide-y divide-[var(--divider)]">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="px-5 py-[14px]"
              data-testid="match-row-skeleton"
            >
              <Skeleton className="h-9 w-full" />
            </div>
          ))}
        </div>
      ) : matches.length === 0 ? (
        <div className="px-5 py-10 text-center text-[12.5px] text-[var(--text-muted)]">
          {isError ? "Couldn't load matches" : "No recent matches"}
        </div>
      ) : (
        <>
          <div
            className={cn(
              MATCH_ROW_GRID,
              "border-b border-[var(--divider)] bg-[var(--bg)] px-5 py-2.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-[var(--text-subtle)]",
            )}
          >
            <div>Date</div>
            <div>Score</div>
            <div>KDA</div>
            <div>ADR</div>
            <div>K/D</div>
            <div>K/R</div>
            <div>Map</div>
            <div />
          </div>
          <div className="divide-y divide-[var(--divider)]">
            {matches.map((match) => (
              <MatchRow key={match.id} match={match} onClick={handleClick} />
            ))}
          </div>
        </>
      )}

      {canLoadMore && (
        <div className="border-t border-[var(--divider)] px-4 py-3 text-center">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setLimit((l) => l + PER_PAGE)}
          >
            Load more
          </Button>
        </div>
      )}
    </Card>
  )
}
