import { Graphics, Sprite, type Container, type Texture } from "pixi.js"
import type { GameEvent } from "@/types/demo"
import type { MapCalibration, PixelCoord } from "@/lib/maps/calibration"
import { worldToPixelInto } from "@/lib/maps/calibration"
import { getWeaponTexture } from "../sprites/weapon-textures"

// Module-scoped scratch slots reused by every per-frame draw helper below.
// EventLayer.update() runs serially per frame, so two slots cover any draw
// function that needs two pixel coords live at the same time (kill: attacker
// + victim, shot: start + end, shot tracer segments: segStart + segEnd).
// Other draws use only `scratchA`. The values never escape the draw call —
// each helper consumes them before returning, so reuse is allocation-free.
const scratchA: PixelCoord = { x: 0, y: 0 }
const scratchB: PixelCoord = { x: 0, y: 0 }
import {
  KILL_DURATION_TICKS,
  SHOT_DURATION_TICKS,
  HE_DURATION_TICKS,
  FLASH_DURATION_TICKS,
  SMOKE_DURATION_TICKS,
  FIRE_DURATION_TICKS,
  BOMB_DEFUSE_TICKS,
  BOMB_DEFUSE_KIT_TICKS,
  SMOKE_RADIUS,
  FLASH_RADIUS,
  FIRE_RADIUS,
  BOMB_ICON_RADIUS,
  SHOT_TRACER_LENGTH,
  GRENADE_ICON_RADIUS,
  GRENADE_ICON_HEIGHT,
  GRENADE_TRAIL_ALPHA,
  COLOR_KILL,
  COLOR_SHOT,
  COLOR_SMOKE,
  COLOR_HE,
  COLOR_FLASH,
  COLOR_FIRE,
  COLOR_GRENADE_DEFAULT,
  COLOR_BOMB_PLANT,
  COLOR_BOMB_DEFUSE,
  computeKillState,
  computeShotState,
  computeSmokeState,
  computeHEState,
  computeFlashState,
  computeFireState,
  computeGrenadeTrajectoryState,
  computeBombPlantState,
  computeBombDefuseState,
  grenadeColor,
  interpolateTrajectory,
  worldRadiusToPixel,
  type EffectState,
  type TrajectoryWaypoint,
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
  | "fire"
  | "grenade_traj"
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
  // Grenade trajectory: ordered waypoints from throw → bounces → detonation,
  // and the rendering color picked by grenadeColor(). The weapon name drives
  // the sprite texture lookup for the in-flight icon.
  waypoints?: readonly TrajectoryWaypoint[]
  color?: number
  weapon?: string | null
}

interface ActiveEffect {
  effect: ScheduledEffect
  graphics: Graphics
  // Grenade trajectories overlay a Sprite for the weapon icon on top of the
  // Graphics-drawn trail. Other effect types leave this undefined.
  sprite?: Sprite
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

// Sprites are cheaper to keep than reconstruct — pool them like Graphics so
// rapid-fire grenade throws don't churn allocations.
class SpritePool {
  private free: Sprite[] = []

  acquire(): Sprite {
    if (this.free.length > 0) return this.free.pop()!
    const s = new Sprite()
    s.anchor.set(0.5, 0.5)
    return s
  }

  release(s: Sprite): void {
    s.removeFromParent()
    s.visible = false
    this.free.push(s)
  }

  dispose(): void {
    for (const s of this.free) {
      s.destroy()
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
  const vp = worldToPixelInto(scratchA, effect.x, effect.y, calibration)
  g.clear()

  // Attacker → victim line
  if (effect.attackerX !== undefined && effect.attackerY !== undefined) {
    const ap = worldToPixelInto(
      scratchB,
      effect.attackerX,
      effect.attackerY,
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
  // Known-impact branch needs only sx/sy / ex/ey scalars — copy out of
  // scratchA before reusing for `end` so we don't keep two scratches alive
  // for the full span. The hit branch returns early; the tracer branch reuses
  // both scratches per segment.
  worldToPixelInto(scratchA, effect.x, effect.y, calibration)
  const startX = scratchA.x
  const startY = scratchA.y
  g.clear()

  // Known impact (parser paired this shot with a player_hurt) — draw a solid
  // line ending exactly at the victim, plus a small marker on the impact.
  if (effect.hitX !== undefined && effect.hitY !== undefined) {
    const end = worldToPixelInto(
      scratchB,
      effect.hitX,
      effect.hitY,
      calibration,
    )
    g.moveTo(startX, startY)
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
    const segStart = worldToPixelInto(
      scratchA,
      effect.x + SHOT_TRACER_LENGTH * t0 * cos,
      effect.y + SHOT_TRACER_LENGTH * t0 * sin,
      calibration,
    )
    const segEnd = worldToPixelInto(
      scratchB,
      effect.x + SHOT_TRACER_LENGTH * t1 * cos,
      effect.y + SHOT_TRACER_LENGTH * t1 * sin,
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
  const p = worldToPixelInto(scratchA, effect.x, effect.y, calibration)
  const r = worldRadiusToPixel(SMOKE_RADIUS, calibration.scale)
  g.clear().circle(p.x, p.y, r).fill({ color: COLOR_SMOKE, alpha: state.alpha })
}

function drawHE(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  const p = worldToPixelInto(scratchA, effect.x, effect.y, calibration)
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
  const p = worldToPixelInto(scratchA, effect.x, effect.y, calibration)
  const r = worldRadiusToPixel(FLASH_RADIUS, calibration.scale)
  g.clear().circle(p.x, p.y, r).fill({ color: COLOR_FLASH, alpha: state.alpha })
}

function drawBombPlant(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  const p = worldToPixelInto(scratchA, effect.x, effect.y, calibration)
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
  const p = worldToPixelInto(scratchA, effect.x, effect.y, calibration)
  const r = worldRadiusToPixel(BOMB_ICON_RADIUS, calibration.scale)
  g.clear()
    .circle(p.x, p.y, r)
    .fill({ color: COLOR_BOMB_DEFUSE, alpha: state.progress })
}

function drawFire(
  g: Graphics,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
): void {
  const p = worldToPixelInto(scratchA, effect.x, effect.y, calibration)
  const r = worldRadiusToPixel(FIRE_RADIUS, calibration.scale)
  g.clear().circle(p.x, p.y, r).fill({ color: COLOR_FIRE, alpha: state.alpha })
}

// Renders an in-flight grenade with the weapon's CS2 sprite at the head,
// trailed by a faint line through the bounce points it has already passed.
// Position is lerped between waypoints so the icon doesn't teleport at
// bounces. Falls back to a colored dot while the SVG texture is still loading
// or when the weapon name is unmapped.
function drawGrenadeTrajectory(
  g: Graphics,
  sprite: Sprite | undefined,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
  tickOffset: number,
): void {
  const waypoints = effect.waypoints
  if (!waypoints || waypoints.length === 0) return

  const currentTick = effect.startTick + tickOffset
  const {
    x: wx,
    y: wy,
    segmentIndex,
  } = interpolateTrajectory(waypoints, currentTick)
  const color = effect.color ?? COLOR_GRENADE_DEFAULT

  g.clear()

  // Trail through completed waypoints + partial segment to the current head.
  // Each draw call consumes the scratch's x/y immediately, so a single
  // module-level scratch slot serves the start, every bounce, and the head.
  if (waypoints.length > 1) {
    const start = worldToPixelInto(
      scratchA,
      waypoints[0].x,
      waypoints[0].y,
      calibration,
    )
    g.moveTo(start.x, start.y)
    for (let i = 1; i <= segmentIndex; i++) {
      const p = worldToPixelInto(
        scratchA,
        waypoints[i].x,
        waypoints[i].y,
        calibration,
      )
      g.lineTo(p.x, p.y)
    }
    const trailHead = worldToPixelInto(scratchA, wx, wy, calibration)
    g.lineTo(trailHead.x, trailHead.y)
    g.stroke({
      color,
      width: 1,
      alpha: state.alpha * GRENADE_TRAIL_ALPHA,
    })
  }

  const head = worldToPixelInto(scratchA, wx, wy, calibration)
  const texture = sprite ? getWeaponTexture(effect.weapon) : null

  if (sprite && texture) {
    if (sprite.texture !== texture) {
      sprite.texture = texture
      const scale = GRENADE_ICON_HEIGHT / texture.height
      sprite.scale.set(scale)
    }
    sprite.x = head.x
    sprite.y = head.y
    sprite.alpha = state.alpha
    sprite.visible = true
  } else {
    if (sprite) sprite.visible = false
    // Texture not loaded yet (or no mapping) — show the legacy colored dot so
    // the grenade is never invisible.
    g.circle(head.x, head.y, GRENADE_ICON_RADIUS).fill({
      color,
      alpha: state.alpha,
    })
  }
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
    case "fire":
      return computeFireState(tickOffset, effect.durationTicks)
    case "grenade_traj":
      return computeGrenadeTrajectoryState(tickOffset, effect.durationTicks)
    case "bomb_plant":
      return computeBombPlantState(tickOffset, effect.durationTicks)
    case "bomb_defuse":
      return computeBombDefuseState(tickOffset, effect.hasKit ?? false)
  }
}

function drawEffect(
  g: Graphics,
  sprite: Sprite | undefined,
  state: EffectState,
  effect: ScheduledEffect,
  calibration: MapCalibration,
  tickOffset: number,
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
    case "fire":
      return drawFire(g, state, effect, calibration)
    case "grenade_traj":
      return drawGrenadeTrajectory(
        g,
        sprite,
        state,
        effect,
        calibration,
        tickOffset,
      )
    case "bomb_plant":
      return drawBombPlant(g, state, effect, calibration)
    case "bomb_defuse":
      return drawBombDefuse(g, state, effect, calibration)
  }
}

// ---------------------------------------------------------------------------
// EventLayer
// ---------------------------------------------------------------------------

// Minimal round shape consumed by the layer — keeps EventLayer decoupled
// from the full database row.
export interface RoundBound {
  start_tick: number
  end_tick: number
}

export class EventLayer {
  private container: Container
  private pool = new GraphicsPool()
  private spritePool = new SpritePool()
  private scheduled: ScheduledEffect[] = []
  private active: ActiveEffect[] = []
  private nextIdx = 0
  private lastTick = -1

  constructor(container: Container) {
    this.container = container
  }

  setEvents(events: GameEvent[], rounds?: readonly RoundBound[]): void {
    this.clear()
    this.scheduled = buildScheduled(events, rounds)
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
        let sprite: Sprite | undefined
        if (effect.type === "grenade_traj") {
          sprite = this.spritePool.acquire()
          // Reset texture so the per-tick assignment in drawGrenadeTrajectory
          // re-applies scale for whatever the new effect's weapon resolves to.
          sprite.texture = null as unknown as Texture
          this.container.addChild(sprite)
        }
        this.active.push({ effect, graphics: g, sprite })
      }
      this.nextIdx++
    }

    // Update active effects, remove expired
    for (let i = this.active.length - 1; i >= 0; i--) {
      const { effect, graphics, sprite } = this.active[i]
      const tickOffset = currentTick - effect.startTick
      const state = computeState(effect, tickOffset)
      if (state.active) {
        drawEffect(graphics, sprite, state, effect, calibration, tickOffset)
      } else {
        this.pool.release(graphics)
        if (sprite) this.spritePool.release(sprite)
        this.active.splice(i, 1)
      }
    }
  }

  clear(): void {
    for (const { graphics, sprite } of this.active) {
      this.pool.release(graphics)
      if (sprite) this.spritePool.release(sprite)
    }
    this.active = []
  }

  destroy(): void {
    this.clear()
    this.pool.dispose()
    this.spritePool.dispose()
  }
}

// ---------------------------------------------------------------------------
// Event preprocessing
// ---------------------------------------------------------------------------

// Normalize entity_id from extra_data to a string map key. Go's
// Entity.ID() ships through JSON as a number, so accepting both shapes
// keeps the pairing robust to upstream type changes.
function entityKey(extra: GameEvent["extra_data"]): string | null {
  const id = extra?.entity_id
  if (typeof id === "number") return String(id)
  if (typeof id === "string") return id
  return null
}

// Locate the index of the round containing `tick` (or the round it most
// recently entered). Returns -1 when the tick is before any round, e.g.
// warmup events.
function findRoundIndex(rounds: readonly RoundBound[], tick: number): number {
  for (let i = rounds.length - 1; i >= 0; i--) {
    if (tick >= rounds[i].start_tick) return i
  }
  return -1
}

function buildScheduled(
  events: GameEvent[],
  rounds?: readonly RoundBound[],
): ScheduledEffect[] {
  const scheduled: ScheduledEffect[] = []
  const haveRounds = !!rounds && rounds.length > 0

  // Cap each effect's duration so it cannot outlive the round that started
  // it. Without this, smokes (~18 s) and molotov fires (~7 s) thrown near
  // round-end would persist into the next round's freeze time, mismatching
  // CS2's own world cleanup. Trajectories are normally short enough not to
  // matter, but the cap defends against any throw-without-detonation slip.
  const cap = (effect: ScheduledEffect): ScheduledEffect => {
    if (!haveRounds || !rounds) return effect
    const idx = findRoundIndex(rounds, effect.startTick)
    if (idx < 0) return effect
    const maxDuration = rounds[idx].end_tick - effect.startTick
    if (maxDuration > 0 && maxDuration < effect.durationTicks) {
      return {
        ...effect,
        durationTicks: maxDuration,
        // Keep smokeDuration consistent so the alpha curve still fades in/out
        // proportionally over the (now shorter) visible window.
        ...(effect.type === "smoke" && { smokeDuration: maxDuration }),
      }
    }
    return effect
  }

  // First pass: index pairing data by entity_id.
  //   - smokeExpired: terminating tick for active smoke clouds
  //   - bounces: ordered list of in-flight bounce points
  //   - terminations: the event that ended the projectile (detonate /
  //     smoke_start / fire_start / decoy_start), used to bookend trajectories
  const smokeExpired = new Map<string, number>()
  const bounces = new Map<string, GameEvent[]>()
  const terminations = new Map<string, GameEvent>()

  for (const e of events) {
    const key = entityKey(e.extra_data)
    if (!key) continue
    switch (e.event_type) {
      case "smoke_expired":
        smokeExpired.set(key, e.tick)
        break
      case "grenade_bounce": {
        const arr = bounces.get(key)
        if (arr) arr.push(e)
        else bounces.set(key, [e])
        break
      }
      case "grenade_detonate":
      case "smoke_start":
      case "fire_start":
      case "decoy_start":
        // Each entity has at most one terminating event, but if a demo somehow
        // surfaces duplicates we keep the first to bound the trajectory tightly.
        if (!terminations.has(key)) terminations.set(key, e)
        break
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
        scheduled.push(cap(effect))
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
        scheduled.push(cap(effect))
        break
      }
      case "grenade_throw": {
        const key = entityKey(e.extra_data)
        if (!key) break
        const term = terminations.get(key)
        // Skip orphaned throws (demo truncated mid-flight) — without an
        // endpoint we can't bound the duration, and rendering a forever-flying
        // grenade would be worse than rendering nothing.
        if (!term) break
        if (term.tick <= e.tick) break

        const segs = bounces.get(key) ?? []
        const waypoints: TrajectoryWaypoint[] = [
          { tick: e.tick, x, y },
          ...segs.map((b) => ({
            tick: b.tick,
            x: b.x ?? 0,
            y: b.y ?? 0,
          })),
          { tick: term.tick, x: term.x ?? 0, y: term.y ?? 0 },
        ]
        scheduled.push(
          cap({
            type: "grenade_traj",
            startTick: e.tick,
            durationTicks: term.tick - e.tick,
            x,
            y,
            waypoints,
            color: grenadeColor(e.weapon),
            weapon: e.weapon,
          }),
        )
        break
      }
      case "smoke_start": {
        const key = entityKey(e.extra_data)
        let smokeDuration = SMOKE_DURATION_TICKS
        if (key && smokeExpired.has(key)) {
          smokeDuration = smokeExpired.get(key)! - e.tick
        }
        scheduled.push(
          cap({
            type: "smoke",
            startTick: e.tick,
            durationTicks: smokeDuration,
            smokeDuration,
            x,
            y,
          }),
        )
        break
      }
      case "fire_start": {
        scheduled.push(
          cap({
            type: "fire",
            startTick: e.tick,
            durationTicks: FIRE_DURATION_TICKS,
            x,
            y,
          }),
        )
        break
      }
      case "grenade_detonate": {
        if (e.weapon === "HE Grenade") {
          scheduled.push(
            cap({
              type: "he",
              startTick: e.tick,
              durationTicks: HE_DURATION_TICKS,
              x,
              y,
            }),
          )
        } else if (e.weapon === "Flashbang") {
          scheduled.push(
            cap({
              type: "flash",
              startTick: e.tick,
              durationTicks: FLASH_DURATION_TICKS,
              x,
              y,
            }),
          )
        }
        break
      }
      case "bomb_plant": {
        scheduled.push(
          cap({
            type: "bomb_plant",
            startTick: e.tick,
            // Bomb plant stays visible until defused/exploded; use defuse time as upper bound
            durationTicks: BOMB_DEFUSE_TICKS,
            x,
            y,
          }),
        )
        break
      }
      case "bomb_defuse": {
        const hasKit = e.extra_data?.has_kit === true
        const duration = hasKit ? BOMB_DEFUSE_KIT_TICKS : BOMB_DEFUSE_TICKS
        scheduled.push(
          cap({
            type: "bomb_defuse",
            startTick: e.tick,
            durationTicks: duration,
            hasKit,
            x,
            y,
          }),
        )
        break
      }
      // Ignored: player_hurt, grenade_bounce (consumed in first pass),
      // smoke_expired, bomb_explode
      default:
        break
    }
  }

  scheduled.sort((a, b) => a.startTick - b.startTick)
  return scheduled
}
