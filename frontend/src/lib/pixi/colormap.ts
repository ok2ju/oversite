/**
 * Pre-computed 256-entry RGBA lookup table.
 * Gradient: transparent → blue → cyan → green → yellow → red.
 */
const LUT = buildLUT()

function buildLUT(): Uint8Array {
  // 4 bytes per entry (RGBA), 256 entries
  const lut = new Uint8Array(256 * 4)

  // Color stops: [position 0-1, r, g, b]
  const stops: [number, number, number, number][] = [
    [0.0, 0, 0, 255], // blue
    [0.25, 0, 255, 255], // cyan
    [0.5, 0, 255, 0], // green
    [0.75, 255, 255, 0], // yellow
    [1.0, 255, 0, 0], // red
  ]

  for (let i = 0; i < 256; i++) {
    const t = i / 255

    // Index 0 is fully transparent (background)
    if (i === 0) {
      lut[0] = 0
      lut[1] = 0
      lut[2] = 0
      lut[3] = 0
      continue
    }

    // Find the two surrounding stops
    let s0 = stops[0]
    let s1 = stops[1]
    for (let s = 1; s < stops.length; s++) {
      if (stops[s][0] >= t) {
        s0 = stops[s - 1]
        s1 = stops[s]
        break
      }
    }

    const range = s1[0] - s0[0]
    const localT = range > 0 ? (t - s0[0]) / range : 0

    const r = Math.round(s0[1] + (s1[1] - s0[1]) * localT)
    const g = Math.round(s0[2] + (s1[2] - s0[2]) * localT)
    const b = Math.round(s0[3] + (s1[3] - s0[3]) * localT)

    // Alpha ramps from 0.3 at low density to 0.9 at high density
    const alpha = Math.round((0.3 + 0.6 * t) * 255)

    const offset = i * 4
    lut[offset] = r
    lut[offset + 1] = g
    lut[offset + 2] = b
    lut[offset + 3] = alpha
  }

  return lut
}

/**
 * Map a normalized density value [0, 1] to an RGBA tuple.
 */
export function densityToRGBA(
  normalizedDensity: number,
): [number, number, number, number] {
  const idx = Math.min(255, Math.max(0, Math.round(normalizedDensity * 255)))
  const offset = idx * 4
  return [LUT[offset], LUT[offset + 1], LUT[offset + 2], LUT[offset + 3]]
}

/**
 * Fill an ImageData buffer from a density grid using the colormap LUT.
 * Much faster than calling densityToRGBA per pixel.
 */
export function fillImageDataFromDensity(
  imageData: Uint8ClampedArray,
  densityData: Float32Array,
  maxDensity: number,
  opacity: number,
): void {
  if (maxDensity <= 0) return

  const invMax = 1 / maxDensity
  const len = densityData.length

  for (let i = 0; i < len; i++) {
    const d = densityData[i]
    if (d <= 0) continue

    const normalized = Math.min(1, d * invMax)
    const idx = Math.min(255, Math.max(1, Math.round(normalized * 255)))
    const lutOffset = idx * 4
    const imgOffset = i * 4

    imageData[imgOffset] = LUT[lutOffset]
    imageData[imgOffset + 1] = LUT[lutOffset + 1]
    imageData[imgOffset + 2] = LUT[lutOffset + 2]
    imageData[imgOffset + 3] = Math.round(LUT[lutOffset + 3] * opacity)
  }
}
