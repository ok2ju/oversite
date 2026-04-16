export interface KDEPoint {
  x: number
  y: number
  weight: number
}

export interface DensityGrid {
  data: Float32Array
  width: number
  height: number
  maxDensity: number
}

/**
 * Compute a Gaussian Kernel Density Estimate on a 2D grid.
 *
 * For each point, only grid cells within `3 * bandwidth` are evaluated,
 * giving O(n * (6b)^2) instead of O(n * w * h).
 */
export function computeKDE(
  points: KDEPoint[],
  gridWidth: number,
  gridHeight: number,
  bandwidth: number,
): DensityGrid {
  const data = new Float32Array(gridWidth * gridHeight)

  if (points.length === 0 || bandwidth <= 0) {
    return { data, width: gridWidth, height: gridHeight, maxDensity: 0 }
  }

  const cutoff = Math.ceil(3 * bandwidth)
  const invBw2 = 1 / (bandwidth * bandwidth)
  // Normalization: 1 / (2π * bandwidth²)
  const norm = 1 / (2 * Math.PI * bandwidth * bandwidth)

  for (const point of points) {
    const cx = point.x
    const cy = point.y
    const w = point.weight

    const xMin = Math.max(0, Math.floor(cx - cutoff))
    const xMax = Math.min(gridWidth - 1, Math.ceil(cx + cutoff))
    const yMin = Math.max(0, Math.floor(cy - cutoff))
    const yMax = Math.min(gridHeight - 1, Math.ceil(cy + cutoff))

    for (let gy = yMin; gy <= yMax; gy++) {
      const dy = gy - cy
      const dy2 = dy * dy
      const rowOffset = gy * gridWidth

      for (let gx = xMin; gx <= xMax; gx++) {
        const dx = gx - cx
        const dist2 = dx * dx + dy2

        if (dist2 <= cutoff * cutoff) {
          const kernel = norm * Math.exp(-0.5 * dist2 * invBw2)
          data[rowOffset + gx] += w * kernel
        }
      }
    }
  }

  let maxDensity = 0
  for (let i = 0; i < data.length; i++) {
    if (data[i] > maxDensity) {
      maxDensity = data[i]
    }
  }

  return { data, width: gridWidth, height: gridHeight, maxDensity }
}
