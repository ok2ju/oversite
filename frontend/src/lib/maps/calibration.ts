export type CS2MapName =
  | "de_dust2"
  | "de_mirage"
  | "de_inferno"
  | "de_nuke"
  | "de_ancient"
  | "de_vertigo"
  | "de_anubis"

export interface MapCalibration {
  originX: number
  originY: number
  scale: number
  width: number
  height: number
}

export interface PixelCoord {
  x: number
  y: number
}

export interface WorldCoord {
  x: number
  y: number
}

export const MAP_CALIBRATIONS: Record<CS2MapName, MapCalibration> = {
  de_dust2: {
    originX: -2476,
    originY: 3239,
    scale: 4.4,
    width: 1024,
    height: 1024,
  },
  de_mirage: {
    originX: -3230,
    originY: 1713,
    scale: 5.0,
    width: 1024,
    height: 1024,
  },
  de_inferno: {
    originX: -2087,
    originY: 3870,
    scale: 4.9,
    width: 1024,
    height: 1024,
  },
  de_nuke: {
    originX: -3453,
    originY: 2887,
    scale: 7.0,
    width: 1024,
    height: 1024,
  },
  de_ancient: {
    originX: -2953,
    originY: 2164,
    scale: 5.0,
    width: 1024,
    height: 1024,
  },
  de_vertigo: {
    originX: -3168,
    originY: 1762,
    scale: 4.0,
    width: 1024,
    height: 1024,
  },
  de_anubis: {
    originX: -2796,
    originY: 3328,
    scale: 5.22,
    width: 1024,
    height: 1024,
  },
}

const CS2_MAP_NAMES = new Set<string>(Object.keys(MAP_CALIBRATIONS))

export function isCS2Map(mapName: string): mapName is CS2MapName {
  return CS2_MAP_NAMES.has(mapName)
}

export function getMapCalibration(mapName: string): MapCalibration | undefined {
  return isCS2Map(mapName) ? MAP_CALIBRATIONS[mapName] : undefined
}

export function getRadarImagePath(mapName: CS2MapName): string {
  return `/maps/${mapName}.png`
}

export function worldToPixel(
  world: WorldCoord,
  calibration: MapCalibration,
): PixelCoord {
  return {
    x: (world.x - calibration.originX) / calibration.scale,
    y: (calibration.originY - world.y) / calibration.scale,
  }
}

// In-place variant for hot paths (per-frame rendering). Writes into `out`
// instead of allocating a fresh literal each call. At 64 Hz × 10 players +
// shot tracers + grenade trajectories, the original `worldToPixel` produces
// hundreds of {x,y} literals per second; reusing a layer-owned scratch object
// eliminates that GC pressure. Returns `out` so callers can chain.
export function worldToPixelInto(
  out: PixelCoord,
  worldX: number,
  worldY: number,
  calibration: MapCalibration,
): PixelCoord {
  out.x = (worldX - calibration.originX) / calibration.scale
  out.y = (calibration.originY - worldY) / calibration.scale
  return out
}

export function pixelToWorld(
  pixel: PixelCoord,
  calibration: MapCalibration,
): WorldCoord {
  return {
    x: pixel.x * calibration.scale + calibration.originX,
    y: calibration.originY - pixel.y * calibration.scale,
  }
}
