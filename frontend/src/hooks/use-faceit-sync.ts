import { useMutation, useQueryClient } from "@tanstack/react-query"
import { SyncFaceitMatches } from "@wailsjs/go/main/App"

export function useFaceitSync() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => SyncFaceitMatches(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["faceit"] })
      queryClient.invalidateQueries({ queryKey: ["faceit-matches"] })
    },
  })
}
