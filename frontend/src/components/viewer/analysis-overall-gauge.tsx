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

  return (
    <div
      data-testid="analysis-overall-gauge"
      className="text-sm font-semibold text-white"
    >
      Overall: {data.overall_score}/100
    </div>
  )
}
