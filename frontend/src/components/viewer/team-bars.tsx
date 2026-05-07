import { useMemo } from "react"
import {
  Bomb,
  Cloud,
  Flame,
  HardHat,
  KeyRound,
  Shield,
  Sparkles,
  Swords,
  Zap,
} from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import { useRounds } from "@/hooks/use-rounds"
import { useRoundRoster } from "@/hooks/use-roster"
import { useLoadoutSnapshot } from "@/hooks/use-loadout-snapshot"
import { cn } from "@/lib/utils"
import type { Round } from "@/types/round"
import type { TickData } from "@/types/demo"
import type { PlayerRosterEntry, TeamSide } from "@/types/roster"

interface PlayerLoadout {
  steamId: string
  name: string
  data: TickData | null
}

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
  side: TeamSide,
): PlayerLoadout[] {
  if (!roster) return []
  return roster
    .filter((r) => r.team_side === side)
    .map((r) => ({
      steamId: r.steam_id,
      name: r.player_name,
      data: loadouts[r.steam_id] ?? null,
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

  const ctPlayers = useMemo(
    () => joinRoster(roster, loadouts, "CT"),
    [roster, loadouts],
  )
  const tPlayers = useMemo(
    () => joinRoster(roster, loadouts, "T"),
    [roster, loadouts],
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

function PlayerRow({
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
  const weapon = pickPrimary(data)

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
        <LoadoutIcons data={data} side={side} />
      </div>
    </div>
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

function HealthRow({ data, side }: { data: TickData | null; side: TeamSide }) {
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
}

function LoadoutIcons({
  data,
  side,
}: {
  data: TickData | null
  side: TeamSide
}) {
  if (!data) return null
  const flashCount = countInventory(data.inventory, isFlash)
  const reverse = side === "T"
  return (
    <div
      className={cn(
        "flex items-center gap-1",
        reverse ? "flex-row-reverse" : "flex-row",
      )}
    >
      {data.has_helmet && <HardHat className="h-3 w-3 text-white/80" />}
      {data.has_defuser && <KeyRound className="h-3 w-3 text-emerald-400" />}
      {hasWeapon(data.inventory, "Kevlar Vest") && !data.has_helmet && (
        <Shield className="h-3 w-3 text-white/60" />
      )}
      {hasWeapon(data.inventory, "C4") && (
        <Bomb className="h-3 w-3 text-amber-300" />
      )}
      {hasWeapon(data.inventory, "Zeus x27") && (
        <Zap className="h-3 w-3 text-cyan-300" />
      )}
      {hasWeapon(data.inventory, "Smoke Grenade") && (
        <Cloud className="h-3 w-3 text-white/80" />
      )}
      {flashCount > 0 && (
        <span className="flex items-center gap-0.5 text-white/80">
          <Sparkles className="h-3 w-3" />
          {flashCount > 1 && (
            <span className="text-[10px] leading-none">×{flashCount}</span>
          )}
        </span>
      )}
      {hasWeapon(data.inventory, "HE Grenade") && (
        <Bomb className="h-3 w-3 text-rose-400" />
      )}
      {(hasWeapon(data.inventory, "Molotov") ||
        hasWeapon(data.inventory, "Incendiary Grenade")) && (
        <Flame className="h-3 w-3 text-orange-400" />
      )}
      {hasWeapon(data.inventory, "Decoy Grenade") && (
        <Cloud className="h-3 w-3 text-purple-300" />
      )}
    </div>
  )
}

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

function pickPrimary(data: TickData | null): string | null {
  if (!data) return null
  for (const w of data.inventory) {
    if (PRIMARY_WEAPONS.has(w)) return w
  }
  for (const w of data.inventory) {
    if (SECONDARY_WEAPONS.has(w)) return w
  }
  if (data.weapon && data.weapon !== "Knife") return data.weapon
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

function WeaponIcon({ name, className }: { name: string; className?: string }) {
  if (PRIMARY_WEAPONS.has(name) || SECONDARY_WEAPONS.has(name)) {
    return <Swords className={className} />
  }
  return <Swords className={className} />
}
