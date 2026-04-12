import type { Container } from "pixi.js"
import { PlayerSprite } from "../sprites/player"
import { worldToPixel, type MapCalibration } from "@/lib/maps/calibration"
import type { TickData } from "@/types/demo"
import type { PlayerRosterEntry, TeamSide } from "@/types/roster"

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
    tickData: TickData[],
    calibration: MapCalibration,
    selectedSteamId: string | null,
  ): void {
    const activeSteamIds = new Set<string>()

    for (const tick of tickData) {
      activeSteamIds.add(tick.steam_id)

      let sprite = this.players.get(tick.steam_id)
      if (!sprite) {
        sprite = new PlayerSprite()
        if (this.clickCallback) {
          sprite.setClickHandler(this.clickCallback, tick.steam_id)
        }
        this.container.addChild(sprite.container)
        this.players.set(tick.steam_id, sprite)
      }

      const rosterEntry = this.roster?.get(tick.steam_id)
      const team: TeamSide = rosterEntry?.team_side ?? "CT"
      const name = rosterEntry?.player_name ?? tick.steam_id.slice(0, 10)

      const pixel = worldToPixel({ x: tick.x, y: tick.y }, calibration)

      sprite.update({
        x: pixel.x,
        y: pixel.y,
        yaw: tick.yaw,
        team,
        name,
        isAlive: tick.is_alive,
        isSelected: tick.steam_id === selectedSteamId,
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
