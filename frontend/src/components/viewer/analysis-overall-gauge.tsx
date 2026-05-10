import { useViewerStore } from "@/stores/viewer"
import { usePlayerAnalysis } from "@/hooks/use-analysis"

// Slice-5 "ugly first" overall-score gauge: a single line of plain text. The
// SVG arc, animated count-up, and color tier are explicitly slice 7+. Returns
// null while loading or when the analysis row is the zero value (unknown
// demo / unknown player) so the panel header collapses to nothing instead of
// flashing "Overall: 0/100".
//
// Reads (demoId, selectedPlayerSteamId) from useViewerStore the same way
// MistakeList does (mistake-list.tsx:79-80).
export function AnalysisOverallGauge() {
  const demoId = useViewerStore((s) => s.demoId)
  const steamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const { data, isLoading } = usePlayerAnalysis(demoId, steamId)

  if (isLoading) return null
  if (!data || !data.steam_id) return null

  const score = Math.max(0, Math.min(100, data.overall_score))
  const radius = 16
  const circumference = 2 * Math.PI * radius
  const dash = (score / 100) * circumference
  const tier =
    score >= 75
      ? "stroke-emerald-400"
      : score >= 50
        ? "stroke-amber-400"
        : "stroke-rose-400"
  const tierGlow =
    score >= 75
      ? "drop-shadow(0 0 6px rgba(74,222,128,0.55))"
      : score >= 50
        ? "drop-shadow(0 0 6px rgba(251,191,36,0.55))"
        : "drop-shadow(0 0 6px rgba(251,113,133,0.55))"

  return (
    <div
      data-testid="analysis-overall-gauge"
      className="flex items-center gap-3"
    >
      <div className="relative h-10 w-10 shrink-0">
        <svg viewBox="0 0 40 40" className="h-full w-full -rotate-90">
          <circle
            cx="20"
            cy="20"
            r={radius}
            fill="none"
            stroke="rgba(255,255,255,0.08)"
            strokeWidth="3"
          />
          <circle
            cx="20"
            cy="20"
            r={radius}
            fill="none"
            strokeWidth="3"
            strokeLinecap="round"
            strokeDasharray={`${dash} ${circumference - dash}`}
            className={tier}
            style={{
              filter: tierGlow,
              transition: "stroke-dasharray 320ms ease",
            }}
          />
        </svg>
        <span className="hud-display absolute inset-0 flex items-center justify-center text-[13px] font-semibold leading-none text-white">
          {score}
        </span>
      </div>
      <div className="flex flex-col leading-tight">
        <span className="hud-callsign text-[9px] font-semibold text-white/45">
          Performance
        </span>
        <span className="font-mono text-[11px] tabular-nums text-white/65">
          {`Overall: ${score}/100`}
        </span>
      </div>
    </div>
  )
}
