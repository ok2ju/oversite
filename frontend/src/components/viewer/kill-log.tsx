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
      className="pointer-events-none absolute left-1/2 top-[60px] z-20 flex -translate-x-1/2 flex-col items-center gap-1"
    >
      {visible.map((kill) => (
        <KillRow key={kill.id} kill={kill} />
      ))}
    </div>
  )
}

const KillRow = memo(function KillRow({ kill }: { kill: KillEntry }) {
  // Side-stripe color keyed off the attacker so the eye finds "who fragged"
  // first. Same sky/orange identity used elsewhere in the viewer.
  const stripe =
    kill.attackerSide === "CT"
      ? "before:bg-sky-400"
      : kill.attackerSide === "T"
        ? "before:bg-orange-400"
        : "before:bg-white/40"

  return (
    <div
      data-testid={`kill-log-row-${kill.id}`}
      className={cn(
        "relative flex items-center gap-2 overflow-hidden rounded-[4px] border border-white/[0.06] bg-[#15181D]/95 px-3 py-1 text-xs font-medium tracking-tight text-white backdrop-blur-sm",
        "before:absolute before:left-0 before:top-1/2 before:h-3 before:w-[2px] before:-translate-y-1/2 before:rounded-r before:content-['']",
        stripe,
      )}
    >
      <span
        data-testid={`kill-attacker-${kill.id}`}
        className={cn(
          "hud-callsign truncate text-[11px]",
          nameColor(kill.attackerSide),
        )}
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
            className="h-4 w-auto select-none object-contain drop-shadow-[0_0_4px_rgba(248,113,113,0.7)]"
          />
        )}
      </span>
      <span
        data-testid={`kill-victim-${kill.id}`}
        className={cn(
          "hud-callsign truncate text-[11px] opacity-80",
          nameColor(kill.victimSide),
        )}
      >
        {kill.victimName || "?"}
      </span>
    </div>
  )
})
