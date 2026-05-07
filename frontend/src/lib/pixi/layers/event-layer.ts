import { Graphics, type Container } from "pixi.js"
import type { GameEvent } from "@/types/demo"
import type { MapCalibration } from "@/lib/maps/calibration"
import { worldToPixel } from "@/lib/maps/calibration"
import {
  KILL_DURATION_TICKS,
  SHOT_DURATION_TICKS,
  HE_DURATION_TICKS,
  FLASH_DURATION_TICKS,
  SMOKE_DURATION_TICKS,
  BOMB_DEFUSE_TICKS,
  BOMB_DEFUSE_KIT_TICKS,
  SMOKE_RADIUS,
  FLASH_RADIUS,
  BOMB_ICON_RADIUS,
  SHOT_TRACER_LENGTH,
  COLOR_KILL,
  COLOR_SHOT,
  COLOR_SMOKE,
  COLOR_HE,
  COLOR_FLASH,
  COLOR_BOMB_PLANT,
  COLOR_BOMB_DEFUSE,
  computeKillState,
  computeShotState,
  computeSmokeState,
  computeHEState,
  computeFlashState,
  computeBombPlantState,
  computeBombDefuseState,
  worldRadiusToPixel,
  type EffectState,
} from "../sprites/effects"

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type EffectType =
  | "kill"
  | "shot"
  | "smoke"
  | "he"
  | "flash"
  | "bomb_plant"
  | "bomb_defuse"

interface ScheduledEffect {
  type: EffectType
  startTick: number
  durationTicks: number
  x: number
  y: number
  attackerX?: number
  attackerY?: number
  yaw?: number
  hitX?: number
  hitY?: number
  hasKit?: boolean
  smokeDuration?: number
}

interface ActiveEffect {
  effect: ScheduledEffect
  graphics: Graphics
}

// ---------------------------------------------------------------------------
// GraphicsPool
// ---------------------------------------------------------------------------

class GraphicsPool {
  private free: Graphics[] = []

  acquire(): Graphics {
    return this.free.length > 0 ? this.free.pop()! : new Graphics()
  }

  release(g: Graphics): void {
    g.clear()
    g.removeFromParent()
    this.free.push(g)
  }

  dispose(): void {
    for (const g of this.free) {
      g.destroy()
    }
    this.free = []
  }
}

// ---------------------------------------------------------------------------
// Drawing helpers
// ---------------------------------------------------------------------------

const KILL_MARKER_SIZE = 8

function drawKill(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  const vp = worldToPixel({ x: effect.x, y: effect.y }, calibration)
  g.clear()

  // Attacker → victim line
  if (effect.attackerX !== undefined && effect.attackerY !== undefined) {
    const ap = worldToPixel(
      { x: effect.attackerX, y: effect.attackerY },
      calibration,
    )
    g.moveTo(ap.x, ap.y)
      .lineTo(vp.x, vp.y)
      .stroke({ color: COLOR_KILL, width: 1, alpha: state.alpha * 0.5 })
  }

  // X marker
  const s = KILL_MARKER_SIZE
  g.moveTo(vp.x - s, vp.y - s)
    .lineTo(vp.x + s, vp.y + s)
    .stroke({ color: COLOR_KILL, width: 2, alpha: state.alpha })
  g.moveTo(vp.x + s, vp.y - s)
    .lineTo(vp.x - s, vp.y + s)
    .stroke({ color: COLOR_KILL, width: 2, alpha: state.alpha })
}

// Segment count for the gradient ray used when a shot has no recorded
// impact — PixiJS Graphics has no native gradient strokes, so we approximate
// with stacked short segments fading toward the unknown endpoint.
const SHOT_TRACER_SEGMENTS = 16
const SHOT_HIT_MARKER_SIZE = 3

function drawShot(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  const start = worldToPixel({ x: effect.x, y: effect.y }, calibration)
  g.clear()

  // Known impact (parser paired this shot with a player_hurt) — draw a solid
  // line ending exactly at the victim, plus a small marker on the impact.
  if (effect.hitX !== undefined && effect.hitY !== undefined) {
    const end = worldToPixel({ x: effect.hitX, y: effect.hitY }, calibration)
    g.moveTo(start.x, start.y)
      .lineTo(end.x, end.y)
      .stroke({ color: COLOR_SHOT, width: 1, alpha: state.alpha })
    g.circle(end.x, end.y, SHOT_HIT_MARKER_SIZE).fill({
      color: COLOR_SHOT,
      alpha: state.alpha,
    })
    return
  }

  // No impact recorded — draw a directional gradient ray that fades into the
  // unknown.
  const yawRad = ((effect.yaw ?? 0) * Math.PI) / 180
  const cos = Math.cos(yawRad)
  const sin = Math.sin(yawRad)

  for (let i = 0; i < SHOT_TRACER_SEGMENTS; i++) {
    const t0 = i / SHOT_TRACER_SEGMENTS
    const t1 = (i + 1) / SHOT_TRACER_SEGMENTS
    const segStart = worldToPixel(
      {
        x: effect.x + SHOT_TRACER_LENGTH * t0 * cos,
        y: effect.y + SHOT_TRACER_LENGTH * t0 * sin,
      },
      calibration,
    )
    const segEnd = worldToPixel(
      {
        x: effect.x + SHOT_TRACER_LENGTH * t1 * cos,
        y: effect.y + SHOT_TRACER_LENGTH * t1 * sin,
      },
      calibration,
    )
    const segmentAlpha = state.alpha * (1 - t0)
    g.moveTo(segStart.x, segStart.y)
      .lineTo(segEnd.x, segEnd.y)
      .stroke({ color: COLOR_SHOT, width: 1, alpha: segmentAlpha })
  }
}

function drawSmoke(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  const p = worldToPixel({ x: effect.x, y: effect.y }, calibration)
  const r = worldRadiusToPixel(SMOKE_RADIUS, calibration.scale)
  g.clear().circle(p.x, p.y, r).fill({ color: COLOR_SMOKE, alpha: state.alpha })
}

function drawHE(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  const p = worldToPixel({ x: effect.x, y: effect.y }, calibration)
  const r = worldRadiusToPixel(state.radius, calibration.scale)
  g.clear()
    .circle(p.x, p.y, Math.max(1, r))
    .fill({ color: COLOR_HE, alpha: state.alpha })
}

function drawFlash(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  const p = worldToPixel({ x: effect.x, y: effect.y }, calibration)
  const r = worldRadiusToPixel(FLASH_RADIUS, calibration.scale)
  g.clear().circle(p.x, p.y, r).fill({ color: COLOR_FLASH, alpha: state.alpha })
}

function drawBombPlant(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  const p = worldToPixel({ x: effect.x, y: effect.y }, calibration)
  const r = worldRadiusToPixel(BOMB_ICON_RADIUS, calibration.scale)
  g.clear()
    .circle(p.x, p.y, r)
    .fill({ color: COLOR_BOMB_PLANT, alpha: state.alpha })
}

function drawBombDefuse(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  const p = worldToPixel({ x: effect.x, y: effect.y }, calibration)
  const r = worldRadiusToPixel(BOMB_ICON_RADIUS, calibration.scale)
  g.clear()
    .circle(p.x, p.y, r)
    .fill({ color: COLOR_BOMB_DEFUSE, alpha: state.progress })
}

function computeState(
  effect: ScheduledEffect,
  tickOffset: number,
): EffectState {
  switch (effect.type) {
    case "kill":
      return computeKillState(tickOffset)
    case "shot":
      return computeShotState(tickOffset)
    case "smoke":
      return computeSmokeState(
        tickOffset,
        effect.smokeDuration ?? SMOKE_DURATION_TICKS,
      )
    case "he":
      return computeHEState(tickOffset)
    case "flash":
      return computeFlashState(tickOffset)
    case "bomb_plant":
      return computeBombPlantState(tickOffset, effect.durationTicks)
    case "bomb_defuse":
      return computeBombDefuseState(tickOffset, effect.hasKit ?? false)
  }
}

function drawEffect(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  switch (effect.type) {
    case "kill":
      return drawKill(g, state, effect, calibration)
    case "shot":
      return drawShot(g, state, effect, calibration)
    case "smoke":
      return drawSmoke(g, state, effect, calibration)
    case "he":
      return drawHE(g, state, effect, calibration)
    case "flash":
      return drawFlash(g, state, effect, calibration)
    case "bomb_plant":
      return drawBombPlant(g, state, effect, calibration)
    case "bomb_defuse":
      return drawBombDefuse(g, state, effect, calibration)
  }
}

// ---------------------------------------------------------------------------
// EventLayer
// ---------------------------------------------------------------------------

export class EventLayer {
  private container: Container
  private pool = new GraphicsPool()
  private scheduled: ScheduledEffect[] = []
  private active: ActiveEffect[] = []
  private nextIdx = 0
  private lastTick = -1

  constructor(container: Container) {
    this.container = container
  }

  setEvents(events: GameEvent[]): void {
    this.clear()
    this.scheduled = buildScheduled(events)
    this.nextIdx = 0
    this.lastTick = -1
  }

  update(currentTick: number, calibration: MapCalibration): void {
    // Backward seek
    if (currentTick < this.lastTick) {
      this.clear()
      this.nextIdx = 0
    }
    this.lastTick = currentTick

    // Activate new effects
    while (this.nextIdx < this.scheduled.length) {
      const effect = this.scheduled[this.nextIdx]
      if (effect.startTick > currentTick) break
      const tickOffset = currentTick - effect.startTick
      if (tickOffset < effect.durationTicks) {
        const g = this.pool.acquire()
        this.container.addChild(g)
        this.active.push({ effect, graphics: g })
      }
      this.nextIdx++
    }

    // Update active effects, remove expired
    for (let i = this.active.length - 1; i >= 0; i--) {
      const { effect, graphics } = this.active[i]
      const tickOffset = currentTick - effect.startTick
      const state = computeState(effect, tickOffset)
      if (state.active) {
        drawEffect(graphics, state, effect, calibration)
      } else {
        this.pool.release(graphics)
        this.active.splice(i, 1)
      }
    }
  }

  clear(): void {
    for (const { graphics } of this.active) {
      this.pool.release(graphics)
    }
    this.active = []
  }

  destroy(): void {
    this.clear()
    // Destroy all pooled graphics
    for (const { graphics } of this.active) {
      graphics.destroy()
    }
    this.pool.dispose()
  }
}

// ---------------------------------------------------------------------------
// Event preprocessing
// ---------------------------------------------------------------------------

function buildScheduled(events: GameEvent[]): ScheduledEffect[] {
  const scheduled: ScheduledEffect[] = []

  // Index smoke_expired events by entity_id for pairing
  const smokeExpired = new Map<string, number>() // entity_id → tick
  for (const e of events) {
    if (e.event_type === "smoke_expired") {
      const entityId = e.extra_data?.entity_id
      if (typeof entityId === "string") {
        smokeExpired.set(entityId, e.tick)
      }
    }
  }

  for (const e of events) {
    const x = e.x ?? 0
    const y = e.y ?? 0

    switch (e.event_type) {
      case "kill": {
        const extra = e.extra_data
        const effect: ScheduledEffect = {
          type: "kill",
          startTick: e.tick,
          durationTicks: KILL_DURATION_TICKS,
          x,
          y,
        }
        if (
          extra &&
          typeof extra.attacker_x === "number" &&
          typeof extra.attacker_y === "number"
        ) {
          effect.attackerX = extra.attacker_x
          effect.attackerY = extra.attacker_y
        }
        scheduled.push(effect)
        break
      }
      case "weapon_fire": {
        const extra = e.extra_data
        const yawRaw = extra?.yaw
        const hitXRaw = extra?.hit_x
        const hitYRaw = extra?.hit_y
        const effect: ScheduledEffect = {
          type: "shot",
          startTick: e.tick,
          durationTicks: SHOT_DURATION_TICKS,
          x,
          y,
          yaw: typeof yawRaw === "number" ? yawRaw : 0,
        }
        if (typeof hitXRaw === "number" && typeof hitYRaw === "number") {
          effect.hitX = hitXRaw
          effect.hitY = hitYRaw
        }
        scheduled.push(effect)
        break
      }
      case "smoke_start": {
        const entityId = e.extra_data?.entity_id
        let smokeDuration = SMOKE_DURATION_TICKS
        if (typeof entityId === "string" && smokeExpired.has(entityId)) {
          smokeDuration = smokeExpired.get(entityId)! - e.tick
        }
        scheduled.push({
          type: "smoke",
          startTick: e.tick,
          durationTicks: smokeDuration,
          smokeDuration,
          x,
          y,
        })
        break
      }
      case "grenade_detonate": {
        if (e.weapon === "HE Grenade") {
          scheduled.push({
            type: "he",
            startTick: e.tick,
            durationTicks: HE_DURATION_TICKS,
            x,
            y,
          })
        } else if (e.weapon === "Flashbang") {
          scheduled.push({
            type: "flash",
            startTick: e.tick,
            durationTicks: FLASH_DURATION_TICKS,
            x,
            y,
          })
        }
        break
      }
      case "bomb_plant": {
        scheduled.push({
          type: "bomb_plant",
          startTick: e.tick,
          // Bomb plant stays visible until defused/exploded; use defuse time as upper bound
          durationTicks: BOMB_DEFUSE_TICKS,
          x,
          y,
        })
        break
      }
      case "bomb_defuse": {
        const hasKit = e.extra_data?.has_kit === true
        const duration = hasKit ? BOMB_DEFUSE_KIT_TICKS : BOMB_DEFUSE_TICKS
        scheduled.push({
          type: "bomb_defuse",
          startTick: e.tick,
          durationTicks: duration,
          hasKit,
          x,
          y,
        })
        break
      }
      // Ignored: player_hurt, grenade_throw, smoke_expired, decoy_start, bomb_explode
      default:
        break
    }
  }

  scheduled.sort((a, b) => a.startTick - b.startTick)
  return scheduled
}
