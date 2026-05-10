import { useQuery } from "@tanstack/react-query"
import { GetAnalysisStatus } from "@wailsjs/go/main/App"
import type { AnalysisStatus } from "@/types/analysis"

// Per-demo analysis-availability sentinel. Used by the viewer's mistake-list
// panel to decide between rendering the populated header + list, a shimmer,
// or nothing.
//
// Note: staleTime is intentionally 0 (NOT Infinity, which is the default for
// other analysis hooks like useMistakeTimeline / usePlayerAnalysis). The
// status flips from "missing" → "ready" after a recompute, and we want the
// post-mutation invalidateQueries to trigger a real refetch instead of
// reading a frozen cache entry. The query itself is cheap — a single SQL
// count(*) — so the extra request cost is negligible.
export function useAnalysisStatus(demoId: string | null) {
  return useQuery({
    queryKey: ["analysis-status", demoId],
    queryFn: () => GetAnalysisStatus(demoId!) as Promise<AnalysisStatus>,
    enabled: !!demoId,
    staleTime: 0,
  })
}
