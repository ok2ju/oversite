import { useMutation, useQueryClient } from "@tanstack/react-query"
import { RecomputeAnalysis } from "@wailsjs/go/main/App"

interface RecomputeArgs {
  demoId: string
}

// Re-runs the full parse-and-analyze pipeline for an already-imported demo
// (legacy backfill). On success, invalidates the analysis-status,
// mistakes-timeline, player-analysis, and player-round-analysis queries so
// the viewer panel and the standalone analysis page both re-read the freshly
// persisted rows.
export function useRecomputeAnalysis() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ demoId }: RecomputeArgs) =>
      RecomputeAnalysis(demoId) as Promise<void>,
    onSuccess: (_data, { demoId }) => {
      // Partial-key invalidation: TanStack matches by prefix, so passing the
      // (key, demoId) pair invalidates every cached entry that starts with
      // that prefix — covering the per-(demo, steamId) variants used by
      // useMistakeTimeline (["mistakes", demoId, steamId]),
      // usePlayerAnalysis (["player-analysis", demoId, steamId]), and
      // usePlayerRoundAnalysis
      // (["player-round-analysis", demoId, steamId]).
      qc.invalidateQueries({ queryKey: ["analysis-status", demoId] })
      qc.invalidateQueries({ queryKey: ["mistakes", demoId] })
      qc.invalidateQueries({ queryKey: ["player-analysis", demoId] })
      qc.invalidateQueries({ queryKey: ["player-round-analysis", demoId] })
      qc.invalidateQueries({ queryKey: ["contact-moments", demoId] })
    },
  })
}
