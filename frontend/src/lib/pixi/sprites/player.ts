import { Container, Graphics, Text } from "pixi.js"
import type { TeamSide } from "@/types/roster"

export function getTeamColor(team: TeamSide): number {
  return team === "CT" ? 0x5b9bd5 : 0xe67e22
}

export function yawToRadians(yaw: number): number {
  return -(yaw * Math.PI) / 180
}

const PLAYER_RADIUS = 8
const VIEW_ANGLE_LENGTH = 16
const VIEW_ANGLE_HALF_FOV = (30 * Math.PI) / 180 // 60° total FOV
const SELECTION_RING_RADIUS = 12

export class PlayerSprite {
  container: Container
  private circle: Graphics
  private nameLabel: Text
  private viewAngle: Graphics
  private deathMarker: Graphics
  private selectionRing: Graphics

  private _team: TeamSide | null = null
  private _isAlive: boolean | null = null
  private _isSelected: boolean | null = null

  constructor() {
    this.container = new Container()
    this.container.eventMode = "static"
    this.container.cursor = "pointer"

    // Circle (filled, team color)
    this.circle = new Graphics()
    this.circle.circle(0, 0, PLAYER_RADIUS).fill(0xffffff)

    // Name label (above circle)
    this.nameLabel = new Text({
      text: "",
      style: { fill: 0xffffff, fontSize: 11 },
    })
    this.nameLabel.anchor.set(0.5, 1)
    this.nameLabel.y = -(PLAYER_RADIUS + 2)

    // View angle cone
    this.viewAngle = new Graphics()
    this.viewAngle
      .moveTo(0, 0)
      .lineTo(
        VIEW_ANGLE_LENGTH * Math.cos(-VIEW_ANGLE_HALF_FOV),
        VIEW_ANGLE_LENGTH * Math.sin(-VIEW_ANGLE_HALF_FOV),
      )
      .lineTo(
        VIEW_ANGLE_LENGTH * Math.cos(VIEW_ANGLE_HALF_FOV),
        VIEW_ANGLE_LENGTH * Math.sin(VIEW_ANGLE_HALF_FOV),
      )
      .closePath()
      .fill({ color: 0xffffff, alpha: 0.4 })

    // Death marker (red X)
    this.deathMarker = new Graphics()
    this.deathMarker
      .moveTo(-6, -6)
      .lineTo(6, 6)
      .stroke({ color: 0xff0000, width: 2 })
    this.deathMarker
      .moveTo(6, -6)
      .lineTo(-6, 6)
      .stroke({ color: 0xff0000, width: 2 })
    this.deathMarker.visible = false

    // Selection ring (stroke only)
    this.selectionRing = new Graphics()
    this.selectionRing
      .circle(0, 0, SELECTION_RING_RADIUS)
      .stroke({ color: 0xffffff, width: 2 })
    this.selectionRing.visible = false

    this.container.addChild(this.circle)
    this.container.addChild(this.nameLabel)
    this.container.addChild(this.viewAngle)
    this.container.addChild(this.deathMarker)
    this.container.addChild(this.selectionRing)
  }

  update(data: {
    x: number
    y: number
    yaw: number
    team: TeamSide
    name: string
    isAlive: boolean
    isSelected: boolean
  }): void {
    this.container.x = data.x
    this.container.y = data.y

    this.nameLabel.text = data.name
    this.viewAngle.rotation = yawToRadians(data.yaw)

    if (data.team !== this._team) {
      this._team = data.team
      const color = getTeamColor(data.team)
      this.circle.clear()
      this.circle.circle(0, 0, PLAYER_RADIUS).fill(color)
    }

    if (data.isAlive !== this._isAlive) {
      this._isAlive = data.isAlive
      this.container.alpha = data.isAlive ? 1.0 : 0.3
      this.deathMarker.visible = !data.isAlive
    }

    if (data.isSelected !== this._isSelected) {
      this._isSelected = data.isSelected
      this.selectionRing.visible = data.isSelected
    }
  }

  setClickHandler(cb: (steamId: string) => void, steamId: string): void {
    this.container.on("pointerdown", () => cb(steamId))
  }

  destroy(): void {
    this.container.destroy()
  }
}
