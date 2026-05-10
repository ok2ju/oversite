import { useMemo, useState } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { usePlayerRoundAnalysis } from "@/hooks/use-player-round-analysis"
import { useMistakeTimeline } from "@/hooks/use-mistake-timeline"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

// Layered round-by-round timeline. For each round we render:
//   - a thin "outcome strip" along the bottom (CT-blue or T-orange tint)
//   - the trade-% bar rising from the strip
//   - a stack of severity dots above each bar (one dot per flagged play)
// Hovering a round reveals a quick scoreboard tooltip and reads the round
// number into the analysis store so the mistakes feed below can scroll/filter.
export function MatchTimeline() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const setTick = useViewerStore((s) => s.setTick)
  const { data: rounds, isLoading: roundsLoading } = useRounds(demoId)
  const { data: roundRows } = usePlayerRoundAnalysis(demoId, steamId)
  const { data: mistakes } = useMistakeTimeline(demoId, steamId)
  const [hoverRound, setHoverRound] = useState<number | null>(null)

  const tradeByRound = useMemo(() => {
    const m = new Map<number, number>()
    for (const r of roundRows ?? []) m.set(r.round_number, r.trade_pct)
    return m
  }, [roundRows])

  const mistakesByRound = useMemo(() => {
    const m = new Map<
      number,
      {
        sev: number
        entry: typeof mistakes extends (infer U)[] | undefined ? U : never
      }[]
    >()
    for (const e of mistakes ?? []) {
      const arr = m.get(e.round_number) ?? []
      arr.push({ sev: e.severity || 1, entry: e as never })
      m.set(e.round_number, arr)
    }
    return m
  }, [mistakes])

  if (roundsLoading) {
    return (
      <div data-testid="match-timeline-loading" className="flex flex-col gap-3">
        <Skeleton className="h-44 w-full bg-white/5" />
      </div>
    )
  }

  if (!rounds || rounds.length === 0) {
    return (
      <p
        data-testid="match-timeline-empty"
        className="text-sm text-[var(--text-muted)]"
      >
        No round data
      </p>
    )
  }

  const ctScore = rounds[rounds.length - 1]?.ct_score ?? 0
  const tScore = rounds[rounds.length - 1]?.t_score ?? 0
  const totalFlagged = (mistakes ?? []).length
  const avgTrade =
    roundRows && roundRows.length > 0
      ? Math.round(
          (roundRows.reduce((s, r) => s + r.trade_pct, 0) / roundRows.length) *
            100,
        )
      : 0

  return (
    <section
      data-testid="match-timeline"
      className="rounded-xl border border-[var(--border)] bg-[var(--bg-elevated)] px-6 py-6"
    >
      <header className="mb-5 flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-3 text-[10.5px] font-semibold uppercase tracking-[0.18em] text-[var(--text-subtle)]">
            <span
              aria-hidden="true"
              className="inline-block h-px w-8 bg-[var(--border-strong)]"
            />
            <span>B · Match timeline</span>
          </div>
          <h3
            className="text-[15px] font-semibold leading-tight text-[var(--text)]"
            style={{ fontFamily: "'Inter Tight', Inter, sans-serif" }}
          >
            Round-by-round arc
          </h3>
          <p className="text-[12px] text-[var(--text-muted)]">
            Bar height = trade %. Dots above = flagged plays in that round.
            Hover for the scoreboard, click to seek the viewer to round
            freeze-end.
          </p>
        </div>
        <dl className="flex items-end gap-6 font-mono text-[11px] uppercase tracking-wide text-[var(--text-faint)]">
          <div className="flex flex-col items-end gap-0.5">
            <dt>CT · T</dt>
            <dd className="font-[Antonio] text-2xl font-semibold leading-none text-[var(--text)] tabular-nums">
              {ctScore}–{tScore}
            </dd>
          </div>
          <div className="flex flex-col items-end gap-0.5">
            <dt>Avg trade</dt>
            <dd className="font-[Antonio] text-2xl font-semibold leading-none text-[var(--text)] tabular-nums">
              {avgTrade}%
            </dd>
          </div>
          <div className="flex flex-col items-end gap-0.5">
            <dt>Flagged</dt>
            <dd className="font-[Antonio] text-2xl font-semibold leading-none text-[var(--accent)] tabular-nums">
              {totalFlagged}
            </dd>
          </div>
        </dl>
      </header>

      {/* Y-axis ticks */}
      <div className="relative">
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-x-0 inset-y-[26px] grid grid-rows-4"
        >
          {[100, 75, 50, 25].map((tick) => (
            <div
              key={tick}
              className="flex items-center justify-between border-t border-dashed border-white/[0.06] pr-2 text-[9px] font-mono uppercase text-[var(--text-faint)]"
            >
              <span className="bg-[var(--bg-elevated)] px-1">{tick}%</span>
            </div>
          ))}
        </div>

        {/* Bars */}
        <ol className="relative flex h-44 items-end gap-[3px] pl-1 pr-1">
          {rounds.map((round) => {
            const tradePct = tradeByRound.get(round.round_number) ?? 0
            const heightPct = Math.max(2, Math.min(1, tradePct) * 100)
            const flagged = mistakesByRound.get(round.round_number) ?? []
            const high = flagged.some((m) => m.sev >= 3)
            const med = flagged.some((m) => m.sev === 2)
            const winnerCT = round.winner_side === "CT"
            const isHover = hoverRound === round.round_number
            const accentBar = high
              ? "bg-[#ff5050] group-hover:bg-[#ff7070]"
              : med
                ? "bg-[var(--accent)] group-hover:bg-[var(--accent-hover)]"
                : "bg-[#3a3f48] group-hover:bg-[#4a5058]"
            return (
              <li
                key={round.round_number}
                data-testid={`match-timeline-round-${round.round_number}`}
                className="group relative flex h-full flex-1 flex-col-reverse items-center"
                onMouseEnter={() => setHoverRound(round.round_number)}
                onMouseLeave={() => setHoverRound(null)}
                onClick={() => {
                  if (round.freeze_end_tick > 0) setTick(round.freeze_end_tick)
                }}
              >
                {/* Outcome strip */}
                <span
                  aria-hidden="true"
                  className={cn(
                    "h-1 w-full rounded-sm",
                    winnerCT ? "bg-[#5db1ff]/70" : "bg-[var(--accent)]/70",
                  )}
                />
                {/* Spacer */}
                <span aria-hidden="true" className="h-1" />
                {/* Trade bar */}
                <span
                  aria-hidden="true"
                  className={cn(
                    "w-full cursor-pointer rounded-sm transition-colors",
                    accentBar,
                  )}
                  style={{ height: `${heightPct}%` }}
                />
                {/* Mistake dots */}
                {flagged.length > 0 ? (
                  <span
                    aria-hidden="true"
                    className="mb-1 flex flex-col-reverse gap-[2px]"
                  >
                    {flagged.slice(0, 3).map((m, i) => (
                      <span
                        key={i}
                        className={cn(
                          "h-1.5 w-1.5 rounded-full",
                          m.sev >= 3
                            ? "bg-[#ff5050]"
                            : m.sev === 2
                              ? "bg-[#ffc233]"
                              : "bg-white/40",
                        )}
                      />
                    ))}
                    {flagged.length > 3 ? (
                      <span className="font-mono text-[8px] leading-none text-[var(--text-faint)]">
                        +{flagged.length - 3}
                      </span>
                    ) : null}
                  </span>
                ) : null}

                {/* Tooltip */}
                {isHover ? (
                  <div
                    role="tooltip"
                    className="pointer-events-none absolute -top-2 z-10 -translate-y-full whitespace-nowrap rounded-md border border-[var(--border-strong)] bg-[var(--bg-sunken)] px-2.5 py-1.5 text-[11px] shadow-xl"
                  >
                    <div className="font-mono text-[10px] uppercase tracking-wide text-[var(--text-faint)]">
                      Round {round.round_number} ·{" "}
                      <span
                        className={
                          winnerCT ? "text-[#5db1ff]" : "text-[var(--accent)]"
                        }
                      >
                        {winnerCT ? "CT win" : "T win"}
                      </span>
                    </div>
                    <div className="text-[var(--text)]">
                      Trade{" "}
                      <span className="font-mono tabular-nums">
                        {Math.round(tradePct * 100)}%
                      </span>{" "}
                      ·{" "}
                      <span className="font-mono tabular-nums">
                        {flagged.length}
                      </span>{" "}
                      flagged
                    </div>
                  </div>
                ) : null}
              </li>
            )
          })}
        </ol>
      </div>

      {/* X-axis labels */}
      <div className="mt-2 flex items-center justify-between font-mono text-[10px] uppercase tracking-wide text-[var(--text-faint)]">
        <span>R1</span>
        <span>R{Math.ceil(rounds.length / 2)}</span>
        <span>R{rounds.length}</span>
      </div>

      {/* Legend */}
      <div className="mt-4 flex flex-wrap items-center gap-x-5 gap-y-1.5 border-t border-[var(--divider)] pt-3 text-[10.5px] font-mono uppercase tracking-wide text-[var(--text-muted)]">
        <span className="flex items-center gap-1.5">
          <span className="h-1.5 w-3 rounded-sm bg-[#3a3f48]" />
          Clean round
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-1.5 w-3 rounded-sm bg-[var(--accent)]" />
          Med flag
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-1.5 w-3 rounded-sm bg-[#ff5050]" />
          High flag
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-1 w-3 rounded-sm bg-[#5db1ff]/70" />
          CT won
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-1 w-3 rounded-sm bg-[var(--accent)]/70" />T won
        </span>
      </div>
    </section>
  )
}
