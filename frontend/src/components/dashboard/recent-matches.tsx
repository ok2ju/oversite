import { useState } from "react"
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

export function RecentMatches() {
  const navigate = useNavigate()
  const [limit, setLimit] = useState(PER_PAGE)
  const { data, isLoading, isError } = useFaceitMatches(1, limit)
  const sync = useFaceitSync()

  const matches = data?.data ?? []
  const total = data?.meta.total ?? matches.length
  const canLoadMore = matches.length < total

  function handleClick(match: FaceitMatch) {
    if (match.has_demo && match.demo_id) {
      navigate(`/matches/${match.demo_id}`)
    }
  }

  return (
    <Card className="overflow-hidden border border-[var(--border)] bg-[var(--bg-elevated)] p-0">
      <div className="flex items-center justify-between border-b border-[var(--border)] px-5 py-3.5">
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
              className="px-4 py-[14px]"
              data-testid="match-row-skeleton"
            >
              <Skeleton className="h-8 w-full" />
            </div>
          ))}
        </div>
      ) : matches.length === 0 ? (
        <div className="px-5 py-8 text-center text-[12.5px] text-[var(--text-muted)]">
          {isError ? "Couldn't load matches" : "No recent matches"}
        </div>
      ) : (
        <div className="divide-y divide-[var(--divider)]">
          {matches.map((match) => (
            <MatchRow key={match.id} match={match} onClick={handleClick} />
          ))}
        </div>
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
