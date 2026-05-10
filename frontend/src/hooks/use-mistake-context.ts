import { useQuery } from "@tanstack/react-query"
import { GetMistakeContext } from "@wailsjs/go/main/App"
import type { MistakeContext } from "@/types/mistake"

// Deep-detail variant of useMistakeTimeline. Returns the surrounding round
// window so the analysis-detail card can render the play without scanning the
// rounds collection client-side.
//
// id may be null when no row is selected — disables the query in that case.
export function useMistakeContext(id: number | null) {
  return useQuery({
    queryKey: ["mistake-context", id],
    queryFn: () =>
      GetMistakeContext(id!) as unknown as Promise<MistakeContext | null>,
    enabled: id != null,
    staleTime: Infinity,
  })
}
