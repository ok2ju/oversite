import { useViewerStore } from "@/stores/viewer"
import { EventMarker } from "./event-marker"
import { EventClusterBadge } from "./event-cluster"
import type { EventCluster, SpineModel } from "@/lib/timeline/types"

interface EventsTrackProps {
  clusters: EventCluster[]
  spine: SpineModel
  roundStartTick: number
  roundEndTick: number
}

function position(tick: number, start: number, end: number): number {
  const span = Math.max(1, end - start)
  return Math.max(0, Math.min(1, (tick - start) / span))
}

// Unified events track. Replaces the legacy top-lane / bomb-spine /
// bottom-lane stack: one 36px row, events placed by tick along X, team
// encoded by tinted chips behind each icon. The bomb-window (plant → end)
// renders as a slim accent strip at the bottom of the row so the duration
// information from the old spine isn't lost.
export function EventsTrack({
  clusters,
  spine,
  roundStartTick,
  roundEndTick,
}: EventsTrackProps) {
  const tickRate = useViewerStore((s) => s.tickRate)
  const bombBar = spine.bombBar
    ? {
        left: position(spine.bombBar.startTick, roundStartTick, roundEndTick),
        right: position(spine.bombBar.endTick, roundStartTick, roundEndTick),
      }
    : null

  return (
    <div
      data-testid="round-timeline-events"
      className="relative h-9 rounded-sm bg-white/[0.025]"
      aria-label="Round events"
    >
      {bombBar ? (
        <span
          data-testid="round-timeline-bomb-bar"
          aria-hidden="true"
          className="pointer-events-none absolute bottom-0 h-[3px] rounded-full shadow-[0_0_6px_-1px_rgba(232,155,42,0.7)]"
          style={{
            left: `${bombBar.left * 100}%`,
            width: `${Math.max(0, bombBar.right - bombBar.left) * 100}%`,
            background: "var(--accent)",
          }}
        />
      ) : null}
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
