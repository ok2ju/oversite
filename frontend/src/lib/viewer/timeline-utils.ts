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
 */
export function roundBoundaryPositions(
  boundaries: RoundBoundaryInput[],
  totalTicks: number,
): Array<{ roundNumber: number; percent: number }> {
  if (totalTicks <= 0) return []
  return boundaries.map((b) => ({
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
