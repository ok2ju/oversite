import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useFaceitMatches } from "@/hooks/use-faceit-matches"
import type { FaceitMatch } from "@/types/faceit"

const CS2_MAPS = [
  "de_dust2",
  "de_mirage",
  "de_inferno",
  "de_nuke",
  "de_overpass",
  "de_vertigo",
  "de_anubis",
  "de_ancient",
]

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  })
}

export function MatchList() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [mapFilter, setMapFilter] = useState("")
  const [resultFilter, setResultFilter] = useState("")
  const perPage = 20

  const { data, isLoading } = useFaceitMatches(page, perPage, {
    map: mapFilter || undefined,
    result: resultFilter || undefined,
  })

  const matches = data?.data ?? []
  const total = data?.meta.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / perPage))

  function handleRowClick(match: FaceitMatch) {
    if (match.has_demo && match.demo_id) {
      navigate(`/demos/${match.demo_id}`)
    }
  }

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Card key={i} data-testid="match-skeleton">
            <CardContent className="p-4">
              <Skeleton className="h-5 w-full" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex gap-3">
        <div>
          <label htmlFor="map-filter" className="sr-only">
            Map
          </label>
          <select
            id="map-filter"
            aria-label="Map"
            value={mapFilter}
            onChange={(e) => {
              setMapFilter(e.target.value)
              setPage(1)
            }}
            className="rounded-md border bg-background px-3 py-2 text-sm"
          >
            <option value="">All Maps</option>
            {CS2_MAPS.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor="result-filter" className="sr-only">
            Result
          </label>
          <select
            id="result-filter"
            aria-label="Result"
            value={resultFilter}
            onChange={(e) => {
              setResultFilter(e.target.value)
              setPage(1)
            }}
            className="rounded-md border bg-background px-3 py-2 text-sm"
          >
            <option value="">All Results</option>
            <option value="W">Win</option>
            <option value="L">Loss</option>
          </select>
        </div>
      </div>

      <div className="space-y-2">
        {matches.map((match) => (
          <Card
            key={match.id}
            data-testid={`match-row-${match.id}`}
            className={
              match.has_demo ? "cursor-pointer hover:bg-accent/50" : ""
            }
            onClick={() => handleRowClick(match)}
          >
            <CardContent className="flex items-center justify-between p-4">
              <div className="flex items-center gap-4">
                <span className="w-24 font-medium">{match.map_name}</span>
                <span className="text-sm text-muted-foreground">
                  {match.score_team} - {match.score_opponent}
                </span>
                <Badge
                  variant={match.result === "W" ? "default" : "destructive"}
                  className={
                    match.result === "W"
                      ? "bg-green-600 hover:bg-green-600/80"
                      : ""
                  }
                >
                  {match.result === "W" ? "WIN" : "LOSS"}
                </Badge>
                <span
                  className={
                    match.elo_change === null
                      ? "text-sm text-muted-foreground"
                      : match.elo_change > 0
                        ? "text-sm font-medium text-green-500"
                        : "text-sm font-medium text-red-500"
                  }
                >
                  {match.elo_change === null
                    ? "--"
                    : match.elo_change > 0
                      ? `+${match.elo_change}`
                      : `${match.elo_change}`}
                </span>
              </div>

              <div className="flex items-center gap-4">
                {match.kills !== null && (
                  <span className="text-sm text-muted-foreground">
                    {match.kills}/{match.deaths}/{match.assists}
                  </span>
                )}
                <span className="text-sm text-muted-foreground">
                  {formatDate(match.played_at)}
                </span>
                {!match.has_demo && (
                  <Button
                    size="sm"
                    variant="outline"
                    data-testid="import-demo-btn"
                    onClick={(e) => e.stopPropagation()}
                  >
                    Import Demo
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="flex items-center justify-between">
        <span className="text-sm text-muted-foreground">
          Page {page} of {totalPages}
        </span>
        <div className="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            aria-label="Previous"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            Previous
          </Button>
          <Button
            size="sm"
            variant="outline"
            aria-label="Next"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  )
}
