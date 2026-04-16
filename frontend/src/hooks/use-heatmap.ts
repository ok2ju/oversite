import { useQuery } from "@tanstack/react-query"
import {
  GetHeatmapData,
  GetUniqueWeapons,
  GetUniquePlayers,
  GetWeaponStats,
} from "@wailsjs/go/main/App"
import type { HeatmapPoint, PlayerInfo, WeaponStat } from "@/types/heatmap"

export function useHeatmapData(
  demoIDs: number[],
  weapons: string[],
  playerSteamID: string,
  side: string,
) {
  return useQuery({
    queryKey: ["heatmap", demoIDs, weapons, playerSteamID, side],
    queryFn: () =>
      GetHeatmapData(demoIDs, weapons, playerSteamID, side) as Promise<
        HeatmapPoint[]
      >,
    enabled: demoIDs.length > 0,
    staleTime: Infinity,
  })
}

export function useUniqueWeapons(demoIDs: number[]) {
  return useQuery({
    queryKey: ["heatmap-weapons", demoIDs],
    queryFn: () => GetUniqueWeapons(demoIDs) as Promise<string[]>,
    enabled: demoIDs.length > 0,
    staleTime: Infinity,
  })
}

export function useUniquePlayers(demoIDs: number[]) {
  return useQuery({
    queryKey: ["heatmap-players", demoIDs],
    queryFn: () => GetUniquePlayers(demoIDs) as Promise<PlayerInfo[]>,
    enabled: demoIDs.length > 0,
    staleTime: Infinity,
  })
}

export function useWeaponStats(demoId: string | null) {
  return useQuery({
    queryKey: ["weapon-stats", demoId],
    queryFn: () => GetWeaponStats(demoId!) as Promise<WeaponStat[]>,
    enabled: !!demoId,
    staleTime: Infinity,
  })
}
