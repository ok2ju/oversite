import { memo, useMemo } from "react"
import { Shield } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { useRoundRoster } from "@/hooks/use-roster"
import { useLoadoutSnapshot } from "@/hooks/use-loadout-snapshot"
import { useRoundLoadouts } from "@/hooks/use-round-loadouts"
import { useLiveKDA, type LiveKDAMap } from "@/hooks/use-live-kda"
import { cn } from "@/lib/utils"
import type { Round } from "@/types/round"
import type { TickData } from "@/types/demo"
import type { PlayerRosterEntry, TeamSide } from "@/types/roster"
import { WeaponIcon } from "./weapon-icon"

interface PlayerLoadout {
  steamId: string
  name: string
  data: TickData | null
  // inventory is the player's live weapon list at the current tick (migration
  // 023). When the live tick doesn't carry one (pre-023 demos backfill to '')
  // we fall back to the round-scoped freeze-end loadout — same display as
  // before, just stale for that demo.
  inventory: string[]
  kills: number
  assists: number
  deaths: number
}

const EMPTY_INVENTORY: string[] = []

function getActiveRoundIndex(
  rounds: Round[] | undefined,
  currentTick: number,
): number {
  if (!rounds?.length) return 0
  for (let i = rounds.length - 1; i >= 0; i--) {
    if (currentTick >= rounds[i].start_tick) return i
  }
  return 0
}

function joinRoster(
  roster: PlayerRosterEntry[] | undefined,
  loadouts: Record<string, TickData>,
  inventories: Record<string, string[]> | undefined,
  kda: LiveKDAMap,
  side: TeamSide,
): PlayerLoadout[] {
  if (!roster) return []
  // Sort by steam_id so player cards keep the same vertical order across
  // rounds. The Go ingest path builds per-round stats from a map (random
  // iteration order) and GetPlayerRoundsByRoundID has no ORDER BY, so the
  // roster array arrives in a different order each round without this sort.
  return roster
    .filter((r) => r.team_side === side)
    .slice()
    .sort((a, b) =>
      a.steam_id < b.steam_id ? -1 : a.steam_id > b.steam_id ? 1 : 0,
    )
    .map((r) => {
      const data = loadouts[r.steam_id] ?? null
      const live = data?.inventory
      const inventory =
        live && live.length > 0
          ? live
          : (inventories?.[r.steam_id] ?? EMPTY_INVENTORY)
      const k = kda[r.steam_id]
      return {
        steamId: r.steam_id,
        name: r.player_name,
        data,
        inventory,
        kills: k?.kills ?? 0,
        assists: k?.assists ?? 0,
        deaths: k?.deaths ?? 0,
      }
    })
}

function teamLabel(
  clanName: string,
  entries: PlayerRosterEntry[] | undefined,
  side: TeamSide,
): string {
  if (clanName) return clanName
  const first = entries?.find((e) => e.team_side === side)
  if (first) return `team_${first.player_name}`
  return side
}

export function TeamBars() {
  const demoId = useViewerStore((s) => s.demoId)
  const currentTick = useViewerStore((s) => s.currentTick)

  const { data: rounds } = useRounds(demoId)
  const activeRound = useMemo(() => {
    if (!rounds?.length) return null
    return rounds[getActiveRoundIndex(rounds, currentTick)]
  }, [rounds, currentTick])

  const { data: roster } = useRoundRoster(
    demoId,
    activeRound?.round_number ?? null,
  )

  const loadouts = useLoadoutSnapshot()
  const { data: roundLoadouts } = useRoundLoadouts(demoId)
  const kda = useLiveKDA()
  const activeInventories = activeRound
    ? roundLoadouts?.[activeRound.round_number]
    : undefined

  const ctPlayers = useMemo(
    () => joinRoster(roster, loadouts, activeInventories, kda, "CT"),
    [roster, loadouts, activeInventories, kda],
  )
  const tPlayers = useMemo(
    () => joinRoster(roster, loadouts, activeInventories, kda, "T"),
    [roster, loadouts, activeInventories, kda],
  )

  if (!demoId || !roster) return null

  const tTeamName = teamLabel(activeRound?.t_team_name ?? "", roster, "T")
  const ctTeamName = teamLabel(activeRound?.ct_team_name ?? "", roster, "CT")

  return (
    <>
      <div
        data-testid="team-bar-t"
        className="pointer-events-none absolute left-4 top-[60px] z-10 flex w-[230px] flex-col gap-1"
      >
        <SectionLabel side="T" team={tTeamName} />
        {tPlayers.map((p) => (
          <PlayerRow key={p.steamId} player={p} side="T" />
        ))}
      </div>
      <div
        data-testid="team-bar-ct"
        className="pointer-events-none absolute right-4 top-[60px] z-10 flex w-[230px] flex-col gap-1"
      >
        <SectionLabel side="CT" team={ctTeamName} />
        {ctPlayers.map((p) => (
          <PlayerRow key={p.steamId} player={p} side="CT" />
        ))}
      </div>
    </>
  )
}

function SectionLabel({ side, team }: { side: TeamSide; team: string }) {
  const isT = side === "T"
  return (
    <div
      data-testid={`team-bar-label-${side.toLowerCase()}`}
      className="flex items-baseline gap-1.5 pb-1.5 pl-0.5 text-[11px]"
    >
      <span
        className={cn(
          "font-semibold tracking-wide",
          isT ? "text-amber-400" : "text-sky-400",
        )}
      >
        {side}
      </span>
      <span className="text-white/30">·</span>
      <span className="text-white/55">{team}</span>
    </div>
  )
}

const PlayerRow = memo(function PlayerRow({
  player,
  side,
}: {
  player: PlayerLoadout
  side: TeamSide
}) {
  const data = player.data
  const isAlive = data?.is_alive ?? true
  const moneyText = data ? `$${data.money.toLocaleString()}` : ""
  const weapon = pickPrimary(data, player.inventory)
  const nameColor = side === "T" ? "text-amber-400" : "text-sky-400"

  return (
    <div
      data-testid={`team-bar-row-${player.steamId}`}
      className={cn(
        "relative overflow-hidden rounded-md border border-white/[0.06] bg-[#15181D] px-3 py-2.5 text-xs text-white",
        !isAlive && "opacity-40 grayscale",
      )}
    >
      <div className="flex items-center gap-2">
        {weapon ? (
          <WeaponIcon name={weapon} className="h-3.5 w-auto text-white/60" />
        ) : (
          <span className="h-3.5 w-4" />
        )}
        <span
          className={cn(
            "flex-1 truncate text-[12.5px] font-semibold leading-none",
            nameColor,
          )}
        >
          {player.name}
        </span>
        <KDAStat
          kills={player.kills}
          assists={player.assists}
          deaths={player.deaths}
        />
      </div>
      <HealthRow data={data} />
      <div className="mt-2 flex items-center gap-1.5">
        <LoadoutIcons data={data} inventory={player.inventory} />
        <span
          className={cn(
            "ml-auto text-[11px] font-medium tabular-nums",
            data && data.money > 0 ? "text-white/75" : "text-white/30",
          )}
        >
          {moneyText}
        </span>
      </div>
    </div>
  )
})

function KDAStat({
  kills,
  assists,
  deaths,
}: {
  kills: number
  assists: number
  deaths: number
}) {
  return (
    <span
      data-testid="team-bar-kda"
      className="flex items-baseline gap-1 text-[12px] leading-none tabular-nums"
    >
      <span className="font-semibold text-white">{kills}</span>
      <span className="text-white/25">/</span>
      <span className="text-white/55">{assists}</span>
      <span className="text-white/25">/</span>
      <span className="text-white/55">{deaths}</span>
    </span>
  )
}

function getHealthBarClass(health: number): string {
  if (health > 60) return "bg-emerald-500"
  if (health > 30) return "bg-yellow-500"
  return "bg-red-500"
}

function getHealthTextClass(health: number): string {
  if (health > 60) return "text-emerald-400"
  if (health > 30) return "text-yellow-400"
  return "text-red-400"
}

const HealthRow = memo(function HealthRow({ data }: { data: TickData | null }) {
  if (!data) return null
  const health = Math.max(0, Math.min(100, data.health))
  const armor = Math.max(0, Math.min(100, data.armor))
  const fillPct = `${health}%`

  return (
    <div
      data-testid={`team-bar-health-${data.steam_id}`}
      className="mt-2 flex items-center gap-2"
    >
      <span
        className={cn(
          "w-8 shrink-0 text-left text-[18px] font-bold leading-none tabular-nums",
          getHealthTextClass(health),
        )}
      >
        {health}
      </span>
      <div className="relative h-[3px] flex-1 overflow-hidden rounded-full bg-white/[0.06]">
        <div
          data-testid={`team-bar-health-fill-${data.steam_id}`}
          className={cn(
            "h-full rounded-full transition-[width] duration-150 ease-out",
            getHealthBarClass(health),
          )}
          style={{ width: fillPct }}
        />
      </div>
      <div
        className="flex shrink-0 items-center gap-1 tabular-nums"
        title={data.has_helmet ? `Armor ${armor} + helmet` : `Armor ${armor}`}
      >
        <span
          className={cn(
            "text-[11px] font-medium",
            armor > 0 ? "text-white/75" : "text-white/30",
          )}
        >
          {armor}
        </span>
        <Shield
          className={cn(
            "h-3 w-3",
            armor > 0 ? "text-white/55" : "text-white/25",
          )}
          fill={data.has_helmet && armor > 0 ? "currentColor" : "none"}
        />
      </div>
    </div>
  )
})

const ICON_PROPS = { className: "h-3 w-auto opacity-90" } as const

function EquipmentIcon({ path }: { path: string }) {
  return (
    <img
      src={path}
      alt=""
      draggable={false}
      className="h-3 w-auto select-none object-contain opacity-90"
    />
  )
}

const LoadoutIcons = memo(function LoadoutIcons({
  data,
  inventory,
}: {
  data: TickData | null
  inventory: string[]
}) {
  if (!data) return null
  const flashCount = countInventory(inventory, isFlash)
  return (
    <div className="flex items-center gap-1">
      {data.has_defuser && <EquipmentIcon path="/equipment/defuser.svg" />}
      {hasWeapon(inventory, "C4") && <WeaponIcon name="C4" {...ICON_PROPS} />}
      {hasWeapon(inventory, "Zeus x27") && (
        <WeaponIcon name="Zeus x27" {...ICON_PROPS} />
      )}
      {hasWeapon(inventory, "Smoke Grenade") && (
        <WeaponIcon name="Smoke Grenade" {...ICON_PROPS} />
      )}
      {flashCount > 0 && (
        <span className="flex items-center gap-0.5 text-white/80">
          <WeaponIcon name="Flashbang" {...ICON_PROPS} />
          {flashCount > 1 && (
            <span className="text-[10px] leading-none">×{flashCount}</span>
          )}
        </span>
      )}
      {hasWeapon(inventory, "HE Grenade") && (
        <WeaponIcon name="HE Grenade" {...ICON_PROPS} />
      )}
      {hasWeapon(inventory, "Molotov") && (
        <WeaponIcon name="Molotov" {...ICON_PROPS} />
      )}
      {hasWeapon(inventory, "Incendiary Grenade") && (
        <WeaponIcon name="Incendiary Grenade" {...ICON_PROPS} />
      )}
      {hasWeapon(inventory, "Decoy Grenade") && (
        <WeaponIcon name="Decoy Grenade" {...ICON_PROPS} />
      )}
    </div>
  )
})

const PRIMARY_WEAPONS = new Set([
  "AK-47",
  "M4A4",
  "M4A1",
  "AUG",
  "SG 553",
  "FAMAS",
  "Galil AR",
  "AWP",
  "G3SG1",
  "SCAR-20",
  "SSG 08",
  "MP7",
  "MP9",
  "MP5-SD",
  "P90",
  "MAC-10",
  "UMP-45",
  "PP-Bizon",
  "Nova",
  "XM1014",
  "Sawed-Off",
  "MAG-7",
  "M249",
  "Negev",
])

const SECONDARY_WEAPONS = new Set([
  "Glock-18",
  "USP-S",
  "P2000",
  "P250",
  "Tec-9",
  "Five-SeveN",
  "CZ75 Auto",
  "Desert Eagle",
  "R8 Revolver",
  "Dual Berettas",
])

function pickPrimary(
  data: TickData | null,
  inventory: string[],
): string | null {
  for (const w of inventory) {
    if (PRIMARY_WEAPONS.has(w)) return w
  }
  for (const w of inventory) {
    if (SECONDARY_WEAPONS.has(w)) return w
  }
  if (data?.weapon && data.weapon !== "Knife") return data.weapon
  return null
}

function hasWeapon(inventory: string[], name: string): boolean {
  return inventory.includes(name)
}

function isFlash(name: string): boolean {
  return name === "Flashbang"
}

function countInventory(
  inventory: string[],
  predicate: (name: string) => boolean,
): number {
  let count = 0
  for (const w of inventory) {
    if (predicate(w)) count++
  }
  return count
}
