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
const WEAPON_LABEL_OFFSET_Y = LABEL_OFFSET_Y + LABEL_FONT_SIZE + 1
const WEAPON_LABEL_FONT_SIZE = 10

const HEALTH_BAR_WIDTH = 22
const HEALTH_BAR_HEIGHT = 3
const HEALTH_BAR_OFFSET_Y = -(BODY_RADIUS + 6)
const DAMAGE_RING_RADIUS = BODY_RADIUS + 4
const DAMAGE_FLASH_DURATION_MS = 450

export function getHealthColor(health: number): number {
  if (health > 60) return 0x22c55e
  if (health > 30) return 0xeab308
  return 0xef4444
}

export class PlayerSprite {
  container: Container
  private body: Graphics
  private pointer: Graphics
  private deathMarker: Graphics
  private selectionRing: Graphics
  private healthBar: Graphics
  private damageRing: Graphics
  private nameLabel: Text
  private weaponLabel: Text

  private _team: TeamSide | null = null
  private _isAlive: boolean | null = null
  private _isSelected: boolean | null = null
  private _name: string | null = null
  private _health: number | null = null
  private _weaponLabelText: string | null = null
  private _damageFlashEnd = 0

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

    this.healthBar = new Graphics()
    this.healthBar.visible = false

    this.damageRing = new Graphics()
    this.damageRing
      .circle(0, 0, DAMAGE_RING_RADIUS)
      .stroke({ color: 0xff3030, width: 2.5 })
    this.damageRing.visible = false
    this.damageRing.alpha = 0

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

    this.weaponLabel = new Text({
      text: "",
      style: {
        fill: 0xe5e7eb,
        fontSize: WEAPON_LABEL_FONT_SIZE,
        fontWeight: "500",
        dropShadow: {
          color: 0x000000,
          alpha: 0.75,
          blur: 3,
          distance: 1,
          angle: Math.PI / 2,
        },
      },
    })
    this.weaponLabel.anchor.set(0.5, 0)
    this.weaponLabel.y = WEAPON_LABEL_OFFSET_Y
    this.weaponLabel.visible = false

    this.container.addChild(this.body)
    this.container.addChild(this.pointer)
    this.container.addChild(this.deathMarker)
    this.container.addChild(this.selectionRing)
    this.container.addChild(this.damageRing)
    this.container.addChild(this.healthBar)
    this.container.addChild(this.nameLabel)
    this.container.addChild(this.weaponLabel)
  }

  private drawBody(team: TeamSide): void {
    this.body.clear()
    this.body
      .circle(0, 0, BODY_RADIUS)
      .fill(getTeamColor(team))
      .stroke({ color: getTeamOutlineColor(team), width: OUTLINE_WIDTH })
  }

  private drawHealthBar(health: number): void {
    this.healthBar.clear()
    if (health <= 0) return

    const clamped = Math.min(100, Math.max(0, health))
    const fillWidth = (clamped / 100) * HEALTH_BAR_WIDTH
    const x = -HEALTH_BAR_WIDTH / 2

    this.healthBar
      .roundRect(
        x - 0.5,
        HEALTH_BAR_OFFSET_Y - 0.5,
        HEALTH_BAR_WIDTH + 1,
        HEALTH_BAR_HEIGHT + 1,
        1.5,
      )
      .fill({ color: 0x000000, alpha: 0.7 })

    this.healthBar
      .roundRect(x, HEALTH_BAR_OFFSET_Y, fillWidth, HEALTH_BAR_HEIGHT, 1)
      .fill(getHealthColor(clamped))
  }

  update(data: {
    x: number
    y: number
    yaw: number
    team: TeamSide
    name: string
    health: number
    isAlive: boolean
    isSelected: boolean
    weaponLabel: string | null
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

    if (data.weaponLabel !== this._weaponLabelText) {
      this._weaponLabelText = data.weaponLabel
      if (data.weaponLabel) {
        this.weaponLabel.text = data.weaponLabel
        this.weaponLabel.visible = true
      } else {
        this.weaponLabel.visible = false
      }
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

    // Trigger a damage ring pulse when HP drops between samples. Skip on the
    // first update (no prior baseline) and on respawn (alive=false → alive=true
    // happens elsewhere; flash here only when health strictly decreases).
    if (this._health !== null && data.health < this._health) {
      this._damageFlashEnd = Date.now() + DAMAGE_FLASH_DURATION_MS
    }

    if (data.health !== this._health) {
      this._health = data.health
      this.drawHealthBar(data.health)
    }

    this.healthBar.visible = data.isAlive && data.health > 0

    const flashRemaining = this._damageFlashEnd - Date.now()
    if (flashRemaining > 0) {
      this.damageRing.visible = true
      this.damageRing.alpha = flashRemaining / DAMAGE_FLASH_DURATION_MS
    } else if (this.damageRing.visible) {
      this.damageRing.visible = false
      this.damageRing.alpha = 0
    }
  }

  setClickHandler(cb: (steamId: string) => void, steamId: string): void {
    this.container.on("pointerdown", () => cb(steamId))
  }

  destroy(): void {
    this.container.destroy()
  }
}
