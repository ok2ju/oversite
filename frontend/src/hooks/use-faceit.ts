import { useQuery } from "@tanstack/react-query"
import { GetFaceitProfile } from "@wailsjs/go/main/App"
import type { FaceitProfile } from "@/types/faceit"

export function useFaceitProfile() {
  return useQuery({
    queryKey: ["faceit", "profile"],
    queryFn: () => GetFaceitProfile() as Promise<FaceitProfile>,
  })
}
