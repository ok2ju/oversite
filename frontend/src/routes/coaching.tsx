import { useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useDemos } from "@/hooks/use-demos"
import { useUniquePlayers } from "@/hooks/use-heatmap"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useCoachingReport } from "@/hooks/use-coaching-report"
import { MicroCard } from "@/components/coaching/micro-card"
import { ErrorsStrip } from "@/components/coaching/errors-strip"
import type { CoachingHabitRow } from "@/types/analysis"

const COACHING_HABIT_KEYS = [
  "counter_strafe",
  "reaction",
  "first_shot_acc",
  "shooting_in_motion",
  "crouch_before_shot",
  "flick_balance",
] as const

const DEFAULT_LOOKBACK = 10

// /coaching — micro-coaching landing surface. Aggregates the active player's
// last N demos into a card grid + errors strip; the page is player-scoped and
// persists the picker across visits via useViewerStore.selectedPlayerSteamId
// (we intentionally don't reset on unmount — see plan §3.2).
export default function CoachingPage() {
  const selectedSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const setSelectedPlayer = useViewerStore((s) => s.setSelectedPlayer)

  const { data: demoListData } = useDemos(1, 200)

  const allDemoIds = useMemo(() => {
    if (!demoListData?.data) return []
    return demoListData.data
      .filter((d) => d.status === "ready")
      .map((d) => d.id)
  }, [demoListData])

  const { data: players } = useUniquePlayers(allDemoIds)

  const orderedPlayers = useMemo(() => {
    if (!players) return []
    return [...players].sort((a, b) =>
      a.player_name.localeCompare(b.player_name),
    )
  }, [players])

  const selectedPlayer = useMemo(
    () => orderedPlayers.find((p) => p.steam_id === selectedSteamId),
    [orderedPlayers, selectedSteamId],
  )

  const { data: report, isLoading } = useCoachingReport(
    selectedSteamId,
    DEFAULT_LOOKBACK,
  )

  const habits = useMemo<CoachingHabitRow[]>(() => {
    if (!report?.habits) return []
    const byKey = new Map<string, CoachingHabitRow>(
      report.habits.map((h: CoachingHabitRow) => [h.key, h]),
    )
    return COACHING_HABIT_KEYS.map((key) => byKey.get(key)).filter(
      (row): row is CoachingHabitRow => row !== undefined,
    )
  }, [report])

  return (
    <main
      data-testid="coaching-page"
      className="mx-auto flex max-w-[1200px] flex-col gap-6 px-6 pb-16 pt-6"
    >
      <header className="flex flex-wrap items-end justify-between gap-x-6 gap-y-3 border-b border-[var(--divider)] pb-4">
        <div className="flex flex-col gap-1.5">
          <span className="font-mono text-[10px] uppercase tracking-[0.22em] text-[var(--text-faint)]">
            Coaching · Last {DEFAULT_LOOKBACK} demos
          </span>
          <h1
            className="text-[32px] font-bold leading-none tracking-tight text-[var(--text)]"
            style={{ fontFamily: "'Inter Tight', Inter, sans-serif" }}
          >
            Your gunfight micro
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
        {orderedPlayers.length > 0 ? (
          <Select
            value={selectedSteamId ?? undefined}
            onValueChange={(v) => setSelectedPlayer(v)}
          >
            <SelectTrigger
              data-testid="coaching-player-picker"
              className="h-9 w-56"
            >
              <SelectValue placeholder="Select player" />
            </SelectTrigger>
            <SelectContent>
              {orderedPlayers.map((p) => (
                <SelectItem key={p.steam_id} value={p.steam_id}>
                  {p.player_name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : null}
      </header>

      {!selectedSteamId ? (
        <p
          data-testid="coaching-empty"
          className="text-sm text-[var(--text-muted)]"
        >
          Pick a player above to see their coaching report.
        </p>
      ) : isLoading ? (
        <p
          data-testid="coaching-loading"
          className="text-sm text-[var(--text-muted)]"
        >
          Loading coaching report…
        </p>
      ) : !report || habits.length === 0 ? (
        <p
          data-testid="coaching-no-data"
          className="text-sm text-[var(--text-muted)]"
        >
          No analyzed demos found for this player yet.
        </p>
      ) : (
        <>
          <section
            data-testid="coaching-card-grid"
            className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
          >
            {habits.map((row) => (
              <MicroCard key={row.key} row={row} />
            ))}
          </section>
          <ErrorsStrip
            errors={report.errors}
            latestDemoId={report.latest_demo_id}
          />
        </>
      )}
    </main>
  )
}
