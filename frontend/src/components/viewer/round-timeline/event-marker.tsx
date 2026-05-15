import { useCallback } from "react"
import { useViewerStore } from "@/stores/viewer"
import { HEADSHOT_ICON_PATH } from "@/lib/viewer/weapon-icons"
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"
import { formatElapsedTime } from "@/lib/viewer/timeline-utils"
import { sideChipClass, sideLabel } from "./side-styles"
import type { TimelineEvent } from "@/lib/timeline/types"

interface EventMarkerProps {
  event: TimelineEvent
  // Position 0..1 inside the lane.
  position: number
  // For grenades that have a known detonate tick, draw a faint duration bar
  // from this position to detonateRight (0..1).
  detonateRight?: number
  roundStartTick: number
  tickRate: number
}

function describeEvent(event: TimelineEvent): {
  label: string
  detail: string
} {
  const src = event.source
  switch (event.kind) {
    case "kill":
      return {
        label: `${src.attacker_name || "?"} → ${src.victim_name || "?"}`,
        detail: `${src.weapon ?? "kill"}${event.headshot ? " · HS" : ""}`,
      }
    case "grenade":
      return {
        label: src.weapon ?? "Grenade",
        detail: `Thrown by ${src.attacker_name || "?"}`,
      }
    case "bomb_plant":
      return {
        label: "Bomb plant",
        detail: `${src.attacker_name || "?"}`,
      }
    case "bomb_defuse":
      return {
        label: "Bomb defuse",
        detail: `${src.attacker_name || "?"}`,
      }
    case "player_hurt":
      return {
        label: `Damage: ${src.health_damage} HP`,
        detail: `${src.attacker_name || "?"} → ${src.victim_name || "?"}${
          src.weapon ? ` · ${src.weapon}` : ""
        }`,
      }
    case "player_flashed":
      return {
        label: "Flashed",
        detail: `By ${src.attacker_name || "?"}`,
      }
    default:
      return { label: event.kind, detail: "" }
  }
}

export function EventMarker({
  event,
  position,
  detonateRight,
  roundStartTick,
  tickRate,
}: EventMarkerProps) {
  const setTick = useViewerStore((s) => s.setTick)
  const pause = useViewerStore((s) => s.pause)
  const handleClick = useCallback(() => {
    pause()
    setTick(event.tick)
  }, [pause, setTick, event.tick])

  const { label, detail } = describeEvent(event)
  const clock = formatElapsedTime(event.tick - roundStartTick, tickRate)
  const useDot = event.kind === "player_hurt"
  const chip = sideChipClass(event.side)
  // Bomb events earn a thicker ring so plant/defuse stand out against the
  // bomb-window accent strip drawn behind them.
  const isBomb = event.kind === "bomb_plant" || event.kind === "bomb_defuse"

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          data-testid={`event-marker-${event.id}`}
          data-kind={event.kind}
          data-side={event.side}
          onClick={handleClick}
          aria-label={`${sideLabel(event.side)} · ${label} at ${clock}`}
          className={cn(
            "group absolute top-1/2 -translate-x-1/2 -translate-y-1/2 cursor-pointer rounded-full p-[3px] ring-1 ring-inset transition-transform hover:z-20 hover:scale-125 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-400/50",
            chip,
            isBomb && "ring-[1.5px]",
          )}
          style={{ left: `${position * 100}%` }}
        >
          {useDot ? (
            <span
              aria-hidden="true"
              className="block h-1.5 w-1.5 rounded-full bg-red-400/90"
            />
          ) : event.iconPath ? (
            <span className="relative block h-3.5 w-3.5">
              <img
                src={event.iconPath}
                alt=""
                draggable={false}
                className="h-3.5 w-3.5 select-none object-contain drop-shadow-[0_0_2px_rgba(0,0,0,0.7)]"
              />
              {event.headshot ? (
                <img
                  src={HEADSHOT_ICON_PATH}
                  alt=""
                  aria-hidden="true"
                  draggable={false}
                  className="absolute -right-0.5 -top-0.5 h-2 w-2 select-none object-contain"
                />
              ) : null}
            </span>
          ) : (
            <span
              aria-hidden="true"
              className="block h-2 w-2 rounded-full bg-white/70"
            />
          )}
          {/* Faint duration bar for smokes / fires — informative, not interactive. */}
          {detonateRight !== undefined ? (
            <span
              aria-hidden="true"
              className="pointer-events-none absolute left-1/2 top-1/2 h-px -translate-y-1/2 bg-white/15"
              style={{
                width: `calc((${detonateRight - position}) * var(--lane-width, 100%))`,
                // Fallback for environments without --lane-width.
                minWidth: 0,
              }}
            />
          ) : null}
        </button>
      </TooltipTrigger>
      <TooltipContent side="top" align="center">
        <div className="font-semibold">
          <span className="hud-callsign mr-1 text-[9px] tracking-wider opacity-80">
            {sideLabel(event.side)}
          </span>
          {label}
        </div>
        {detail ? <div className="text-white/70">{detail}</div> : null}
        <div className="text-white/50">@ {clock}</div>
      </TooltipContent>
    </Tooltip>
  )
}
