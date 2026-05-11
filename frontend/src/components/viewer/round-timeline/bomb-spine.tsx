import type { SpineModel } from "@/lib/timeline/types"

interface BombSpineProps {
  spine: SpineModel
  roundStartTick: number
  roundEndTick: number
}

function position(
  tick: number,
  start: number,
  end: number,
): { left: number; width: number } {
  const span = Math.max(1, end - start)
  return {
    left: Math.max(0, Math.min(1, (tick - start) / span)),
    width: span,
  }
}

function range(
  range: { startTick: number; endTick: number } | null,
  start: number,
  end: number,
): { left: string; width: string } | null {
  if (!range) return null
  const left = position(range.startTick, start, end).left
  const right = position(range.endTick, start, end).left
  return {
    left: `${left * 100}%`,
    width: `${Math.max(0, right - left) * 100}%`,
  }
}

// Spine: thin horizontal band between the two lanes that draws round phases
// (freeze / live / post-plant) as subtle background blocks plus a 2px orange
// bomb bar when a plant exists.
export function BombSpine({
  spine,
  roundStartTick,
  roundEndTick,
}: BombSpineProps) {
  const live = range(spine.live, roundStartTick, roundEndTick)
  const postPlant = range(spine.postPlant, roundStartTick, roundEndTick)
  const bombBar = range(spine.bombBar, roundStartTick, roundEndTick)

  return (
    <div
      data-testid="round-timeline-spine"
      className="relative h-3 overflow-hidden rounded-sm bg-white/[0.02]"
      aria-hidden="true"
    >
      {live ? (
        <span
          data-testid="round-timeline-spine-live"
          className="absolute inset-y-0 bg-white/[0.08]"
          style={live}
        />
      ) : null}
      {postPlant ? (
        <span
          data-testid="round-timeline-spine-postplant"
          className="absolute inset-y-0 bg-orange-500/15"
          style={postPlant}
        />
      ) : null}
      {bombBar ? (
        <span
          data-testid="round-timeline-spine-bombbar"
          className="absolute left-0 top-1/2 h-[2px] -translate-y-1/2 rounded-full bg-orange-400 shadow-[0_0_6px_-1px_rgba(255,122,26,0.7)]"
          style={bombBar}
        />
      ) : null}
    </div>
  )
}
