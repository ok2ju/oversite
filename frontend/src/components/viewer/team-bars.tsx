import { memo, useMemo } from "react"
import { Shield } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { useRoundRoster } from "@/hooks/use-roster"
import { useLoadoutSnapshot } from "@/hooks/use-loadout-snapshot"
import { useRoundLoadouts } from "@/hooks/use-round-loadouts"
import { cn } from "@/lib/utils"
import type { Round } from "@/types/round"
import type { TickData } from "@/types/demo"
import type { PlayerRosterEntry, TeamSide } from "@/types/roster"
import { WeaponIcon } from "./weapon-icon"

interface PlayerLoadout {
  steamId: string
  name: string
  data: TickData | null
  // inventory comes from round_loadouts (migration 011) — captured at
  // freeze-end and stable across the round, separate from the per-tick
  // mutable fields on `data`.
  inventory: string[]
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
  side: TeamSide,
): PlayerLoadout[] {
  if (!roster) return []
  return roster
    .filter((r) => r.team_side === side)
    .map((r) => ({
      steamId: r.steam_id,
      name: r.player_name,
      data: loadouts[r.steam_id] ?? null,
      inventory: inventories?.[r.steam_id] ?? EMPTY_INVENTORY,
    }))
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
  const activeInventories = activeRound
    ? roundLoadouts?.[activeRound.round_number]
    : undefined

  const ctPlayers = useMemo(
    () => joinRoster(roster, loadouts, activeInventories, "CT"),
    [roster, loadouts, activeInventories],
  )
  const tPlayers = useMemo(
    () => joinRoster(roster, loadouts, activeInventories, "T"),
    [roster, loadouts, activeInventories],
  )

  if (!demoId || !roster) return null

  return (
    <>
      <div
        data-testid="team-bar-ct"
        className="pointer-events-none absolute left-2 top-1/2 z-10 flex w-[200px] -translate-y-1/2 flex-col gap-1"
      >
        {ctPlayers.map((p) => (
          <PlayerRow key={p.steamId} player={p} side="CT" />
        ))}
      </div>
      <div
        data-testid="team-bar-t"
        className="pointer-events-none absolute right-2 top-1/2 z-10 flex w-[200px] -translate-y-1/2 flex-col gap-1"
      >
        {tPlayers.map((p) => (
          <PlayerRow key={p.steamId} player={p} side="T" />
        ))}
      </div>
    </>
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
  const isCt = side === "CT"
  const tone = isCt ? "bg-sky-500" : "bg-orange-500"
  const align = isCt ? "items-start text-left" : "items-end text-right"
  const moneyText = data ? `$${data.money}` : ""
  const weapon = pickPrimary(data, player.inventory)

  return (
    <div
      data-testid={`team-bar-row-${player.steamId}`}
      className={cn(
        "rounded-sm border border-black/40 bg-black/60 text-xs text-white shadow",
        !isAlive && "opacity-40 grayscale",
      )}
    >
      <div
        className={cn(
          "flex items-center gap-2 px-2 py-1",
          tone,
          isCt ? "flex-row" : "flex-row-reverse",
        )}
      >
        {weapon ? (
          <WeaponIcon name={weapon} className="h-3.5 w-3.5 text-white/80" />
        ) : (
          <span className="h-3.5 w-3.5" />
        )}
        <span
          className={cn(
            "flex-1 truncate font-semibold",
            isCt ? "text-left" : "text-right",
          )}
        >
          {player.name}
        </span>
      </div>
      <WeaponLabelRow data={data} side={side} />
      <HealthRow data={data} side={side} />
      <div
        className={cn(
          "flex items-center justify-between gap-2 px-2 py-0.5",
          align,
        )}
      >
        <span className={cn("font-mono text-emerald-400", !isCt && "order-2")}>
          {moneyText}
        </span>
        <LoadoutIcons data={data} inventory={player.inventory} side={side} />
      </div>
    </div>
  )
})

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

const WeaponLabelRow = memo(function WeaponLabelRow({
  data,
  side,
}: {
  data: TickData | null
  side: TeamSide
}) {
  if (!data || !data.is_alive || !data.weapon) return null
  const isCt = side === "CT"
  return (
    <div
      data-testid={`team-bar-weapon-${data.steam_id}`}
      className={cn(
        "flex items-center px-2 py-0.5",
        isCt ? "flex-row justify-start" : "flex-row-reverse justify-start",
      )}
    >
      <WeaponIcon name={data.weapon} className="h-3 w-auto opacity-90" />
    </div>
  )
})

const HealthRow = memo(function HealthRow({
  data,
  side,
}: {
  data: TickData | null
  side: TeamSide
}) {
  if (!data) return null
  const health = Math.max(0, Math.min(100, data.health))
  const armor = Math.max(0, Math.min(100, data.armor))
  const isCt = side === "CT"
  const fillPct = `${health}%`

  return (
    <div
      data-testid={`team-bar-health-${data.steam_id}`}
      className={cn(
        "flex items-center gap-2 px-2 py-1",
        isCt ? "flex-row" : "flex-row-reverse",
      )}
    >
      <span
        className={cn(
          "w-7 shrink-0 font-mono tabular-nums",
          getHealthTextClass(health),
          isCt ? "text-left" : "text-right",
        )}
      >
        {health}
      </span>
      <div className="relative h-1.5 flex-1 overflow-hidden rounded-sm bg-zinc-800/80">
        <div
          data-testid={`team-bar-health-fill-${data.steam_id}`}
          className={cn(
            "h-full transition-[width] duration-150 ease-out",
            getHealthBarClass(health),
            !isCt && "ml-auto",
          )}
          style={{ width: fillPct }}
        />
      </div>
      <div
        className={cn(
          "flex w-9 shrink-0 items-center gap-0.5 font-mono tabular-nums text-white/70",
          isCt ? "flex-row justify-end" : "flex-row-reverse justify-end",
        )}
        title={`Armor ${armor}`}
      >
        <Shield
          className={cn(
            "h-3 w-3",
            armor > 0 ? "text-sky-300" : "text-white/30",
          )}
        />
        <span className="text-[10px]">{armor}</span>
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
  side,
}: {
  data: TickData | null
  inventory: string[]
  side: TeamSide
}) {
  if (!data) return null
  const flashCount = countInventory(inventory, isFlash)
  const reverse = side === "T"
  return (
    <div
      className={cn(
        "flex items-center gap-1",
        reverse ? "flex-row-reverse" : "flex-row",
      )}
    >
      {data.has_helmet && <EquipmentIcon path="/equipment/helmet.svg" />}
      {data.has_defuser && <EquipmentIcon path="/equipment/defuser.svg" />}
      {hasWeapon(inventory, "Kevlar Vest") && !data.has_helmet && (
        <EquipmentIcon path="/equipment/kevlar.svg" />
      )}
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
