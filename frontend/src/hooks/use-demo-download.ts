import { useMutation, useQueryClient } from "@tanstack/react-query"
import { ImportMatchDemo } from "@wailsjs/go/main/App"

export function useDemoDownload() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (faceitMatchID: string) => ImportMatchDemo(faceitMatchID),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["demos"] })
      queryClient.invalidateQueries({ queryKey: ["faceit-matches"] })
    },
  })
}
