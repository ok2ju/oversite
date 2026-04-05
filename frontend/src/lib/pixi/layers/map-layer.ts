import { Assets, Sprite, type Container, type Texture } from "pixi.js"
import {
  isCS2Map,
  getMapCalibration,
  getRadarImagePath,
  type MapCalibration,
} from "@/lib/maps/calibration"

export class MapLayer {
  private container: Container
  private sprite: Sprite | null = null
  private _mapName: string | null = null
  private _calibration: MapCalibration | null = null
  private _loadId = 0

  constructor(container: Container) {
    this.container = container
  }

  get mapName(): string | null {
    return this._mapName
  }

  get calibration(): MapCalibration | null {
    return this._calibration
  }

  async setMap(mapName: string): Promise<void> {
    if (!isCS2Map(mapName)) {
      throw new Error(`Unknown CS2 map: "${mapName}"`)
    }

    this.clear()

    const loadId = ++this._loadId
    const calibration = getMapCalibration(mapName)!
    const imagePath = getRadarImagePath(mapName)
    const texture: Texture = await Assets.load(imagePath)

    if (loadId !== this._loadId) return

    const sprite = new Sprite({ texture })
    sprite.width = calibration.width
    sprite.height = calibration.height

    this.container.addChild(sprite)
    this.sprite = sprite
    this._mapName = mapName
    this._calibration = calibration
  }

  clear(): void {
    this._loadId++
    if (this.sprite) {
      this.container.removeChild(this.sprite)
      this.sprite.destroy()
      this.sprite = null
    }
    this._mapName = null
    this._calibration = null
  }

  destroy(): void {
    this.clear()
  }
}
