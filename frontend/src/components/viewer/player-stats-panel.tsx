import { useMemo } from "react"
import { X } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { usePlayerStats } from "@/hooks/use-player-stats"
import { usePlayerAnalysis } from "@/hooks/use-analysis"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Progress } from "@/components/ui/progress"
import { PlayerLiveHud } from "@/components/viewer/player-live-hud"
import type { PlayerMatchStats, PlayerRoundDetail } from "@/types/player-stats"
import type { Round } from "@/types/round"

const PANEL_WIDTH = 380

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
const sideDot = (side: string) =>
  side === "CT"
    ? "bg-sky-400 shadow-[0_0_6px_rgba(56,189,248,0.7)]"
    : side === "T"
      ? "bg-amber-400 shadow-[0_0_6px_rgba(251,191,36,0.7)]"
      : "bg-white/40"

function HeaderPerformance() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const { data, isLoading } = usePlayerAnalysis(demoId, steamId)

  if (isLoading || !data || !data.steam_id) return null

  const score = Math.max(0, Math.min(100, data.overall_score))
  const radius = 13
  const circumference = 2 * Math.PI * radius
  const dash = (score / 100) * circumference

  const tierStroke =
    score >= 75
      ? "stroke-emerald-400"
      : score >= 50
        ? "stroke-amber-400"
        : "stroke-rose-400"
  const tierText =
    score >= 75
      ? "text-emerald-400"
      : score >= 50
        ? "text-amber-400"
        : "text-rose-400"
  const tierGlow =
    score >= 75
      ? "drop-shadow(0 0 5px rgba(74,222,128,0.55))"
      : score >= 50
        ? "drop-shadow(0 0 5px rgba(251,191,36,0.55))"
        : "drop-shadow(0 0 5px rgba(251,113,133,0.55))"
  const tierLabel = score >= 75 ? "Elite" : score >= 50 ? "Solid" : "Low"

  return (
    <div
      data-testid="player-stats-header-performance"
      className="flex items-center gap-2"
    >
      <div className="flex flex-col items-end leading-none">
        <span className="hud-callsign text-[8.5px] font-semibold uppercase tracking-[0.22em] text-white/40">
          Perf
        </span>
        <span
          className={`mt-1 text-[9.5px] font-semibold uppercase tracking-[0.18em] ${tierText}`}
        >
          {tierLabel}
        </span>
      </div>
      <div className="relative h-9 w-9 shrink-0">
        <svg viewBox="0 0 36 36" className="h-full w-full -rotate-90">
          <circle
            cx="18"
            cy="18"
            r={radius}
            fill="none"
            stroke="rgba(255,255,255,0.07)"
            strokeWidth="2.5"
          />
          <circle
            cx="18"
            cy="18"
            r={radius}
            fill="none"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeDasharray={`${dash} ${circumference - dash}`}
            className={tierStroke}
            style={{
              filter: tierGlow,
              transition: "stroke-dasharray 320ms ease",
            }}
          />
        </svg>
        <span className="hud-display absolute inset-0 flex items-center justify-center text-[11.5px] font-semibold leading-none text-white tabular-nums">
          {score}
        </span>
      </div>
    </div>
  )
}

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
    <div className="rounded-md border border-white/[0.07] bg-white/[0.02] px-2.5 py-2">
      <div className="text-[10px] font-medium uppercase tracking-wide text-white/55">
        {label}
      </div>
      <div className="mt-0.5 text-[12.5px] font-medium tabular-nums leading-none text-white">
        {value}
      </div>
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
    <div data-testid="player-stats-round-strip" className="space-y-1.5">
      <div className="text-[10px] font-medium uppercase tracking-wide text-white/55">
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
              className={`flex h-12 w-7 flex-col items-center justify-end overflow-hidden rounded-[4px] border text-[10px] tabular-nums transition-colors ${
                active
                  ? "border-white/70 bg-white/[0.12] text-white"
                  : "border-white/[0.07] bg-white/[0.02] text-white/65 hover:bg-white/[0.08]"
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
    <div data-testid="player-stats-movement-sparkline" className="space-y-1.5">
      <div className="text-[10px] font-medium uppercase tracking-wide text-white/55">
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
        className="rounded-md border border-white/[0.07] bg-white/[0.02] p-3 text-[12.5px] text-white/55"
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
      className="space-y-3 rounded-md border border-white/[0.07] bg-white/[0.02] p-3"
    >
      <div className="flex items-baseline justify-between">
        <span className="text-[10.5px] font-medium uppercase tracking-wide text-white/55">
          Round {round.round_number}
        </span>
        <span
          className={`text-[10.5px] font-semibold uppercase tracking-wide ${sideColor(round.team_side)}`}
        >
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
      className="space-y-2 rounded-md border border-white/[0.07] bg-black/30 p-2.5"
    >
      <div className="flex items-baseline justify-between">
        <span className="text-[10px] font-medium uppercase tracking-wide text-white/55">
          Movement (match)
        </span>
        <span className="text-[11px] tabular-nums text-white/70">
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
    <div className="space-y-1">
      <div className="flex items-baseline justify-between">
        <span
          className={`text-[10px] font-medium uppercase tracking-wide text-white/55 ${
            helpText ? "cursor-help underline decoration-dotted" : ""
          }`}
          title={helpText}
        >
          {label}
        </span>
        <span className="text-[11px] tabular-nums text-white/80">{value}%</span>
      </div>
      <Progress
        data-testid={testId}
        value={Math.max(0, Math.min(100, value))}
        className="h-1 bg-white/[0.06]"
      />
    </div>
  )
}

function DetailLists({ stats }: { stats: PlayerMatchStats }) {
  return (
    <div className="space-y-3 text-[12px]">
      <Section title="Damage by weapon">
        {stats.damage_by_weapon.length === 0 ? (
          <p className="text-[12px] text-white/55">No damage recorded.</p>
        ) : (
          <ul className="space-y-1">
            {stats.damage_by_weapon.map((row) => (
              <li
                key={row.weapon}
                className="flex items-center justify-between gap-2"
                data-testid={`player-stats-damage-weapon-${row.weapon}`}
              >
                <span className="truncate text-white/75">{row.weapon}</span>
                <span className="tabular-nums font-medium text-white">
                  {row.damage}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Section>
      <Section title="Damage by opponent">
        {stats.damage_by_opponent.length === 0 ? (
          <p className="text-[12px] text-white/55">No damage recorded.</p>
        ) : (
          <ul className="space-y-1">
            {stats.damage_by_opponent.map((row) => (
              <li
                key={row.steam_id}
                className="flex items-center justify-between gap-2"
                data-testid={`player-stats-damage-opponent-${row.steam_id}`}
              >
                <span className="flex min-w-0 items-center gap-1.5">
                  <span
                    className={`text-[10px] font-semibold uppercase tracking-wide ${sideColor(row.team_side)}`}
                  >
                    {row.team_side || "?"}
                  </span>
                  <span className="truncate text-white/75">
                    {row.player_name || row.steam_id}
                  </span>
                </span>
                <span className="tabular-nums font-medium text-white">
                  {row.damage}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Section>
      <Section title="Damage by hit group">
        {stats.hit_groups.length === 0 ? (
          <p className="text-[12px] text-white/55">No hits recorded.</p>
        ) : (
          <ul className="space-y-1">
            {stats.hit_groups.map((row) => (
              <li
                key={row.hit_group}
                className="flex items-center justify-between gap-2"
                data-testid={`player-stats-hit-group-${row.hit_group}`}
              >
                <span className="truncate text-white/75">{row.label}</span>
                <span className="tabular-nums font-medium text-white">
                  {row.damage}{" "}
                  <span className="font-normal text-white/45">
                    ({row.hits})
                  </span>
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
      className="space-y-2 rounded-md border border-white/[0.07] bg-black/30 p-2.5"
    >
      <div className="text-[10px] font-medium uppercase tracking-wide text-white/55">
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
      <div className="text-[10px] font-medium uppercase tracking-wide text-white/55">
        {title}
      </div>
      {children}
    </div>
  )
}

// PlayerStatsPanel is the right-side player deep-stats pane. It renders as a
// flex sibling of the viewer area, so opening it squeezes the canvas rather
// than overlaying it (the canvas re-flows via its ResizeObserver). Closes via
// the in-header X button or by clicking the same player a second time.
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

  return (
    <aside
      data-testid="player-stats-panel"
      className="relative flex h-full shrink-0 flex-col border-l border-white/[0.07] bg-[#14171c] text-white"
      style={{ width: PANEL_WIDTH }}
    >
      <header className="flex h-[68px] shrink-0 items-center gap-3 border-b border-white/[0.07] pl-4 pr-2">
        <div className="flex min-w-0 flex-1 flex-col gap-1.5">
          <div
            data-testid="player-stats-panel-name"
            className="truncate text-[14px] font-semibold leading-none tracking-tight text-white"
          >
            {stats?.player_name ?? (isLoading ? "Loading…" : "Player")}
          </div>
          <div className="flex items-center gap-1.5">
            <span
              aria-hidden="true"
              className={`h-1.5 w-1.5 shrink-0 rounded-full ${sideDot(team)}`}
            />
            <span
              data-testid="player-stats-panel-team"
              className={`text-[9.5px] font-semibold uppercase leading-none tracking-[0.22em] ${sideColor(team)}`}
            >
              {team ? `${team} Side` : "—"}
            </span>
          </div>
        </div>

        <HeaderPerformance />

        <span
          aria-hidden="true"
          className="h-8 w-px shrink-0 bg-white/[0.07]"
        />

        <button
          type="button"
          data-testid="player-stats-panel-close"
          onClick={() => setSelectedPlayer(null)}
          className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-white/40 transition-colors hover:bg-white/[0.08] hover:text-white focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-white/30"
          aria-label="Close player panel"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </header>

      <div className="flex-1 overflow-y-auto px-3 py-3">
        <Tabs defaultValue="match" className="space-y-3">
          <TabsList className="grid h-8 w-full grid-cols-4 bg-white/[0.04] p-0.5">
            <TabsTrigger value="live" className="h-7 text-[11px] font-medium">
              Live
            </TabsTrigger>
            <TabsTrigger value="match" className="h-7 text-[11px] font-medium">
              Match
            </TabsTrigger>
            <TabsTrigger value="round" className="h-7 text-[11px] font-medium">
              Round
            </TabsTrigger>
            <TabsTrigger value="detail" className="h-7 text-[11px] font-medium">
              Detail
            </TabsTrigger>
          </TabsList>

          <TabsContent value="live" className="space-y-2">
            <PlayerLiveHud steamId={steamId} />
          </TabsContent>

          <TabsContent value="match" className="space-y-3">
            {isLoading || !stats ? (
              <p className="text-[12px] text-white/55">Loading match stats…</p>
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
              <p className="text-[12px] text-white/55">Loading…</p>
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
              <p className="text-[12px] text-white/55">Loading…</p>
            ) : (
              <DetailLists stats={stats} />
            )}
          </TabsContent>
        </Tabs>
      </div>
    </aside>
  )
}
