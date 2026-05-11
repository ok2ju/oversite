import { useMemo } from "react"
import { X } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { usePlayerStats } from "@/hooks/use-player-stats"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Progress } from "@/components/ui/progress"
import { PlayerLiveHud } from "@/components/viewer/player-live-hud"
import { AnalysisOverallGauge } from "@/components/viewer/analysis-overall-gauge"
import type { PlayerMatchStats, PlayerRoundDetail } from "@/types/player-stats"
import type { Round } from "@/types/round"

const PANEL_WIDTH = 420

const sideColor = (side: string) =>
  side === "CT"
    ? "text-sky-400"
    : side === "T"
      ? "text-amber-400"
      : "text-white"
const sideBg = (side: string) =>
  side === "CT"
    ? "bg-sky-400/30"
    : side === "T"
      ? "bg-amber-400/30"
      : "bg-white/10"

function getActiveRoundIndex(rounds: Round[], currentTick: number): number {
  for (let i = rounds.length - 1; i >= 0; i--) {
    if (currentTick >= rounds[i].start_tick) return i
  }
  return 0
}

function MatchSummary({ stats }: { stats: PlayerMatchStats }) {
  return (
    <div className="grid grid-cols-3 gap-2 text-xs">
      <Stat
        label="K / D / A"
        value={`${stats.kills} / ${stats.deaths} / ${stats.assists}`}
      />
      <Stat label="ADR" value={`${Math.round(stats.adr)}`} />
      <Stat label="HS%" value={`${Math.round(stats.hs_percent)}%`} />
      <Stat
        label="Opening W/L"
        value={`${stats.opening_wins} / ${stats.opening_losses}`}
      />
      <Stat label="Clutches" value={`${stats.clutch_kills}`} />
      <Stat label="Trades" value={`${stats.trade_kills}`} />
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded border border-white/10 bg-white/5 p-2">
      <div className="text-[10px] uppercase tracking-wide text-white/60">
        {label}
      </div>
      <div className="tabular-nums text-white">{value}</div>
    </div>
  )
}

function RoundStrip({
  rounds,
  currentRound,
  onSelectRound,
}: {
  rounds: PlayerRoundDetail[]
  currentRound: number
  onSelectRound: (round: number) => void
}) {
  const maxDamage = Math.max(1, ...rounds.map((r) => r.damage))
  return (
    <div data-testid="player-stats-round-strip" className="space-y-1">
      <div className="text-[10px] uppercase tracking-wide text-white/60">
        Per round
      </div>
      <div className="flex flex-wrap gap-1">
        {rounds.map((r) => {
          const active = r.round_number === currentRound
          const intensity = Math.round((r.damage / maxDamage) * 100)
          return (
            <button
              key={r.round_number}
              type="button"
              data-testid={`player-stats-round-cell-${r.round_number}`}
              data-active={active ? "true" : undefined}
              onClick={() => onSelectRound(r.round_number)}
              className={`flex h-12 w-7 flex-col items-center justify-end rounded border text-[10px] tabular-nums ${
                active
                  ? "border-white/80 bg-white/15 text-white"
                  : "border-white/10 bg-white/5 text-white/70 hover:bg-white/10"
              }`}
              title={`Round ${r.round_number}: ${r.kills}K / ${r.deaths}D, ${r.damage} dmg`}
            >
              <div
                className={`w-full ${sideBg(r.team_side)}`}
                style={{ height: `${intensity}%` }}
              />
              <span className="mt-0.5">{r.round_number}</span>
            </button>
          )
        })}
      </div>
    </div>
  )
}

// MovementSparkline plots per-round distance traveled as a thin bar strip,
// scaled to the round with the highest distance. It complements the damage
// strip above without taking a second tab.
function MovementSparkline({ rounds }: { rounds: PlayerRoundDetail[] }) {
  const maxDistance = Math.max(1, ...rounds.map((r) => r.distance_units))
  return (
    <div data-testid="player-stats-movement-sparkline" className="space-y-1">
      <div className="text-[10px] uppercase tracking-wide text-white/60">
        Distance per round
      </div>
      <div className="flex h-8 items-end gap-[2px]">
        {rounds.map((r) => {
          const intensity = Math.round((r.distance_units / maxDistance) * 100)
          return (
            <div
              key={r.round_number}
              data-testid={`player-stats-distance-bar-${r.round_number}`}
              title={`Round ${r.round_number}: ${r.distance_units.toLocaleString()} units`}
              className="flex-1 rounded-sm bg-white/15"
              style={{ height: `${Math.max(intensity, 2)}%` }}
            />
          )
        })}
      </div>
    </div>
  )
}

function RoundDetailCard({
  round,
  movement,
  utility,
}: {
  round: PlayerRoundDetail | null
  movement: PlayerMatchStats["movement"] | undefined
  utility: PlayerMatchStats["utility"] | undefined
}) {
  if (!round) {
    return (
      <div
        data-testid="player-stats-round-empty"
        className="rounded border border-white/10 bg-white/5 p-3 text-sm text-white/60"
      >
        No data for this round.
      </div>
    )
  }
  const ttfc =
    round.time_to_first_contact_sec === null
      ? "—"
      : `${round.time_to_first_contact_sec.toFixed(1)}s`
  return (
    <div
      data-testid="player-stats-round-detail"
      className="space-y-3 rounded border border-white/10 bg-white/5 p-3"
    >
      <div className="flex items-baseline justify-between">
        <span className="text-xs uppercase tracking-wide text-white/60">
          Round {round.round_number}
        </span>
        <span className={`text-xs ${sideColor(round.team_side)}`}>
          {round.team_side}
        </span>
      </div>
      <div className="grid grid-cols-3 gap-2 text-xs">
        <Stat
          label="K / D / A"
          value={`${round.kills} / ${round.deaths} / ${round.assists}`}
        />
        <Stat label="Damage" value={`${round.damage}`} />
        <Stat label="HS" value={`${round.hs_kills}`} />
      </div>
      <div className="grid grid-cols-3 gap-2 text-xs">
        <Stat label="First K" value={round.first_kill ? "yes" : "—"} />
        <Stat label="First D" value={round.first_death ? "yes" : "—"} />
        <Stat label="Trade" value={round.trade_kill ? "yes" : "—"} />
      </div>
      <div className="grid grid-cols-2 gap-2 text-xs">
        <Stat
          label="Loadout"
          value={`$${round.loadout_value.toLocaleString()}`}
        />
        <Stat label="Clutch K" value={`${round.clutch_kills}`} />
      </div>
      <div className="grid grid-cols-3 gap-2 text-xs">
        <Stat
          label="Distance"
          value={`${round.distance_units.toLocaleString()}u`}
        />
        <Stat
          label="Alive"
          value={`${round.alive_duration_secs.toFixed(1)}s`}
        />
        <Stat label="1st contact" value={ttfc} />
      </div>
      {movement ? <MovementCard movement={movement} /> : null}
      {utility ? <UtilityCard utility={utility} /> : null}
    </div>
  )
}

// MovementCard is the match-level movement profile shown alongside the
// per-round breakdown. The strafe metric is sampled at 16 Hz (parser default)
// and is therefore a trend indicator rather than a competitive stat — the
// title attribute carries the inline tooltip until shadcn Tooltip is added.
function MovementCard({
  movement,
}: {
  movement: PlayerMatchStats["movement"]
}) {
  const stationaryPct = Math.round(movement.stationary_ratio * 100)
  const walkingPct = Math.round(movement.walking_ratio * 100)
  const runningPct = Math.round(movement.running_ratio * 100)
  return (
    <div
      data-testid="player-stats-movement-card"
      className="space-y-2 rounded border border-white/10 bg-black/30 p-2 text-xs"
    >
      <div className="flex items-baseline justify-between">
        <span className="text-[10px] uppercase tracking-wide text-white/60">
          Movement (match)
        </span>
        <span className="tabular-nums text-white/70">
          {movement.distance_units.toLocaleString()}u total
        </span>
      </div>
      <MovementBar label="Stationary" value={stationaryPct} />
      <MovementBar label="Walking" value={walkingPct} />
      <MovementBar label="Running" value={runningPct} />
      <MovementBar
        label="Strafing"
        value={Math.round(movement.strafe_percent)}
        helpText="Approximate; sampled at 16 Hz (parser default). Trend signal, not a competitive metric."
        testId="player-stats-strafe-bar"
      />
      <div className="grid grid-cols-2 gap-2 pt-1">
        <Stat
          label="Avg speed"
          value={`${Math.round(movement.avg_speed_ups)} u/s`}
        />
        <Stat
          label="Max speed"
          value={`${Math.round(movement.max_speed_ups)} u/s`}
        />
      </div>
    </div>
  )
}

function MovementBar({
  label,
  value,
  helpText,
  testId,
}: {
  label: string
  value: number
  helpText?: string
  testId?: string
}) {
  return (
    <div className="space-y-0.5">
      <div className="flex items-baseline justify-between">
        <span
          className={`text-[10px] uppercase tracking-wide text-white/60 ${
            helpText ? "cursor-help underline decoration-dotted" : ""
          }`}
          title={helpText}
        >
          {label}
        </span>
        <span className="tabular-nums text-white/80">{value}%</span>
      </div>
      <Progress
        data-testid={testId}
        value={Math.max(0, Math.min(100, value))}
        className="h-1.5 bg-white/10"
      />
    </div>
  )
}

function DetailLists({ stats }: { stats: PlayerMatchStats }) {
  return (
    <div className="space-y-3 text-xs">
      <Section title="Damage by weapon">
        {stats.damage_by_weapon.length === 0 ? (
          <p className="text-white/60">No damage recorded.</p>
        ) : (
          <ul className="space-y-1">
            {stats.damage_by_weapon.map((row) => (
              <li
                key={row.weapon}
                className="flex items-center justify-between gap-2"
                data-testid={`player-stats-damage-weapon-${row.weapon}`}
              >
                <span className="truncate text-white/80">{row.weapon}</span>
                <span className="tabular-nums text-white">{row.damage}</span>
              </li>
            ))}
          </ul>
        )}
      </Section>
      <Section title="Damage by opponent">
        {stats.damage_by_opponent.length === 0 ? (
          <p className="text-white/60">No damage recorded.</p>
        ) : (
          <ul className="space-y-1">
            {stats.damage_by_opponent.map((row) => (
              <li
                key={row.steam_id}
                className="flex items-center justify-between gap-2"
                data-testid={`player-stats-damage-opponent-${row.steam_id}`}
              >
                <span className="flex min-w-0 items-center gap-1">
                  <span className={`text-[10px] ${sideColor(row.team_side)}`}>
                    {row.team_side || "?"}
                  </span>
                  <span className="truncate text-white/80">
                    {row.player_name || row.steam_id}
                  </span>
                </span>
                <span className="tabular-nums text-white">{row.damage}</span>
              </li>
            ))}
          </ul>
        )}
      </Section>
      <Section title="Damage by hit group">
        {stats.hit_groups.length === 0 ? (
          <p className="text-white/60">No hits recorded.</p>
        ) : (
          <ul className="space-y-1">
            {stats.hit_groups.map((row) => (
              <li
                key={row.hit_group}
                className="flex items-center justify-between gap-2"
                data-testid={`player-stats-hit-group-${row.hit_group}`}
              >
                <span className="truncate text-white/80">{row.label}</span>
                <span className="tabular-nums text-white">
                  {row.damage}{" "}
                  <span className="text-white/50">({row.hits})</span>
                </span>
              </li>
            ))}
          </ul>
        )}
      </Section>
    </div>
  )
}

// UtilityCard renders the match-level utility profile: throws by type, flash
// assists, total blind time, enemies flashed.
function UtilityCard({ utility }: { utility: PlayerMatchStats["utility"] }) {
  return (
    <div
      data-testid="player-stats-utility-card"
      className="space-y-2 rounded border border-white/10 bg-black/30 p-2 text-xs"
    >
      <div className="text-[10px] uppercase tracking-wide text-white/60">
        Utility (match)
      </div>
      <div className="grid grid-cols-3 gap-2">
        <Stat label="Smokes" value={`${utility.smokes_thrown}`} />
        <Stat label="Flashes" value={`${utility.flashes_thrown}`} />
        <Stat label="HEs" value={`${utility.hes_thrown}`} />
      </div>
      <div className="grid grid-cols-3 gap-2">
        <Stat label="Molotovs" value={`${utility.molotovs_thrown}`} />
        <Stat label="Decoys" value={`${utility.decoys_thrown}`} />
        <Stat label="Flash assists" value={`${utility.flash_assists}`} />
      </div>
      <div className="grid grid-cols-2 gap-2">
        <Stat label="Enemies flashed" value={`${utility.enemies_flashed}`} />
        <Stat
          label="Blind time"
          value={`${utility.blind_time_inflicted_secs.toFixed(1)}s`}
        />
      </div>
    </div>
  )
}

function Section({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <div className="text-[10px] uppercase tracking-wide text-white/60">
        {title}
      </div>
      {children}
    </div>
  )
}

// PlayerStatsPanel is the right-side player deep-stats overlay. It is non-modal
// (a plain absolute aside, not a Radix Dialog) so the canvas keeps receiving
// pointer/wheel events outside the panel rectangle. Closes via the in-header
// X button or by clicking the same player a second time.
export function PlayerStatsPanel() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const setSelectedPlayer = useViewerStore((s) => s.setSelectedPlayer)
  const setRound = useViewerStore((s) => s.setRound)
  const currentTick = useViewerStore((s) => s.currentTick)

  const { data: rounds } = useRounds(demoId)
  const { data: stats, isLoading } = usePlayerStats(demoId, steamId)

  const activeRoundNumber = useMemo(() => {
    if (!rounds?.length) return 1
    const idx = getActiveRoundIndex(rounds, currentTick)
    return rounds[idx].round_number
  }, [rounds, currentTick])

  if (!steamId) return null

  const roundDetail = stats
    ? (stats.rounds.find((r) => r.round_number === activeRoundNumber) ?? null)
    : null

  const team = stats?.team_side ?? ""
  const sideStripe =
    team === "CT"
      ? "from-sky-400/80 to-sky-400/0"
      : team === "T"
        ? "from-orange-400/80 to-orange-400/0"
        : "from-white/30 to-transparent"

  return (
    <aside
      data-testid="player-stats-panel"
      className="hud-panel absolute right-0 top-0 z-30 flex h-full flex-col rounded-none border-l border-r-0 border-t-0 border-white/[0.07] text-white"
      style={{ width: PANEL_WIDTH }}
    >
      {/* Side accent gradient stripe along the top edge */}
      <span
        aria-hidden="true"
        className={`absolute left-0 right-0 top-0 h-px bg-gradient-to-r ${sideStripe}`}
      />
      <header className="flex items-center justify-between gap-2 border-b border-white/[0.07] bg-white/[0.015] px-3.5 py-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <div
            aria-hidden="true"
            className={`hud-display flex h-9 w-9 shrink-0 items-center justify-center rounded-md text-[13px] font-semibold leading-none ring-1 ring-inset ${
              team === "CT"
                ? "bg-sky-400/15 text-sky-200 ring-sky-400/30"
                : team === "T"
                  ? "bg-orange-400/15 text-orange-200 ring-orange-400/30"
                  : "bg-white/5 text-white/70 ring-white/10"
            }`}
          >
            {(stats?.player_name ?? "P").slice(0, 2).toUpperCase()}
          </div>
          <div className="min-w-0">
            <div
              data-testid="player-stats-panel-name"
              className="truncate text-[13px] font-semibold leading-tight"
            >
              {stats?.player_name ?? (isLoading ? "Loading…" : "Player")}
            </div>
            <div
              data-testid="player-stats-panel-team"
              className={`hud-callsign text-[10px] ${sideColor(team)}`}
            >
              {team || "—"}
            </div>
          </div>
        </div>
        <button
          type="button"
          data-testid="player-stats-panel-close"
          onClick={() => setSelectedPlayer(null)}
          className="rounded-md p-1.5 text-white/60 ring-1 ring-inset ring-white/0 transition-all hover:bg-white/10 hover:text-white hover:ring-white/15"
          aria-label="Close player panel"
        >
          <X className="h-4 w-4" />
        </button>
      </header>

      <div className="flex-1 overflow-y-auto p-3">
        <Tabs defaultValue="match" className="space-y-3">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="live">Live</TabsTrigger>
            <TabsTrigger value="match">Match</TabsTrigger>
            <TabsTrigger value="round">Round</TabsTrigger>
            <TabsTrigger value="detail">Detail</TabsTrigger>
          </TabsList>

          <TabsContent value="live" className="space-y-2">
            <PlayerLiveHud steamId={steamId} />
          </TabsContent>

          <TabsContent value="match" className="space-y-3">
            <AnalysisOverallGauge />
            {isLoading || !stats ? (
              <p className="text-sm text-white/60">Loading match stats…</p>
            ) : (
              <>
                <MatchSummary stats={stats} />
                <RoundStrip
                  rounds={stats.rounds}
                  currentRound={activeRoundNumber}
                  onSelectRound={setRound}
                />
                <MovementSparkline rounds={stats.rounds} />
              </>
            )}
          </TabsContent>

          <TabsContent value="round" className="space-y-3">
            {isLoading || !stats ? (
              <p className="text-sm text-white/60">Loading…</p>
            ) : (
              <RoundDetailCard
                round={roundDetail}
                movement={stats.movement}
                utility={stats.utility}
              />
            )}
          </TabsContent>

          <TabsContent value="detail" className="space-y-3">
            {isLoading || !stats ? (
              <p className="text-sm text-white/60">Loading…</p>
            ) : (
              <DetailLists stats={stats} />
            )}
          </TabsContent>
        </Tabs>
      </div>
    </aside>
  )
}
