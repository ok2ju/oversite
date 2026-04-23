/**
 * Convert a tick to a percentage of the total timeline (0–100).
 */
export function tickToPercent(tick: number, totalTicks: number): number {
  if (totalTicks <= 0) return 0
  const clamped = Math.max(0, Math.min(tick, totalTicks))
  return (clamped / totalTicks) * 100
}

/**
 * Convert a percentage (0–100) to a tick number (integer), clamped to [0, totalTicks-1].
 */
export function percentToTick(percent: number, totalTicks: number): number {
  if (totalTicks <= 0) return 0
  const clamped = Math.max(0, Math.min(percent, 100))
  return Math.min(Math.floor((clamped / 100) * totalTicks), totalTicks - 1)
}

/**
 * Convert a mouse clientX to a percentage position within a track rect (0–100).
 */
export function clientXToPercent(clientX: number, trackRect: DOMRect): number {
  const raw = ((clientX - trackRect.left) / trackRect.width) * 100
  return Math.max(0, Math.min(100, raw))
}

interface RoundBoundaryInput {
  roundNumber: number
  startTick: number
  endTick: number
}

/**
 * Compute the percentage position for each round boundary marker.
 * Uses startTick as the marker position.
 * Filters out boundaries at tick 0 (round 1) since they overlap the track origin.
 */
export function roundBoundaryPositions(
  boundaries: RoundBoundaryInput[],
  totalTicks: number,
): Array<{ roundNumber: number; percent: number }> {
  if (totalTicks <= 0) return []
  return boundaries
    .filter((b) => b.startTick > 0)
    .map((b) => ({
      roundNumber: b.roundNumber,
      percent: (b.startTick / totalTicks) * 100,
    }))
}

/**
 * Format tick display as "currentTick / totalTicks" with locale-aware number formatting.
 */
export function formatTickDisplay(tick: number, totalTicks: number): string {
  return `${tick.toLocaleString("en-US")} / ${totalTicks.toLocaleString("en-US")}`
}

const DEFAULT_FREEZE_SECS = 15
const ROUND_TIME_SECS = 115

/**
 * Format ticks as elapsed time M:SS. Used by the scrubber to show position
 * within the current round.
 */
export function formatElapsedTime(ticks: number, tickRate: number): string {
  if (tickRate <= 0) return "0:00"
  const totalSecs = Math.max(0, Math.floor(ticks / tickRate))
  const mins = Math.floor(totalSecs / 60)
  const secs = totalSecs % 60
  return `${mins}:${String(secs).padStart(2, "0")}`
}

/**
 * Format ticks as a CS2 round clock in M:SS — a countdown, not elapsed time.
 *
 * For the first `freezeDurationTicks`, counts down the freeze time (e.g. 0:15 → 0:00).
 * Then counts down the round time 1:55 → 0:00 for the next 115 seconds.
 * Past the full round duration, returns "0:00".
 *
 * If `freezeDurationTicks` is 0 or missing, falls back to a 15-second default
 * (e.g. rounds parsed before RoundFreezetimeEnd was captured).
 *
 * Zero-pads seconds; minutes have no leading zero (e.g. "1:53", "0:39").
 */
export function formatRoundTime(
  elapsedTicks: number,
  tickRate: number,
  freezeDurationTicks = 0,
): string {
  if (tickRate <= 0) return "0:00"
  const freezeSecs =
    freezeDurationTicks > 0
      ? Math.round(freezeDurationTicks / tickRate)
      : DEFAULT_FREEZE_SECS
  const elapsedSecs = Math.max(0, Math.floor(elapsedTicks / tickRate))
  let remaining: number
  if (elapsedSecs < freezeSecs) {
    remaining = freezeSecs - elapsedSecs
  } else if (elapsedSecs < freezeSecs + ROUND_TIME_SECS) {
    remaining = freezeSecs + ROUND_TIME_SECS - elapsedSecs
  } else {
    remaining = 0
  }
  const mins = Math.floor(remaining / 60)
  const secs = remaining % 60
  return `${mins}:${String(secs).padStart(2, "0")}`
}
