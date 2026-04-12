import type { CS2MapName } from "../../calibration"

interface CoordinatePair {
  label: string
  world: { x: number; y: number }
  expectedPixel: { x: number; y: number }
}

/**
 * Known CS2 landmark positions mapped to expected pixel coordinates.
 * World coords sourced from game console (getpos), expected pixels
 * computed via: px = (wx - originX) / scale, py = (originY - wy) / scale.
 */
export const MAP_TEST_COORDINATES: Record<CS2MapName, CoordinatePair[]> = {
  de_dust2: [
    {
      label: "T Spawn",
      world: { x: -672, y: -672 },
      expectedPixel: { x: 410, y: 889 },
    },
    {
      label: "CT Spawn",
      world: { x: -536, y: 2060 },
      expectedPixel: { x: 441, y: 268 },
    },
    {
      label: "A Site",
      world: { x: 1248, y: 2480 },
      expectedPixel: { x: 846, y: 172 },
    },
    {
      label: "B Site",
      world: { x: -1384, y: 2560 },
      expectedPixel: { x: 248, y: 154 },
    },
    {
      label: "Mid Doors",
      world: { x: -240, y: 1264 },
      expectedPixel: { x: 508, y: 449 },
    },
  ],
  de_mirage: [
    {
      label: "T Spawn",
      world: { x: 462, y: -2980 },
      expectedPixel: { x: 738, y: 939 },
    },
    {
      label: "CT Spawn",
      world: { x: -430, y: 620 },
      expectedPixel: { x: 560, y: 219 },
    },
    {
      label: "A Site",
      world: { x: -220, y: 460 },
      expectedPixel: { x: 602, y: 251 },
    },
    {
      label: "B Site",
      world: { x: -2140, y: -400 },
      expectedPixel: { x: 218, y: 423 },
    },
    {
      label: "Mid",
      world: { x: -390, y: -800 },
      expectedPixel: { x: 568, y: 503 },
    },
  ],
  de_inferno: [
    {
      label: "T Spawn",
      world: { x: 572, y: -1800 },
      expectedPixel: { x: 543, y: 1157 },
    },
    {
      label: "CT Spawn",
      world: { x: -660, y: 2732 },
      expectedPixel: { x: 291, y: 232 },
    },
    {
      label: "A Site",
      world: { x: 2120, y: 230 },
      expectedPixel: { x: 858, y: 743 },
    },
    {
      label: "B Site",
      world: { x: 176, y: 3020 },
      expectedPixel: { x: 462, y: 173 },
    },
    {
      label: "Mid",
      world: { x: 40, y: 330 },
      expectedPixel: { x: 434, y: 722 },
    },
  ],
  de_nuke: [
    {
      label: "T Spawn",
      world: { x: -940, y: -1625 },
      expectedPixel: { x: 359, y: 644 },
    },
    {
      label: "CT Spawn",
      world: { x: -300, y: 2220 },
      expectedPixel: { x: 450, y: 95 },
    },
    {
      label: "A Site",
      world: { x: -453, y: 887 },
      expectedPixel: { x: 429, y: 286 },
    },
    {
      label: "B Site",
      world: { x: -750, y: -347 },
      expectedPixel: { x: 386, y: 462 },
    },
    {
      label: "Outside",
      world: { x: 900, y: -150 },
      expectedPixel: { x: 622, y: 434 },
    },
  ],
  de_ancient: [
    {
      label: "T Spawn",
      world: { x: -710, y: -1870 },
      expectedPixel: { x: 449, y: 807 },
    },
    {
      label: "CT Spawn",
      world: { x: -450, y: 1100 },
      expectedPixel: { x: 501, y: 213 },
    },
    {
      label: "A Site",
      world: { x: 350, y: 1010 },
      expectedPixel: { x: 661, y: 231 },
    },
    {
      label: "B Site",
      world: { x: -1900, y: 450 },
      expectedPixel: { x: 211, y: 343 },
    },
    {
      label: "Mid",
      world: { x: -200, y: -200 },
      expectedPixel: { x: 551, y: 473 },
    },
  ],
  de_vertigo: [
    {
      label: "T Spawn",
      world: { x: -1400, y: -740 },
      expectedPixel: { x: 442, y: 626 },
    },
    {
      label: "CT Spawn",
      world: { x: -420, y: 1200 },
      expectedPixel: { x: 687, y: 141 },
    },
    {
      label: "A Site",
      world: { x: -368, y: 1092 },
      expectedPixel: { x: 700, y: 168 },
    },
    {
      label: "B Site",
      world: { x: -2500, y: 560 },
      expectedPixel: { x: 167, y: 301 },
    },
    {
      label: "Mid",
      world: { x: -1600, y: 400 },
      expectedPixel: { x: 392, y: 340 },
    },
  ],
  de_anubis: [
    {
      label: "T Spawn",
      world: { x: -450, y: -690 },
      expectedPixel: { x: 449, y: 770 },
    },
    {
      label: "CT Spawn",
      world: { x: -340, y: 2370 },
      expectedPixel: { x: 470, y: 184 },
    },
    {
      label: "A Site",
      world: { x: 820, y: 1380 },
      expectedPixel: { x: 693, y: 373 },
    },
    {
      label: "B Site",
      world: { x: -2230, y: 1600 },
      expectedPixel: { x: 108, y: 331 },
    },
    {
      label: "Mid",
      world: { x: -650, y: 720 },
      expectedPixel: { x: 411, y: 500 },
    },
  ],
}
