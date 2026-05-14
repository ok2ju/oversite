import { useEffect } from "react"
import { useNavigate, useParams, useOutletContext } from "react-router-dom"
import { ChevronLeft, Play } from "lucide-react"
import { useMatchOverview } from "@/hooks/use-match-overview"
import { resolveMap } from "@/components/demos/map-tile"
import { Skeleton } from "@/components/ui/skeleton"
import type {
  HalfOverview,
  MatchFormat,
  MatchOverview,
  PlayerOverview,
  RoundOverview,
  TeamOverview,
} from "@/types/match-overview"
import type { HeaderActionsContext } from "@/routes/root"

function formatDuration(secs: number): string {
  if (!secs) return "—"
  const h = Math.floor(secs / 3600)
  const m = Math.floor((secs % 3600) / 60)
  const s = secs % 60
  return h > 0
    ? `${h}:${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`
    : `${m}:${s.toString().padStart(2, "0")}`
}

function formatDate(iso: string): string {
  if (!iso) return ""
  return new Date(iso).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}

export default function MatchOverviewPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const ctx = useOutletContext<HeaderActionsContext | undefined>()

  const overviewQuery = useMatchOverview(id ?? null)

  useEffect(() => {
    ctx?.setHeaderActions(null)
    return () => ctx?.setHeaderActions(null)
  }, [ctx])

  if (overviewQuery.isLoading) {
    return (
      <div className="mo-wrap">
        <Skeleton className="h-[60px] w-full" />
        <Skeleton className="h-[240px] w-full" />
        <Skeleton className="h-[120px] w-full" />
      </div>
    )
  }

  const overview = overviewQuery.data
  if (!overview || !overview.demo) {
    return (
      <div className="mo-wrap">
        <div className="rounded-md border border-[var(--border)] bg-[var(--bg-elevated)] p-6 text-center text-[var(--text-muted)]">
          Demo not found.
        </div>
      </div>
    )
  }

  const { demo, format, team_a, team_b, rounds, halves, kpis } = overview
  const mapMeta = resolveMap(demo.map_name)
  const ghost = (mapMeta?.name ?? demo.map_name).toUpperCase()
  const streakTeamLabel =
    kpis.streak_team === "a"
      ? team_a.name
      : kpis.streak_team === "b"
        ? team_b.name
        : ""
  const formatLine = format.has_overtime
    ? `of ${format.regulation_rounds} reg. + OT`
    : `of ${format.regulation_rounds} max`

  return (
    <div className="mo-wrap">
      <div className="mo-topbar">
        <button
          type="button"
          className="btn-sm ghost"
          onClick={() => navigate("/demos")}
        >
          <ChevronLeft className="h-3 w-3" />
          Back to demos
        </button>
        <div className="mo-crumb">
          <span>{formatDate(demo.match_date || demo.created_at)}</span>
          <span className="mo-crumb-sep">·</span>
          <span>{formatDuration(demo.duration_secs)}</span>
        </div>
        <div className="flex-1" />
        <button
          type="button"
          className="btn-sm primary"
          onClick={() => navigate(`/demos/${demo.id}`)}
        >
          <Play className="h-3 w-3" />
          Open in viewer
        </button>
      </div>

      {/* Hero */}
      <div className="mo-hero">
        <div className="mo-hero-band-a" />
        <div className="mo-hero-band-b" />
        <div className="mo-hero-map-ghost">{ghost}</div>
        <div className="mo-hero-inner">
          <div className="mo-hero-side">
            <div className="mo-team-flag a">A</div>
            <div className="mo-team-block">
              <div className="mo-team-eyebrow">Team</div>
              <div className="mo-team-name">{team_a.name}</div>
              <div className="mo-team-meta">
                {team_a.players.length} players
              </div>
            </div>
          </div>
          <div className="mo-hero-center">
            <div className="mo-hero-eyebrow">Final score</div>
            <div className="mo-score">
              <span className="mo-score-a">{team_a.score}</span>
              <span className="mo-score-sep">/</span>
              <span className="mo-score-b">{team_b.score}</span>
            </div>
            <div className="mo-map-line">
              <span className="mo-map">{mapMeta?.name ?? demo.map_name}</span>
              <span className="mo-sep">·</span>
              <span>{format.total_rounds} rounds</span>
              <span className="mo-sep">·</span>
              <span>{formatDuration(demo.duration_secs)}</span>
            </div>
          </div>
          <div className="mo-hero-side them">
            <div className="mo-team-flag b">B</div>
            <div className="mo-team-block">
              <div className="mo-team-eyebrow">Team</div>
              <div className="mo-team-name">{team_b.name}</div>
              <div className="mo-team-meta">
                {team_b.players.length} players
              </div>
            </div>
          </div>
        </div>
        {halves.length > 0 && (
          <div className="mo-hero-halves">
            {halves.map((h) => (
              <HalfBlock key={h.label} half={h} />
            ))}
          </div>
        )}
      </div>

      {/* KPI strip */}
      <div className="mo-kpis">
        <div className="mo-kpi">
          <div className="mo-kpi-eyebrow">Rounds</div>
          <div className="mo-kpi-num">{kpis.total_rounds}</div>
          <div className="mo-kpi-foot">{formatLine}</div>
        </div>
        <div className="mo-kpi">
          <div className="mo-kpi-eyebrow">Pistol rounds</div>
          <div className="mo-kpi-num">
            <span className="a">{kpis.pistol_a}</span>
            <i>/</i>
            <span className="b">{kpis.pistol_b}</span>
          </div>
          <div className="mo-kpi-foot">A vs B opens</div>
        </div>
        <div className="mo-kpi">
          <div className="mo-kpi-eyebrow">Longest streak</div>
          <div className="mo-kpi-num">
            <span className={kpis.streak_team || ""}>
              {kpis.longest_streak}
            </span>
          </div>
          <div className="mo-kpi-foot">
            {streakTeamLabel
              ? `consecutive · ${streakTeamLabel}`
              : "consecutive"}
          </div>
        </div>
        <div className="mo-kpi">
          <div className="mo-kpi-eyebrow">Max lead</div>
          <div className="mo-kpi-num">{kpis.max_lead}</div>
          <div className="mo-kpi-foot">rounds in front</div>
        </div>
      </div>

      {/* Economy chart */}
      {rounds.length > 0 && (
        <ChartCard
          title="Economy"
          sub="Tall bar = full buy · short bar = eco · mixed = force / anti-eco"
          teamAName={team_a.name}
          teamBName={team_b.name}
          rounds={rounds}
          format={format}
          getA={(r) => r.team_a_equip_value}
          getB={(r) => r.team_b_equip_value}
          fmt={(v) => `$${(v / 1000).toFixed(1)}k`}
          buildTicks={buildMoneyTicks}
        />
      )}

      {/* Damage chart */}
      {rounds.length > 0 && (
        <ChartCard
          title="Damage"
          sub="Total damage dealt by each team — spot carry rounds and lopsided trades"
          teamAName={team_a.name}
          teamBName={team_b.name}
          rounds={rounds}
          format={format}
          getA={(r) => r.team_a_damage}
          getB={(r) => r.team_b_damage}
          fmt={(v) => v.toFixed(0)}
          buildTicks={buildDamageTicks}
        />
      )}

      {/* Head-to-head + Top performers */}
      <div className="mo-grid">
        <div className="mo-card">
          <div className="mo-card-head">
            <div>
              <div className="mo-card-title">Head to head</div>
              <div className="mo-card-sub">Aggregate team totals</div>
            </div>
          </div>
          <div className="mo-compare">
            <CompareRow
              label="Kills"
              a={team_a.totals.kills}
              b={team_b.totals.kills}
            />
            <CompareRow
              label="Deaths"
              a={team_a.totals.deaths}
              b={team_b.totals.deaths}
              invert
            />
            <CompareRow
              label="Assists"
              a={team_a.totals.assists}
              b={team_b.totals.assists}
            />
            <CompareRow
              label="ADR"
              a={team_a.totals.adr}
              b={team_b.totals.adr}
              fmt={(v) => v.toFixed(1)}
            />
            <CompareRow
              label="Rating"
              a={team_a.totals.rating}
              b={team_b.totals.rating}
              fmt={(v) => v.toFixed(2)}
            />
          </div>
        </div>

        <div className="mo-card">
          <div className="mo-card-head">
            <div>
              <div className="mo-card-title">Top performers</div>
              <div className="mo-card-sub">
                Highest rated player from each side
              </div>
            </div>
          </div>
          <div className="mo-top">
            {team_a.top_performer ? (
              <TopRow p={team_a.top_performer} team="a" />
            ) : (
              <NoTop side="A" />
            )}
            {team_b.top_performer ? (
              <TopRow p={team_b.top_performer} team="b" />
            ) : (
              <NoTop side="B" />
            )}
          </div>
        </div>
      </div>

      {/* Expanded scoreboard */}
      {(team_a.players.length > 0 || team_b.players.length > 0) && (
        <div className="mo-sb">
          {team_a.players.length > 0 && (
            <ScoreboardTeam team={team_a} cls="a" />
          )}
          {team_b.players.length > 0 && (
            <ScoreboardTeam team={team_b} cls="b" />
          )}
        </div>
      )}
    </div>
  )
}

function ScoreboardTeam({ team, cls }: { team: TeamOverview; cls: "a" | "b" }) {
  return (
    <>
      <div className={`mo-sb-team ${cls}`}>
        {team.name} · {team.score}
      </div>
      <ScoreboardHead />
      {team.players.map((p) => (
        <SBRow key={p.steam_id} p={p} />
      ))}
    </>
  )
}

function ScoreboardHead() {
  return (
    <div className="mo-sb-head">
      <span>Player</span>
      <span>K</span>
      <span>A</span>
      <span>D</span>
      <span>HS%</span>
      <span>ADR</span>
      <span>KAST%</span>
      <span>Rating</span>
    </div>
  )
}

function SBRow({ p }: { p: PlayerOverview }) {
  const ratingClass =
    p.rating_2 >= 1.15
      ? "mo-rating hi"
      : p.rating_2 < 0.9
        ? "mo-rating lo"
        : "mo-rating"
  return (
    <div className="mo-sb-row">
      <span className="name">{p.player_name}</span>
      <span>{p.kills}</span>
      <span>{p.assists}</span>
      <span>{p.deaths}</span>
      <span>{Math.round(p.hs_percent)}%</span>
      <span>{p.adr.toFixed(1)}</span>
      <span>{Math.round(p.kast)}%</span>
      <span className={ratingClass}>{p.rating_2.toFixed(2)}</span>
    </div>
  )
}

function CompareRow({
  label,
  a,
  b,
  invert = false,
  fmt = (v: number) => `${Math.round(v * 100) / 100}`,
}: {
  label: string
  a: number
  b: number
  invert?: boolean
  fmt?: (v: number) => string
}) {
  const total = a + b || 1
  const aPct = (a / total) * 100
  const bPct = (b / total) * 100
  const aWin = invert ? a < b : a > b
  const bWin = invert ? b < a : b > a
  return (
    <div className="mo-cmp-row">
      <div className={`mo-cmp-val left ${aWin ? "win" : ""}`}>{fmt(a)}</div>
      <div className="mo-cmp-bar">
        <div
          className={`mo-cmp-fill mine ${aWin ? "win" : ""}`}
          style={{ width: `${aPct}%` }}
        />
        <div
          className={`mo-cmp-fill them ${bWin ? "win" : ""}`}
          style={{ width: `${bPct}%` }}
        />
      </div>
      <div className={`mo-cmp-val right ${bWin ? "win" : ""}`}>{fmt(b)}</div>
      <div className="mo-cmp-label">{label}</div>
    </div>
  )
}

function TopRow({ p, team }: { p: PlayerOverview; team: "a" | "b" }) {
  return (
    <div className={`mo-top-row ${team}`}>
      <div>
        <div className="mo-top-name">{p.player_name}</div>
        <div className="mo-top-stats">
          <span className="kda">
            {p.kills}
            <i>/</i>
            {p.deaths}
            <i>/</i>
            {p.assists}
          </span>
          <span className="stat">
            <i>ADR</i>
            <b>{p.adr.toFixed(1)}</b>
          </span>
          <span className="stat">
            <i>HS</i>
            <b>{Math.round(p.hs_percent)}%</b>
          </span>
        </div>
      </div>
      <div>
        <div className="mo-top-rating-n">{p.rating_2.toFixed(2)}</div>
        <div className="mo-top-rating-l">Rating</div>
      </div>
    </div>
  )
}

function NoTop({ side }: { side: "A" | "B" }) {
  return (
    <div className="mo-top-row">
      <div className="mo-top-name text-[var(--text-muted)]">
        No Team {side} data
      </div>
    </div>
  )
}

function HalfBlock({ half }: { half: HalfOverview }) {
  return (
    <div className="mo-hh">
      <span className="mo-hh-label">{half.label}</span>
      <span className="mo-hh-score">
        <b className={half.team_a_side === "T" ? "t" : "ct"}>
          {half.team_a_wins}
        </b>
        <i>—</i>
        <b className={half.team_b_side === "T" ? "t" : "ct"}>
          {half.team_b_wins}
        </b>
      </span>
      <span className="mo-hh-sides">
        <span className={half.team_a_side === "T" ? "pill-t" : "pill-ct"}>
          {half.team_a_side}
        </span>
        <span className="mo-hh-vs">vs</span>
        <span className={half.team_b_side === "T" ? "pill-t" : "pill-ct"}>
          {half.team_b_side}
        </span>
      </span>
    </div>
  )
}

interface ChartCardProps {
  title: string
  sub: string
  teamAName: string
  teamBName: string
  rounds: RoundOverview[]
  format: MatchFormat
  getA: (r: RoundOverview) => number
  getB: (r: RoundOverview) => number
  fmt: (v: number) => string
  buildTicks: (max: number) => { labels: string[]; yMax: number }
}

interface HalfSlice {
  label: string
  rounds: RoundOverview[]
}

// splitRoundsIntoHalves walks the per-round payload and groups it into the
// same halves the server identified. Returns one slice per non-empty half so
// overtime demos render 4+ groups naturally.
function splitRoundsIntoHalves(
  rounds: RoundOverview[],
  format: MatchFormat,
): HalfSlice[] {
  if (!rounds.length) return []
  const slices: HalfSlice[] = []
  const half1: RoundOverview[] = []
  const half2: RoundOverview[] = []
  const overtime: RoundOverview[] = []
  for (const r of rounds) {
    if (r.is_overtime) overtime.push(r)
    else if (r.round_number <= format.halftime_round) half1.push(r)
    else half2.push(r)
  }
  if (half1.length) slices.push({ label: "1st half", rounds: half1 })
  if (half2.length) slices.push({ label: "2nd half", rounds: half2 })
  if (overtime.length && format.overtime_half_len > 0) {
    let idx = 0
    let from = 0
    while (from < overtime.length) {
      const to = Math.min(from + format.overtime_half_len, overtime.length)
      const otNum = Math.floor(idx / 2) + 1
      const half = idx % 2 === 0 ? "first" : "second"
      slices.push({
        label: `OT${otNum} ${half}`,
        rounds: overtime.slice(from, to),
      })
      from = to
      idx++
    }
  }
  return slices
}

function buildDamageTicks(max: number): { labels: string[]; yMax: number } {
  const yMax = Math.max(100, Math.ceil(max / 100) * 100)
  const half = Math.round(yMax / 2 / 10) * 10
  const quarter = Math.round(yMax / 4 / 10) * 10
  return { labels: [`${yMax}`, `${half}`, `${quarter}`], yMax }
}

function buildMoneyTicks(max: number): { labels: string[]; yMax: number } {
  const yMax = Math.max(1000, Math.ceil(max / 500) * 500)
  const half = Math.round(yMax / 2 / 100) * 100
  const quarter = Math.round(yMax / 4 / 100) * 100
  const f = (v: number) => `$${(v / 1000).toFixed(1)}k`
  return { labels: [f(yMax), f(half), f(quarter)], yMax }
}

function ChartCard({
  title,
  sub,
  teamAName,
  teamBName,
  rounds,
  format,
  getA,
  getB,
  fmt,
  buildTicks,
}: ChartCardProps) {
  const slices = splitRoundsIntoHalves(rounds, format)
  // Compute one shared yMax across all halves so per-slice bars are visually
  // comparable. The label set comes from the same scale.
  const maxVal = rounds.reduce((acc, r) => Math.max(acc, getA(r), getB(r)), 0)
  const { labels, yMax } = buildTicks(maxVal)

  return (
    <div className="mo-chart-card">
      <div className="mo-chart-head">
        <div>
          <div className="mo-chart-title">
            {title} <span className="mo-sep">·</span>{" "}
            <span style={{ color: "var(--text-muted)", fontWeight: 400 }}>
              {title === "Economy" ? "equipment value per round" : "per round"}
            </span>
          </div>
          <div className="mo-chart-sub">{sub}</div>
        </div>
        <div className="mo-chart-legend">
          <span className="mo-chart-legend-dot">{teamAName}</span>
          <span className="mo-chart-legend-dot b">{teamBName}</span>
        </div>
      </div>
      <div className="mo-chart-body">
        {slices.map((slice, i) => {
          const avg =
            slice.rounds.length > 0
              ? slice.rounds.reduce((s, r) => s + getA(r) + getB(r), 0) /
                (slice.rounds.length * 2)
              : 0
          return (
            <ChartHalf
              key={slice.label}
              label={slice.label}
              rounds={slice.rounds}
              getA={getA}
              getB={getB}
              fmt={fmt}
              yTicks={labels}
              yMax={yMax}
              showHT={i < slices.length - 1}
              avg={avg}
            />
          )
        })}
      </div>
      {/* Per-round result strip aligned to the bars above */}
      <div className="mo-rounds-bar">
        {slices.map((slice, i) => (
          <RoundsBarHalf
            key={slice.label}
            rounds={slice.rounds}
            showHT={i < slices.length - 1}
          />
        ))}
      </div>
    </div>
  )
}

function ChartHalf({
  label,
  rounds,
  getA,
  getB,
  fmt,
  yTicks,
  yMax,
  showHT,
  avg,
}: {
  label: string
  rounds: RoundOverview[]
  getA: (r: RoundOverview) => number
  getB: (r: RoundOverview) => number
  fmt: (v: number) => string
  yTicks: string[]
  yMax: number
  showHT: boolean
  avg: number
}) {
  return (
    <>
      <div className="mo-chart-half">
        <div className="mo-chart-half-label">
          <span>{label}</span>
          {rounds.length > 0 && (
            <span className="mo-chart-half-avg">avg {fmt(avg)}</span>
          )}
        </div>
        <ChartPlot
          rounds={rounds}
          getA={getA}
          getB={getB}
          yTicks={yTicks}
          yMax={yMax}
        />
        <div className="mo-chart-xaxis">
          <span className="mo-chart-xaxis-label">RND</span>
          <div className="mo-chart-xaxis-nums">
            {rounds.map((r) => (
              <span key={r.round_number}>{r.round_number}</span>
            ))}
          </div>
        </div>
      </div>
      {showHT && <div className="mo-chart-divider" aria-hidden="true" />}
    </>
  )
}

function ChartPlot({
  rounds,
  getA,
  getB,
  yTicks,
  yMax,
}: {
  rounds: RoundOverview[]
  getA: (r: RoundOverview) => number
  getB: (r: RoundOverview) => number
  yTicks: string[]
  yMax: number
}) {
  return (
    <div className="mo-chart-plot">
      <div className="mo-chart-yaxis">
        {yTicks.map((t) => (
          <span key={t}>{t}</span>
        ))}
      </div>
      <div className="mo-chart-bars">
        {rounds.map((r) => {
          const aH = Math.min(100, (getA(r) / yMax) * 100)
          const bH = Math.min(100, (getB(r) / yMax) * 100)
          return (
            <div key={r.round_number} className="mo-chart-pair">
              <div
                className="mo-chart-bar a"
                style={{ height: `${aH}%` }}
                title={`R${r.round_number} · A ${getA(r)}`}
              />
              <div
                className="mo-chart-bar b"
                style={{ height: `${bH}%` }}
                title={`R${r.round_number} · B ${getB(r)}`}
              />
            </div>
          )
        })}
      </div>
    </div>
  )
}

function RoundsBarHalf({
  rounds,
  showHT,
}: {
  rounds: RoundOverview[]
  showHT: boolean
}) {
  return (
    <>
      <div className="mo-rounds-bar-half">
        <span />
        <div className="mo-rounds-bar-cells">
          {rounds.map((r) => (
            <div key={r.round_number} className={`mo-rb-cell ${r.winner}`}>
              <span className="mo-rb-n">{r.round_number}</span>
              <span
                className={`mo-rb-side ${r.winner_side === "T" ? "t" : "ct"}`}
              >
                {r.winner_side}
              </span>
            </div>
          ))}
        </div>
      </div>
      {showHT && <span className="mo-chart-ht">HT</span>}
    </>
  )
}

// Export of unused types for TypeScript module resolution.
export type { MatchOverview }
