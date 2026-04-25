import { useMutation, useQueryClient } from "@tanstack/react-query"
import { SyncFaceitMatches } from "@wailsjs/go/main/App"
import { useFaceitStore } from "@/stores/faceit"

export function useFaceitSync() {
  const queryClient = useQueryClient()
  const setLastSyncedAt = useFaceitStore((s) => s.setLastSyncedAt)

  return useMutation({
    mutationFn: () => SyncFaceitMatches(),
    onSuccess: () => {
      setLastSyncedAt(Date.now())
      queryClient.invalidateQueries({ queryKey: ["faceit"] })
      queryClient.invalidateQueries({ queryKey: ["faceit-matches"] })
    },
  })
}
