import { useViewerStore } from "@/stores/viewer"
import { EventMarker } from "./event-marker"
import { EventClusterBadge } from "./event-cluster"
import { cn } from "@/lib/utils"
import type { EventCluster } from "@/lib/timeline/types"

interface LaneProps {
  clusters: EventCluster[]
  roundStartTick: number
  roundEndTick: number
  // "ct" or "caused" colors the lane sky; "t" or "affected" colors amber.
  variant: "top" | "bottom"
  side: "team" | "player"
  // Test-only override; viewer reads from store otherwise.
  testid?: string
}

// Compute fractional position 0..1 inside the round window.
function position(tick: number, start: number, end: number): number {
  const span = Math.max(1, end - start)
  return Math.max(0, Math.min(1, (tick - start) / span))
}

export function Lane({
  clusters,
  roundStartTick,
  roundEndTick,
  variant,
  side,
  testid,
}: LaneProps) {
  const tickRate = useViewerStore((s) => s.tickRate)
  const label =
    side === "team"
      ? variant === "top"
        ? "CT"
        : "T"
      : variant === "top"
        ? "Caused"
        : "Affected"

  return (
    <div
      data-testid={testid ?? `round-timeline-lane-${variant}`}
      data-side={side}
      className={cn(
        "relative h-7 rounded-sm",
        variant === "top"
          ? side === "team"
            ? "bg-sky-400/[0.04]"
            : "bg-orange-500/[0.04]"
          : side === "team"
            ? "bg-amber-400/[0.04]"
            : "bg-fuchsia-500/[0.04]",
      )}
      aria-label={`${label} events`}
    >
      <span
        aria-hidden="true"
        className={cn(
          "hud-callsign pointer-events-none absolute left-1.5 top-1/2 -translate-y-1/2 text-[9px] font-semibold tracking-wider",
          variant === "top"
            ? side === "team"
              ? "text-sky-300/80"
              : "text-orange-300/80"
            : side === "team"
              ? "text-amber-300/80"
              : "text-fuchsia-300/80",
        )}
      >
        {label}
      </span>
      {clusters.map((cluster) => {
        const pos = position(cluster.tick, roundStartTick, roundEndTick)
        if (cluster.events.length === 1) {
          const ev = cluster.events[0]
          const detonateRight =
            ev.kind === "grenade" && ev.detonateTick !== undefined
              ? position(ev.detonateTick, roundStartTick, roundEndTick)
              : undefined
          return (
            <EventMarker
              key={cluster.id}
              event={ev}
              position={pos}
              detonateRight={detonateRight}
              roundStartTick={roundStartTick}
              tickRate={tickRate}
            />
          )
        }
        return (
          <EventClusterBadge
            key={cluster.id}
            cluster={cluster}
            position={pos}
            roundStartTick={roundStartTick}
            tickRate={tickRate}
          />
        )
      })}
    </div>
  )
}
