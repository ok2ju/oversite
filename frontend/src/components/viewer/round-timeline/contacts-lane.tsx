import { useCallback } from "react"
import { useViewerStore } from "@/stores/viewer"
import { cn } from "@/lib/utils"
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip"
import { ContactTooltip } from "./contact-tooltip"
import type { ContactMarker } from "@/lib/timeline/types"

interface ContactsLaneProps {
  contacts: ContactMarker[]
  roundStartTick: number
  roundEndTick: number
  // When false, render the "Select a player…" placeholder (round mode).
  hasPlayer: boolean
}

const SEVERITY_CLASS: Record<number, string> = {
  // Severity 0 = clean contact; gray pill so the moment is visible but not
  // read as a problem.
  0: "bg-white/30 ring-white/20",
  1: "bg-yellow-400/70 ring-yellow-300/40",
  2: "bg-orange-400/80 ring-orange-300/40",
  3: "bg-red-500/85 ring-red-300/40",
}

function position(tick: number, start: number, end: number): number {
  const span = Math.max(1, end - start)
  return Math.max(0, Math.min(1, (tick - start) / span))
}

export function ContactsLane({
  contacts,
  roundStartTick,
  roundEndTick,
  hasPlayer,
}: ContactsLaneProps) {
  const tickRate = useViewerStore((s) => s.tickRate)
  const setTick = useViewerStore((s) => s.setTick)
  const pause = useViewerStore((s) => s.pause)

  // Click → pause first, then seek to t_pre (lead-up tick) so the user sees
  // the approach. Pausing first prevents the next tick advance from racing
  // the seek.
  const handleClick = useCallback(
    (c: ContactMarker) => {
      pause()
      setTick(c.tPre)
    },
    [pause, setTick],
  )

  if (!hasPlayer) {
    return (
      <div
        data-testid="round-timeline-contacts-placeholder"
        className="relative flex h-5 items-center justify-center rounded-sm bg-white/[0.02] text-[10px] text-white/40"
      >
        Select a player to see contacts for this round
      </div>
    )
  }

  if (contacts.length === 0) {
    return (
      <div
        data-testid="round-timeline-contacts-empty"
        className="relative flex h-5 items-center justify-center rounded-sm bg-white/[0.02] text-[10px] text-white/40"
      >
        No contacts this round
      </div>
    )
  }

  return (
    <div
      data-testid="round-timeline-contacts"
      className="relative h-5 rounded-sm bg-white/[0.02]"
      aria-label="Contacts timeline"
    >
      <span
        aria-hidden="true"
        className="hud-callsign pointer-events-none absolute left-1.5 top-1/2 -translate-y-1/2 text-[9px] font-semibold tracking-wider text-white/50"
      >
        CONTACTS
      </span>
      {contacts.map((c) => {
        const pos = position(c.tFirst, roundStartTick, roundEndTick)
        const sev = SEVERITY_CLASS[c.worstSeverity] ?? SEVERITY_CLASS[0]
        return (
          <Tooltip key={c.id}>
            <TooltipTrigger asChild>
              <button
                type="button"
                data-testid={`contact-marker-${c.id}`}
                onClick={() => handleClick(c)}
                aria-label={`Contact at ${c.tFirst - roundStartTick} ticks, ${c.outcome}`}
                className={cn(
                  "absolute top-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rotate-45 cursor-pointer rounded-sm ring-1 ring-inset transition-transform hover:scale-125 focus:outline-none focus-visible:ring-2 focus-visible:ring-orange-400/50",
                  sev,
                )}
                style={{ left: `${pos * 100}%` }}
              />
            </TooltipTrigger>
            <TooltipContent side="bottom" align="center" className="max-w-xs">
              <ContactTooltip
                contact={c}
                tickRate={tickRate}
                roundStartTick={roundStartTick}
              />
            </TooltipContent>
          </Tooltip>
        )
      })}
    </div>
  )
}
