import { useCallback, useState } from "react"
import { useViewerStore } from "@/stores/viewer"
import { HEADSHOT_ICON_PATH } from "@/lib/viewer/weapon-icons"
import { formatElapsedTime } from "@/lib/viewer/timeline-utils"
import { cn } from "@/lib/utils"
import type { EventCluster, TimelineEvent } from "@/lib/timeline/types"

interface EventClusterBadgeProps {
  cluster: EventCluster
  position: number
  roundStartTick: number
  tickRate: number
}

function describeShort(event: TimelineEvent): string {
  const src = event.source
  switch (event.kind) {
    case "kill":
      return `${src.attacker_name || "?"} → ${src.victim_name || "?"}${
        src.weapon ? ` · ${src.weapon}` : ""
      }`
    case "grenade":
      return `${src.weapon ?? "Grenade"} · ${src.attacker_name || "?"}`
    case "bomb_plant":
      return `Plant · ${src.attacker_name || "?"}`
    case "bomb_defuse":
      return `Defuse · ${src.attacker_name || "?"}`
    case "player_hurt":
      return `Damage ${src.health_damage} · ${src.attacker_name || "?"} → ${src.victim_name || "?"}`
    case "player_flashed":
      return `Flashed by ${src.attacker_name || "?"}`
    default:
      return event.kind
  }
}

export function EventClusterBadge({
  cluster,
  position,
  roundStartTick,
  tickRate,
}: EventClusterBadgeProps) {
  const setTick = useViewerStore((s) => s.setTick)
  const pause = useViewerStore((s) => s.pause)
  const [open, setOpen] = useState(false)
  const handleSeek = useCallback(
    (tick: number) => {
      pause()
      setTick(tick)
      setOpen(false)
    },
    [pause, setTick],
  )

  // Render the first 2 icons stacked + a +N count.
  const stack = cluster.events.slice(0, 2)
  const overflow = cluster.events.length - stack.length

  return (
    <div
      data-testid={`event-cluster-${cluster.id}`}
      className="absolute top-1/2 -translate-x-1/2 -translate-y-1/2"
      style={{ left: `${position * 100}%` }}
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={(e) => {
        // Close only when focus leaves the entire cluster (popout included).
        if (!e.currentTarget.contains(e.relatedTarget as Node | null)) {
          setOpen(false)
        }
      }}
    >
      <button
        type="button"
        data-testid={`event-cluster-trigger-${cluster.id}`}
        aria-label={`${cluster.events.length} events`}
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className={cn(
          "flex items-center gap-0.5 rounded-md bg-black/55 px-1 py-0.5 ring-1 ring-inset ring-white/20 backdrop-blur transition-transform hover:scale-110 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-400/50",
        )}
      >
        {stack.map((ev) => (
          <span key={ev.id} className="relative block h-3.5 w-3.5">
            {ev.iconPath ? (
              <img
                src={ev.iconPath}
                alt=""
                draggable={false}
                className="h-3.5 w-3.5 select-none object-contain"
              />
            ) : (
              <span className="block h-2 w-2 rounded-full bg-white/70" />
            )}
            {ev.headshot ? (
              <img
                src={HEADSHOT_ICON_PATH}
                alt=""
                aria-hidden="true"
                draggable={false}
                className="absolute -right-0.5 -top-0.5 h-1.5 w-1.5 select-none object-contain"
              />
            ) : null}
          </span>
        ))}
        {overflow > 0 ? (
          <span className="hud-callsign text-[9px] font-semibold text-white/90">
            +{overflow}
          </span>
        ) : null}
      </button>
      {open ? (
        <div
          role="menu"
          data-testid={`event-cluster-popout-${cluster.id}`}
          className="hud-panel absolute bottom-full left-1/2 z-50 mb-1 w-56 -translate-x-1/2 rounded-md p-1 text-[11px] text-white shadow-xl"
        >
          {cluster.events.map((ev) => (
            <button
              key={ev.id}
              type="button"
              role="menuitem"
              data-testid={`event-cluster-row-${ev.id}`}
              onClick={() => handleSeek(ev.tick)}
              className="flex w-full items-center gap-2 rounded px-1.5 py-1 text-left hover:bg-white/10 focus:outline-none focus-visible:bg-white/10"
            >
              {ev.iconPath ? (
                <img
                  src={ev.iconPath}
                  alt=""
                  draggable={false}
                  className="h-3 w-3 select-none object-contain"
                />
              ) : (
                <span className="h-2 w-2 shrink-0 rounded-full bg-white/70" />
              )}
              <span className="min-w-0 flex-1 truncate">
                {describeShort(ev)}
              </span>
              <span className="tabular-nums text-white/50">
                {formatElapsedTime(ev.tick - roundStartTick, tickRate)}
              </span>
            </button>
          ))}
        </div>
      ) : null}
    </div>
  )
}
