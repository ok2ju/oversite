import { Container, Graphics, Text } from "pixi.js"
import type { TeamSide } from "@/types/roster"

export function getTeamColor(team: TeamSide): number {
  return team === "CT" ? 0x5b9bd5 : 0xe67e22
}

export function getTeamOutlineColor(team: TeamSide): number {
  return team === "CT" ? 0x1e3a5f : 0x7a4417
}

export function yawToRadians(yaw: number): number {
  return -(yaw * Math.PI) / 180
}

const BODY_RADIUS = 11
const OUTLINE_WIDTH = 1.5
const POINTER_TIP_EXTENSION = 5
const POINTER_HALF_WIDTH = 4
const SELECTION_RING_RADIUS = 18
const LABEL_OFFSET_Y = 22
const LABEL_FONT_SIZE = 13

export class PlayerSprite {
  container: Container
  private body: Graphics
  private pointer: Graphics
  private deathMarker: Graphics
  private selectionRing: Graphics
  private nameLabel: Text

  private _team: TeamSide | null = null
  private _isAlive: boolean | null = null
  private _isSelected: boolean | null = null
  private _name: string | null = null

  constructor() {
    this.container = new Container()
    this.container.eventMode = "static"
    this.container.cursor = "pointer"

    this.body = new Graphics()

    this.pointer = new Graphics()
    this.pointer
      .moveTo(BODY_RADIUS - 1, -POINTER_HALF_WIDTH)
      .lineTo(BODY_RADIUS + POINTER_TIP_EXTENSION, 0)
      .lineTo(BODY_RADIUS - 1, POINTER_HALF_WIDTH)
      .closePath()
      .fill(0xffffff)

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

    this.selectionRing = new Graphics()
    this.selectionRing
      .circle(0, 0, SELECTION_RING_RADIUS)
      .stroke({ color: 0xffffff, width: 2 })
    this.selectionRing.visible = false

    this.nameLabel = new Text({
      text: "",
      style: {
        fill: 0xffffff,
        fontSize: LABEL_FONT_SIZE,
        fontWeight: "700",
        dropShadow: {
          color: 0x000000,
          alpha: 0.75,
          blur: 3,
          distance: 1,
          angle: Math.PI / 2,
        },
      },
    })
    this.nameLabel.anchor.set(0.5, 0)
    this.nameLabel.y = LABEL_OFFSET_Y

    this.container.addChild(this.body)
    this.container.addChild(this.pointer)
    this.container.addChild(this.deathMarker)
    this.container.addChild(this.selectionRing)
    this.container.addChild(this.nameLabel)
  }

  private drawBody(team: TeamSide): void {
    this.body.clear()
    this.body
      .circle(0, 0, BODY_RADIUS)
      .fill(getTeamColor(team))
      .stroke({ color: getTeamOutlineColor(team), width: OUTLINE_WIDTH })
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
    this.pointer.rotation = yawToRadians(data.yaw)

    if (data.team !== this._team) {
      this._team = data.team
      this.drawBody(data.team)
    }

    if (data.name !== this._name) {
      this._name = data.name
      this.nameLabel.text = data.name
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
