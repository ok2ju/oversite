import type { Container } from "pixi.js"
import { PlayerSprite } from "../sprites/player"
import {
  worldToPixelInto,
  type MapCalibration,
  type PixelCoord,
} from "@/lib/maps/calibration"
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

  // Scratch collections reused across update() calls. The map indexes the
  // current frame's `next` ticks by steam_id for O(1) pairing; the set tracks
  // which steam_ids appeared in `current` so we can prune stale sprites.
  // Both are .clear()'d (not reallocated) every tick to avoid GC churn.
  // Safe to reuse because update() finishes consuming both before returning,
  // and no caller retains a reference to either collection.
  private nextById = new Map<string, TickData>()
  private activeSteamIds = new Set<string>()
  // Per-layer scratch slot for worldToPixelInto — never escapes update().
  private pixelScratch: PixelCoord = { x: 0, y: 0 }

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
    // Reuse the index map; clear leftovers from the previous tick before we
    // repopulate. A fresh `new Map()` per frame would allocate ~10 entries +
    // backing storage at 64 Hz.
    this.nextById.clear()
    if (next) {
      for (const t of next) this.nextById.set(t.steam_id, t)
    }

    this.activeSteamIds.clear()
    const canInterpolate = next !== null && alpha > 0

    for (const cur of current) {
      this.activeSteamIds.add(cur.steam_id)

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

      const nxt = canInterpolate ? this.nextById.get(cur.steam_id) : undefined
      const worldX = nxt ? cur.x + (nxt.x - cur.x) * alpha : cur.x
      const worldY = nxt ? cur.y + (nxt.y - cur.y) * alpha : cur.y
      const yaw = nxt ? lerpAngle(cur.yaw, nxt.yaw, alpha) : cur.yaw

      // Write into the layer-owned scratch object instead of allocating a
      // fresh {x, y} literal per player per tick.
      worldToPixelInto(this.pixelScratch, worldX, worldY, calibration)

      sprite.update({
        x: this.pixelScratch.x,
        y: this.pixelScratch.y,
        yaw,
        team,
        name,
        health: cur.health,
        isAlive: cur.is_alive,
        isSelected: cur.steam_id === selectedSteamId,
        weapon: cur.weapon,
      })
    }

    // Remove sprites for players no longer in tick data.
    for (const [steamId, sprite] of this.players) {
      if (!this.activeSteamIds.has(steamId)) {
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
