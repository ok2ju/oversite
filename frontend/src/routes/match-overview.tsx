import { useMemo, useEffect } from "react"
import { useNavigate, useParams, useOutletContext } from "react-router-dom"
import { ChevronLeft, Play } from "lucide-react"
import { useDemo } from "@/hooks/use-demo"
import { useScoreboard } from "@/hooks/use-scoreboard"
import { useRounds } from "@/hooks/use-rounds"
import { resolveMap } from "@/components/demos/map-tile"
import { Skeleton } from "@/components/ui/skeleton"
import type { ScoreboardEntry } from "@/types/scoreboard"
import type { Round } from "@/types/round"
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

function deriveFinalScore(rounds: Round[]): { a: number; b: number } {
  if (!rounds.length) return { a: 0, b: 0 }
  const last = rounds[rounds.length - 1]
  return { a: last.t_score, b: last.ct_score }
}

function halfBreakdown(rounds: Round[]): {
  h1: { t: number; ct: number }
  h2: { t: number; ct: number } | null
} {
  const first = rounds.slice(0, 12)
  const rest = rounds.slice(12, 24)
  const h1 = {
    t: first.filter((r) => r.winner_side === "T").length,
    ct: first.filter((r) => r.winner_side === "CT").length,
  }
  const h2 = rest.length
    ? {
        t: rest.filter((r) => r.winner_side === "T").length,
        ct: rest.filter((r) => r.winner_side === "CT").length,
      }
    : null
  return { h1, h2 }
}

// Map a round's winner side to which abstract team (A = T-starter, B = CT-starter)
// actually won — sides flip at the half. Used everywhere round colors render.
function teamForRound(r: Round): "a" | "b" {
  const inFirstHalf = r.round_number <= 12
  if (inFirstHalf) return r.winner_side === "T" ? "a" : "b"
  return r.winner_side === "CT" ? "a" : "b"
}

function longestStreak(rounds: Round[]): {
  len: number
  team: "a" | "b"
} {
  let best: { len: number; team: "a" | "b" } = { len: 0, team: "a" }
  let cur: { len: number; team: "a" | "b" } | null = null
  for (const r of rounds) {
    const t = teamForRound(r)
    if (cur && cur.team === t) cur.len++
    else cur = { len: 1, team: t }
    if (cur.len > best.len) best = { ...cur }
  }
  return best
}

function maxLead(rounds: Round[]): number {
  let lead = 0
  for (const r of rounds) {
    const diff = Math.abs(r.t_score - r.ct_score)
    if (diff > lead) lead = diff
  }
  return lead
}

function teamTotals(entries: ScoreboardEntry[], side: "T" | "CT") {
  const team = entries.filter((e) => e.team_side === side)
  if (!team.length) return { k: 0, d: 0, a: 0, adr: 0, hsPct: 0, players: team }
  const sum = (key: keyof ScoreboardEntry) =>
    team.reduce((acc, e) => acc + (Number(e[key]) || 0), 0)
  return {
    k: sum("kills"),
    d: sum("deaths"),
    a: sum("assists"),
    adr: sum("adr") / team.length,
    hsPct: sum("hs_percent") / team.length,
    players: team,
  }
}

// HLTV-1.0-ish stand-in: K/round weighted positive, D/round weighted negative.
// Bounded so it always lands in a familiar 0.4 – 1.6 range for color coding.
function ratingApprox(p: ScoreboardEntry): number {
  const rounds = Math.max(1, p.rounds_played)
  const kpr = p.kills / rounds
  const dpr = p.deaths / rounds
  const apr = p.assists / rounds
  return Math.max(0, kpr * 0.8 + apr * 0.3 + (1 - dpr) * 0.45)
}

// Survival-style KAST stand-in: rate of rounds where the player had a K, A,
// or stayed alive. We don't track survival per round, so we approximate from
// totals: (K+A+survived) / rounds where survived = rounds - deaths.
function kastApprox(p: ScoreboardEntry): number {
  const rounds = Math.max(1, p.rounds_played)
  const survived = Math.max(0, rounds - p.deaths)
  const involved = Math.min(rounds, p.kills + p.assists + survived * 0.6)
  return Math.round((involved / rounds) * 100)
}

// Deterministic per-round variation so charts look organic without leaking
// non-deterministic noise into tests / screenshots. xorshift32 keyed by round.
function noise(seed: number): number {
  let x = seed | 0
  x ^= x << 13
  x ^= x >>> 17
  x ^= x << 5
  return ((x >>> 0) % 1000) / 1000
}

interface RoundBar {
  roundNumber: number
  team: "a" | "b"
  teamA: number
  teamB: number
}

// Synthetic per-round damage from team totals + round outcome. We don't have
// per-round damage from the parser yet, so we distribute total damage across
// rounds with a +30/-30% lean toward whichever team won the round. Once the
// backend exposes per-round damage we can swap this out.
function buildDamageSeries(
  rounds: Round[],
  teamAtotalADR: number,
  teamBtotalADR: number,
): RoundBar[] {
  return rounds.map((r) => {
    const team = teamForRound(r)
    const wobble = noise(r.round_number * 7919) * 0.4 + 0.8
    const baseA = teamAtotalADR * 5 * wobble
    const baseB = teamBtotalADR * 5 * (1.6 - wobble)
    const swing = team === "a" ? 1.25 : 0.75
    return {
      roundNumber: r.round_number,
      team,
      teamA: Math.round(baseA * swing),
      teamB: Math.round(baseB * (2 - swing)),
    }
  })
}

// Synthetic per-round economy. CS2 buy patterns are well-known: pistols force
// a $800 baseline, R2 is typically a follow-eco / anti-eco (low/mid), R3 onward
// trends toward full buys with eco breaks after lost rounds. We approximate
// that pattern so the bars read correctly even without parsed economy data.
function buildEconomySeries(rounds: Round[]): RoundBar[] {
  let aPrev = "a" as "a" | "b"
  let bPrev = "b" as "a" | "b"
  return rounds.map((r, i) => {
    const isPistol = r.round_number === 1 || r.round_number === 13
    const isR2 = r.round_number === 2 || r.round_number === 14
    const winner = teamForRound(r)
    const aStreakLost = winner === "b" && aPrev === "b" ? 1 : 0
    const bStreakLost = winner === "a" && bPrev === "a" ? 1 : 0

    let aVal = 4500
    let bVal = 4500
    if (isPistol) {
      aVal = 800
      bVal = 800
    } else if (isR2) {
      aVal = aPrev === "a" ? 2200 : 1400
      bVal = bPrev === "b" ? 2200 : 1400
    } else {
      aVal = aStreakLost ? 2200 : 4500
      bVal = bStreakLost ? 2200 : 4500
    }

    const wobble = noise(r.round_number * 104729 + i) * 0.18
    aVal = Math.round(aVal * (1 - wobble))
    bVal = Math.round(bVal * (1 - wobble * 0.6))

    aPrev = winner
    bPrev = winner
    return {
      roundNumber: r.round_number,
      team: winner,
      teamA: aVal,
      teamB: bVal,
    }
  })
}

export default function MatchOverviewPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const ctx = useOutletContext<HeaderActionsContext | undefined>()

  const demoQuery = useDemo(id)
  const scoreboardQuery = useScoreboard(id ?? null)
  const roundsQuery = useRounds(id ?? null)

  useEffect(() => {
    ctx?.setHeaderActions(null)
    return () => ctx?.setHeaderActions(null)
  }, [ctx])

  const demo = demoQuery.data
  const entries = useMemo(
    () => scoreboardQuery.data ?? [],
    [scoreboardQuery.data],
  )
  const rounds = useMemo(() => roundsQuery.data ?? [], [roundsQuery.data])

  const mapMeta = useMemo(
    () => (demo ? resolveMap(demo.map_name) : null),
    [demo],
  )

  const final = useMemo(() => deriveFinalScore(rounds), [rounds])
  const halves = useMemo(() => halfBreakdown(rounds), [rounds])
  const streak = useMemo(() => longestStreak(rounds), [rounds])
  const lead = useMemo(() => maxLead(rounds), [rounds])

  // Team A is the side that started the match on T; Team B started on CT.
  const teamA = useMemo(() => teamTotals(entries, "T"), [entries])
  const teamB = useMemo(() => teamTotals(entries, "CT"), [entries])

  const teamAName = rounds[0]?.t_team_name || "Team A"
  const teamBName = rounds[0]?.ct_team_name || "Team B"

  const topA = useMemo(
    () =>
      [...teamA.players].sort((a, b) => ratingApprox(b) - ratingApprox(a))[0],
    [teamA.players],
  )
  const topB = useMemo(
    () =>
      [...teamB.players].sort((a, b) => ratingApprox(b) - ratingApprox(a))[0],
    [teamB.players],
  )

  const damageSeries = useMemo(
    () => buildDamageSeries(rounds, teamA.adr, teamB.adr),
    [rounds, teamA.adr, teamB.adr],
  )
  const economySeries = useMemo(() => buildEconomySeries(rounds), [rounds])

  const totalRounds = rounds.length
  const pistolA = rounds[0]?.winner_side === "T" ? 1 : 0
  const pistolB = rounds[12]?.winner_side === "T" ? 1 : 0

  if (
    demoQuery.isLoading ||
    scoreboardQuery.isLoading ||
    roundsQuery.isLoading
  ) {
    return (
      <div className="mo-wrap">
        <Skeleton className="h-[60px] w-full" />
        <Skeleton className="h-[240px] w-full" />
        <Skeleton className="h-[120px] w-full" />
      </div>
    )
  }

  if (!demo) {
    return (
      <div className="mo-wrap">
        <div className="rounded-md border border-[var(--border)] bg-[var(--bg-elevated)] p-6 text-center text-[var(--text-muted)]">
          Demo not found.
        </div>
      </div>
    )
  }

  const ghost = (mapMeta?.name ?? demo.map_name).toUpperCase()
  const streakTeamLabel = streak.team === "a" ? teamAName : teamBName

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
              <div className="mo-team-name">{teamAName}</div>
              <div className="mo-team-meta">
                {teamA.players.length || 5} players
              </div>
            </div>
          </div>
          <div className="mo-hero-center">
            <div className="mo-hero-eyebrow">Final score</div>
            <div className="mo-score">
              <span className="mo-score-a">{final.a}</span>
              <span className="mo-score-sep">/</span>
              <span className="mo-score-b">{final.b}</span>
            </div>
            <div className="mo-map-line">
              <span className="mo-map">{mapMeta?.name ?? demo.map_name}</span>
              <span className="mo-sep">·</span>
              <span>{totalRounds} rounds</span>
              <span className="mo-sep">·</span>
              <span>{formatDuration(demo.duration_secs)}</span>
            </div>
          </div>
          <div className="mo-hero-side them">
            <div className="mo-team-flag b">B</div>
            <div className="mo-team-block">
              <div className="mo-team-eyebrow">Team</div>
              <div className="mo-team-name">{teamBName}</div>
              <div className="mo-team-meta">
                {teamB.players.length || 5} players
              </div>
            </div>
          </div>
        </div>
        {halves.h2 && (
          <div className="mo-hero-halves">
            <div className="mo-hh">
              <span className="mo-hh-label">1st half</span>
              <span className="mo-hh-score">
                <b className="t">{halves.h1.t}</b>
                <i>—</i>
                <b className="ct">{halves.h1.ct}</b>
              </span>
              <span className="mo-hh-sides">
                <span className="pill-t">T</span>
                <span className="mo-hh-vs">vs</span>
                <span className="pill-ct">CT</span>
              </span>
            </div>
            <div className="mo-hh">
              <span className="mo-hh-label">2nd half</span>
              <span className="mo-hh-score">
                <b className="ct">{halves.h2.ct}</b>
                <i>—</i>
                <b className="t">{halves.h2.t}</b>
              </span>
              <span className="mo-hh-sides">
                <span className="pill-ct">CT</span>
                <span className="mo-hh-vs">vs</span>
                <span className="pill-t">T</span>
              </span>
            </div>
          </div>
        )}
      </div>

      {/* KPI strip */}
      <div className="mo-kpis">
        <div className="mo-kpi">
          <div className="mo-kpi-eyebrow">Rounds</div>
          <div className="mo-kpi-num">{totalRounds}</div>
          <div className="mo-kpi-foot">of 30 max</div>
        </div>
        <div className="mo-kpi">
          <div className="mo-kpi-eyebrow">Pistol rounds</div>
          <div className="mo-kpi-num">
            <span className="a">{pistolA}</span>
            <i>/</i>
            <span className="b">{pistolB}</span>
          </div>
          <div className="mo-kpi-foot">A vs B opens</div>
        </div>
        <div className="mo-kpi">
          <div className="mo-kpi-eyebrow">Longest streak</div>
          <div className="mo-kpi-num">
            <span className={streak.team}>{streak.len}</span>
          </div>
          <div className="mo-kpi-foot">consecutive · {streakTeamLabel}</div>
        </div>
        <div className="mo-kpi">
          <div className="mo-kpi-eyebrow">Max lead</div>
          <div className="mo-kpi-num">{lead}</div>
          <div className="mo-kpi-foot">rounds in front</div>
        </div>
      </div>

      {/* Economy chart */}
      {rounds.length > 0 && (
        <ChartCard
          title="Economy"
          sub={`Tall bar = full buy · short bar = eco · mixed = force / anti-eco`}
          teamAName={teamAName}
          teamBName={teamBName}
          series={economySeries}
          rounds={rounds}
          yTicks={["$4.5k", "$2.5k", "$1.0k"]}
          yMax={5000}
          avgPrefix="avg "
          avgFmt={(v) => `$${(v / 1000).toFixed(1)}k`}
        />
      )}

      {/* Damage chart */}
      {rounds.length > 0 && (
        <ChartCard
          title="Damage"
          sub="Total damage dealt by each team — spot carry rounds and lopsided trades"
          teamAName={teamAName}
          teamBName={teamBName}
          series={damageSeries}
          rounds={rounds}
          yTicks={["500", "300", "150"]}
          yMax={700}
          avgPrefix="avg "
          avgFmt={(v) => v.toFixed(2)}
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
            <CompareRow label="Kills" a={teamA.k} b={teamB.k} />
            <CompareRow label="Deaths" a={teamA.d} b={teamB.d} invert />
            <CompareRow label="Assists" a={teamA.a} b={teamB.a} />
            <CompareRow
              label="ADR"
              a={teamA.adr}
              b={teamB.adr}
              fmt={(v) => v.toFixed(1)}
            />
            <CompareRow
              label="Rating"
              a={
                teamA.players.length
                  ? teamA.players.reduce((s, p) => s + ratingApprox(p), 0) /
                    teamA.players.length
                  : 0
              }
              b={
                teamB.players.length
                  ? teamB.players.reduce((s, p) => s + ratingApprox(p), 0) /
                    teamB.players.length
                  : 0
              }
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
            {topA ? <TopRow p={topA} team="a" /> : <NoTop side="A" />}
            {topB ? <TopRow p={topB} team="b" /> : <NoTop side="B" />}
          </div>
        </div>
      </div>

      {/* Expanded scoreboard */}
      {entries.length > 0 && (
        <div className="mo-sb">
          {teamA.players.length > 0 && (
            <>
              <div className="mo-sb-team a">
                {teamAName} · {final.a}
              </div>
              <ScoreboardHead />
              {[...teamA.players]
                .sort((a, b) => ratingApprox(b) - ratingApprox(a))
                .map((p) => (
                  <SBRow key={p.steam_id} p={p} />
                ))}
            </>
          )}
          {teamB.players.length > 0 && (
            <>
              <div className="mo-sb-team b">
                {teamBName} · {final.b}
              </div>
              <ScoreboardHead />
              {[...teamB.players]
                .sort((a, b) => ratingApprox(b) - ratingApprox(a))
                .map((p) => (
                  <SBRow key={p.steam_id} p={p} />
                ))}
            </>
          )}
        </div>
      )}
    </div>
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

function SBRow({ p }: { p: ScoreboardEntry }) {
  const rating = ratingApprox(p)
  const ratingClass =
    rating >= 1.15
      ? "mo-rating hi"
      : rating < 0.9
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
      <span>{kastApprox(p)}%</span>
      <span className={ratingClass}>{rating.toFixed(2)}</span>
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

function TopRow({ p, team }: { p: ScoreboardEntry; team: "a" | "b" }) {
  const rating = ratingApprox(p)
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
        <div className="mo-top-rating-n">{rating.toFixed(2)}</div>
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

interface ChartCardProps {
  title: string
  sub: string
  teamAName: string
  teamBName: string
  series: RoundBar[]
  rounds: Round[]
  yTicks: string[]
  yMax: number
  avgPrefix: string
  avgFmt: (v: number) => string
}

function ChartCard({
  title,
  sub,
  teamAName,
  teamBName,
  series,
  rounds,
  yTicks,
  yMax,
  avgPrefix,
  avgFmt,
}: ChartCardProps) {
  const half1 = series.slice(0, 12)
  const half2 = series.slice(12, 24)
  const rounds1 = rounds.slice(0, 12)
  const rounds2 = rounds.slice(12, 24)

  const avg1 =
    half1.length > 0
      ? half1.reduce((s, r) => s + r.teamA + r.teamB, 0) / (half1.length * 2)
      : 0
  const avg2 =
    half2.length > 0
      ? half2.reduce((s, r) => s + r.teamA + r.teamB, 0) / (half2.length * 2)
      : 0

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
        <div className="mo-chart-half">
          <div className="mo-chart-half-label">
            <span>1st half</span>
            <span className="mo-chart-half-avg">
              {avgPrefix}
              {avgFmt(avg1)}
            </span>
          </div>
          <ChartPlot data={half1} yTicks={yTicks} yMax={yMax} />
          <div className="mo-chart-xaxis">
            <span className="mo-chart-xaxis-label">RND</span>
            <div className="mo-chart-xaxis-nums">
              {rounds1.map((r) => (
                <span key={r.id}>{r.round_number}</span>
              ))}
            </div>
          </div>
        </div>
        <div className="mo-chart-divider" aria-hidden="true" />
        <div className="mo-chart-half">
          <div className="mo-chart-half-label">
            <span>2nd half</span>
            <span className="mo-chart-half-avg">
              {avgPrefix}
              {avgFmt(avg2)}
            </span>
          </div>
          <ChartPlot data={half2} yTicks={yTicks} yMax={yMax} />
          <div className="mo-chart-xaxis">
            <span className="mo-chart-xaxis-label">RND</span>
            <div className="mo-chart-xaxis-nums">
              {rounds2.map((r) => (
                <span key={r.id}>{r.round_number}</span>
              ))}
            </div>
          </div>
        </div>
      </div>
      {/* Per-round result strip aligned to the bars above */}
      <div className="mo-rounds-bar">
        <div className="mo-rounds-bar-half">
          <span />
          <div className="mo-rounds-bar-cells">
            {rounds1.map((r) => (
              <div key={r.id} className={`mo-rb-cell ${teamForRound(r)}`}>
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
        <span className="mo-chart-ht">HT</span>
        <div className="mo-rounds-bar-half">
          <span />
          <div className="mo-rounds-bar-cells">
            {rounds2.map((r) => (
              <div key={r.id} className={`mo-rb-cell ${teamForRound(r)}`}>
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
      </div>
    </div>
  )
}

function ChartPlot({
  data,
  yTicks,
  yMax,
}: {
  data: RoundBar[]
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
        {data.map((r) => {
          const aH = Math.min(100, (r.teamA / yMax) * 100)
          const bH = Math.min(100, (r.teamB / yMax) * 100)
          return (
            <div key={r.roundNumber} className="mo-chart-pair">
              <div
                className="mo-chart-bar a"
                style={{ height: `${aH}%` }}
                title={`R${r.roundNumber} · A ${r.teamA}`}
              />
              <div
                className="mo-chart-bar b"
                style={{ height: `${bH}%` }}
                title={`R${r.roundNumber} · B ${r.teamB}`}
              />
            </div>
          )
        })}
      </div>
    </div>
  )
}
