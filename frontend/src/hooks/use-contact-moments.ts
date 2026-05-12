import { useQuery } from "@tanstack/react-query"
import { GetContactMoments } from "@wailsjs/go/main/App"
import type { main } from "@wailsjs/go/models"

// Per-(demo, round, player) contacts with their mistakes embedded. The Go
// side computes once at import time and the rows are static for the
// lifetime of an import, so staleTime: Infinity matches
// useMistakeTimeline / useDuelTimeline. Disabled when any of the three
// inputs is null — round mode (no player selected) doesn't fetch contacts.
export function useContactMoments(
  demoId: string | null,
  roundNumber: number | null,
  steamId: string | null,
) {
  return useQuery({
    queryKey: ["contact-moments", demoId, roundNumber, steamId],
    queryFn: () =>
      GetContactMoments(demoId!, roundNumber!, steamId!) as Promise<
        main.ContactMoment[]
      >,
    enabled: !!demoId && roundNumber !== null && !!steamId,
    staleTime: Infinity,
  })
}
