import type { Container } from "pixi.js"
import { PlayerSprite } from "../sprites/player"
import { worldToPixel, type MapCalibration } from "@/lib/maps/calibration"
import type { TickData } from "@/types/demo"
import type { PlayerRosterEntry, TeamSide } from "@/types/roster"

// Shortest-arc lerp on a degree angle. ViewDirectionX wraps at 360, so naive
// lerp across the 359→0 boundary would spin the sprite the long way around.
function lerpAngle(a: number, b: number, t: number): number {
  const diff = ((b - a + 540) % 360) - 180
  return a + diff * t
}

export class PlayerLayer {
  private container: Container
  private players = new Map<string, PlayerSprite>()
  private roster: Map<string, PlayerRosterEntry> | null = null
  private clickCallback: ((steamId: string) => void) | null = null

  constructor(container: Container) {
    this.container = container
  }

  setRoster(entries: PlayerRosterEntry[]): void {
    this.roster = new Map(entries.map((e) => [e.steam_id, e]))
  }

  update(
    current: TickData[],
    next: TickData[] | null,
    alpha: number,
    calibration: MapCalibration,
    selectedSteamId: string | null,
  ): void {
    const nextById = new Map<string, TickData>()
    if (next) {
      for (const t of next) nextById.set(t.steam_id, t)
    }

    const activeSteamIds = new Set<string>()
    const canInterpolate = next !== null && alpha > 0

    for (const cur of current) {
      activeSteamIds.add(cur.steam_id)

      let sprite = this.players.get(cur.steam_id)
      if (!sprite) {
        sprite = new PlayerSprite()
        if (this.clickCallback) {
          sprite.setClickHandler(this.clickCallback, cur.steam_id)
        }
        this.container.addChild(sprite.container)
        this.players.set(cur.steam_id, sprite)
      }

      const rosterEntry = this.roster?.get(cur.steam_id)
      const team: TeamSide = rosterEntry?.team_side ?? "CT"
      const name = rosterEntry?.player_name ?? cur.steam_id.slice(0, 10)

      const nxt = canInterpolate ? nextById.get(cur.steam_id) : undefined
      const worldX = nxt ? cur.x + (nxt.x - cur.x) * alpha : cur.x
      const worldY = nxt ? cur.y + (nxt.y - cur.y) * alpha : cur.y
      const yaw = nxt ? lerpAngle(cur.yaw, nxt.yaw, alpha) : cur.yaw

      const pixel = worldToPixel({ x: worldX, y: worldY }, calibration)

      sprite.update({
        x: pixel.x,
        y: pixel.y,
        yaw,
        team,
        name,
        health: cur.health,
        isAlive: cur.is_alive,
        isSelected: cur.steam_id === selectedSteamId,
      })
    }

    // Remove sprites for players no longer in tick data.
    for (const [steamId, sprite] of this.players) {
      if (!activeSteamIds.has(steamId)) {
        this.container.removeChild(sprite.container)
        sprite.destroy()
        this.players.delete(steamId)
      }
    }
  }

  onPlayerClick(callback: (steamId: string) => void): void {
    this.clickCallback = callback
  }

  clear(): void {
    for (const sprite of this.players.values()) {
      this.container.removeChild(sprite.container)
      sprite.destroy()
    }
    this.players.clear()
  }

  destroy(): void {
    this.clear()
  }
}
