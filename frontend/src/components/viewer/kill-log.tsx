import { memo, useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useKillFeed } from "@/hooks/use-game-events"
import { cn } from "@/lib/utils"
import { selectVisibleKills, type KillEntry } from "@/lib/viewer/kill-log"
import { HEADSHOT_ICON_PATH } from "@/lib/viewer/weapon-icons"
import { WeaponIcon } from "./weapon-icon"
import type { TeamSide } from "@/types/roster"

const SIDE_COLOR: Record<"CT" | "T" | "unknown", string> = {
  CT: "text-sky-400",
  T: "text-orange-400",
  unknown: "text-white/70",
}

function nameColor(side: TeamSide | null): string {
  return SIDE_COLOR[side ?? "unknown"]
}

export function KillLog() {
  const demoId = useViewerStore((s) => s.demoId)
  const currentTick = useViewerStore((s) => s.currentTick)
  const tickRate = useViewerStore((s) => s.tickRate)
  const { data: events } = useKillFeed(demoId)

  const visible = useMemo(
    () => selectVisibleKills(events, currentTick, { tickRate }),
    [events, currentTick, tickRate],
  )

  if (!demoId || visible.length === 0) return null

  return (
    <div
      data-testid="kill-log"
      className="pointer-events-none absolute right-2 top-2 z-20 flex flex-col items-end gap-1"
    >
      {visible.map((kill) => (
        <KillRow key={kill.id} kill={kill} />
      ))}
    </div>
  )
}

const KillRow = memo(function KillRow({ kill }: { kill: KillEntry }) {
  return (
    <div
      data-testid={`kill-log-row-${kill.id}`}
      className="flex items-center gap-2 rounded-sm bg-black/70 px-2 py-1 text-xs font-semibold tracking-tight text-white shadow-sm"
    >
      <span
        data-testid={`kill-attacker-${kill.id}`}
        className={cn("truncate", nameColor(kill.attackerSide))}
      >
        {kill.attackerName || "?"}
      </span>
      <span className="flex items-center gap-1 text-white/90">
        <WeaponIcon name={kill.weapon} className="h-4 w-auto" />
        {kill.headshot && (
          <img
            src={HEADSHOT_ICON_PATH}
            alt="headshot"
            draggable={false}
            data-testid={`kill-headshot-${kill.id}`}
            className="h-4 w-auto select-none object-contain"
          />
        )}
      </span>
      <span
        data-testid={`kill-victim-${kill.id}`}
        className={cn("truncate", nameColor(kill.victimSide))}
      >
        {kill.victimName || "?"}
      </span>
    </div>
  )
})
