import { useQuery } from "@tanstack/react-query"
import { GetDemoByID } from "@wailsjs/go/main/App"
import type { Demo } from "@/types/demo"

export function useDemo(id: string | undefined) {
  return useQuery({
    queryKey: ["demo", id],
    queryFn: () => GetDemoByID(id!) as Promise<Demo>,
    enabled: !!id,
    staleTime: Infinity,
  })
}
