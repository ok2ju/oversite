import type { GameEvent } from "@/types/demo"
import type { MistakeEntry } from "@/types/mistake"
import type { Round } from "@/types/round"
import { getEventIconPath } from "./icons"
import type {
  EventCluster,
  FilterSet,
  GrenadeMarker,
  LaneSide,
  MistakeMarker,
  RoundTimelineModel,
  SpineModel,
  TimelineEvent,
  TimelineEventKind,
} from "./types"

// Two markers whose pixel positions are within MIN_GAP_PX collapse into a
// single +N cluster badge. 12px is roughly half the 20×24 hit-target width;
// anything tighter overlaps visually.
export const MIN_GAP_PX = 12

interface BuildLanesInput {
  events: GameEvent[]
  mistakes: MistakeEntry[]
  round: Round
  selectedPlayerSteamId: string | null
  filters: FilterSet
  // Width of the lane track in pixels; used for clustering distance.
  laneWidthPx: number
}

// Helper: derive the round-live start tick (skipping freeze time when known).
function liveStart(round: Round): number {
  return round.freeze_end_tick > 0 ? round.freeze_end_tick : round.start_tick
}

function tickToPixel(
  tick: number,
  startTick: number,
  endTick: number,
  laneWidthPx: number,
): number {
  const span = Math.max(1, endTick - startTick)
  const clamped = Math.max(startTick, Math.min(tick, endTick))
  return ((clamped - startTick) / span) * laneWidthPx
}

// Group grenade lifecycle events (throw + bounces + detonate) by entity_id.
// Returns one GrenadeMarker per entity inside the round window.
function correlateGrenades(events: GameEvent[]): GrenadeMarker[] {
  const byEntity = new Map<
    string,
    { throw?: GameEvent; detonate?: GameEvent }
  >()
  for (const e of events) {
    if (
      e.event_type !== "grenade_throw" &&
      e.event_type !== "grenade_detonate"
    ) {
      continue
    }
    const entityId = String(
      (e.extra_data?.entity_id as number | string | undefined) ?? "",
    )
    if (!entityId) continue
    const bucket = byEntity.get(entityId) ?? {}
    if (e.event_type === "grenade_throw") {
      // Earliest throw wins (defensive; the parser only emits one per entity).
      if (!bucket.throw || e.tick < bucket.throw.tick) bucket.throw = e
    } else {
      if (!bucket.detonate || e.tick < bucket.detonate.tick) {
        bucket.detonate = e
      }
    }
    byEntity.set(entityId, bucket)
  }
  const out: GrenadeMarker[] = []
  for (const [entityId, { throw: thr, detonate }] of byEntity) {
    if (!thr) continue
    out.push({
      entityId,
      throwerSteamId: thr.attacker_steam_id,
      throwerTeam: thr.attacker_team,
      weapon: thr.weapon,
      throwTick: thr.tick,
      detonateTick: detonate?.tick ?? null,
      source: thr,
    })
  }
  return out
}

function teamToLaneSide(team: string): LaneSide | null {
  if (team === "CT") return "ct"
  if (team === "T") return "t"
  return null
}

// Decide which lane an event belongs to in team mode (CT top / T bottom).
function teamSideForEvent(event: GameEvent): LaneSide | null {
  // Kills: side of the attacker.
  if (event.event_type === "kill") {
    return teamToLaneSide(event.attacker_team)
  }
  if (event.event_type === "grenade_throw") {
    return teamToLaneSide(event.attacker_team)
  }
  if (event.event_type === "bomb_plant") {
    // T plants — always.
    return "t"
  }
  if (event.event_type === "bomb_defuse") {
    return "ct"
  }
  return null
}

// Decide which lane an event belongs to in player mode (caused top /
// affected bottom). Returns null if the event neither involves the player.
function playerSideForEvent(
  event: GameEvent,
  steamId: string,
): LaneSide | null {
  const isAttacker = event.attacker_steam_id === steamId
  const isVictim = event.victim_steam_id === steamId
  if (!isAttacker && !isVictim) return null
  if (event.event_type === "kill") {
    return isAttacker ? "caused" : "affected"
  }
  if (event.event_type === "grenade_throw") {
    return isAttacker ? "caused" : null
  }
  if (event.event_type === "bomb_plant" || event.event_type === "bomb_defuse") {
    return isAttacker ? "caused" : null
  }
  if (event.event_type === "player_hurt") {
    if (isVictim) return "affected"
    if (isAttacker) return "caused"
  }
  if (event.event_type === "player_flashed") {
    return isVictim ? "affected" : null
  }
  return null
}

function eventKindOf(event: GameEvent): TimelineEventKind | null {
  switch (event.event_type) {
    case "kill":
      return "kill"
    case "grenade_throw":
      return "grenade"
    case "bomb_plant":
      return "bomb_plant"
    case "bomb_defuse":
      return "bomb_defuse"
    case "bomb_explode":
      return "bomb_explode"
    case "player_hurt":
      return "player_hurt"
    case "player_flashed":
      return "player_flashed"
    default:
      return null
  }
}

// Mask an event against the active filter chips. In player mode, myEvents
// further narrows to events that involve the selected player.
function passesFilters(
  event: GameEvent,
  kind: TimelineEventKind,
  filters: FilterSet,
  selectedPlayerSteamId: string | null,
): boolean {
  if (kind === "kill" && !filters.kills) return false
  if (
    (kind === "grenade" ||
      kind === "player_hurt" ||
      kind === "player_flashed") &&
    !filters.utility
  ) {
    return false
  }
  if (
    (kind === "bomb_plant" ||
      kind === "bomb_defuse" ||
      kind === "bomb_explode") &&
    !filters.bomb
  ) {
    return false
  }
  if (filters.myEvents && selectedPlayerSteamId) {
    const involved =
      event.attacker_steam_id === selectedPlayerSteamId ||
      event.victim_steam_id === selectedPlayerSteamId
    if (!involved) return false
  }
  return true
}

// Project a GameEvent into a TimelineEvent, given the lane it belongs to.
function projectEvent(
  event: GameEvent,
  kind: TimelineEventKind,
  side: LaneSide,
): TimelineEvent {
  return {
    id: event.id,
    kind,
    tick: event.tick,
    side,
    iconPath: getEventIconPath(event.event_type, event.weapon),
    headshot: event.event_type === "kill" ? event.headshot : undefined,
    source: event,
  }
}

// Project a GrenadeMarker into a TimelineEvent on the given lane.
function projectGrenade(marker: GrenadeMarker, side: LaneSide): TimelineEvent {
  return {
    id: `grenade:${marker.entityId}:${marker.throwTick}`,
    kind: "grenade",
    tick: marker.throwTick,
    side,
    iconPath: getEventIconPath("grenade_throw", marker.weapon),
    detonateTick: marker.detonateTick ?? undefined,
    source: marker.source,
  }
}

// Greedy left-to-right clustering pass: events on the same lane whose pixel
// positions are within MIN_GAP_PX collapse into a single cluster.
function clusterLane(
  events: TimelineEvent[],
  startTick: number,
  endTick: number,
  laneWidthPx: number,
  laneId: string,
): EventCluster[] {
  if (events.length === 0) return []
  const sorted = [...events].sort((a, b) => a.tick - b.tick)
  const clusters: EventCluster[] = []
  let current: TimelineEvent[] = []
  let currentAnchorPx = -Infinity
  for (const ev of sorted) {
    const px = tickToPixel(ev.tick, startTick, endTick, laneWidthPx)
    if (current.length === 0 || px - currentAnchorPx <= MIN_GAP_PX) {
      current.push(ev)
      if (current.length === 1) currentAnchorPx = px
    } else {
      clusters.push(finalizeCluster(current, laneId, clusters.length))
      current = [ev]
      currentAnchorPx = px
    }
  }
  if (current.length > 0) {
    clusters.push(finalizeCluster(current, laneId, clusters.length))
  }
  return clusters
}

function finalizeCluster(
  events: TimelineEvent[],
  laneId: string,
  index: number,
): EventCluster {
  // Anchor at the median tick — keeps the badge centered over the constituent
  // group without being pulled by a single outlier.
  const sortedTicks = events.map((e) => e.tick).sort((a, b) => a - b)
  const mid = sortedTicks[Math.floor(sortedTicks.length / 2)]
  return {
    id: `${laneId}:${index}:${events[0].id}`,
    side: events[0].side,
    tick: mid,
    events,
  }
}

// Compute the round-phase + bomb spine geometry. All ranges are clamped to
// the round window so the renderer can divide by (end - start) without
// worrying about negative widths.
function buildSpine(round: Round, events: GameEvent[]): SpineModel {
  const start = round.start_tick
  const end = round.end_tick
  const live = liveStart(round)
  const bombPlant = events.find(
    (e) => e.event_type === "bomb_plant" && e.tick >= start && e.tick <= end,
  )
  const bombDefuse = events.find(
    (e) => e.event_type === "bomb_defuse" && e.tick >= start && e.tick <= end,
  )
  const bombExplode = events.find(
    (e) => e.event_type === "bomb_explode" && e.tick >= start && e.tick <= end,
  )
  const plantTick = bombPlant?.tick ?? null
  const bombEnd = bombDefuse?.tick ?? bombExplode?.tick ?? end

  const freeze = live > start ? { startTick: start, endTick: live } : null
  const liveEnd = plantTick ?? end
  const liveRange =
    liveEnd > live ? { startTick: live, endTick: liveEnd } : null
  const postPlant =
    plantTick !== null && bombEnd > plantTick
      ? { startTick: plantTick, endTick: bombEnd }
      : null
  const bombBar =
    plantTick !== null ? { startTick: plantTick, endTick: bombEnd } : null

  return {
    freeze,
    live: liveRange,
    postPlant,
    bombBar,
  }
}

// Pure: builds the full timeline model for a round.
//
// Steps:
//   1. Filter events to the round window.
//   2. Correlate grenades into per-entity markers.
//   3. Apply filter chips + (player-mode) MyEvents narrowing.
//   4. Assign each surviving event to a lane (CT/T or Caused/Affected).
//   5. Cluster within each lane.
//   6. Build the spine model.
//   7. Project mistakes into the mistakes lane (player mode only).
export function buildLanes(input: BuildLanesInput): RoundTimelineModel {
  const {
    events,
    mistakes,
    round,
    selectedPlayerSteamId,
    filters,
    laneWidthPx,
  } = input
  const start = round.start_tick
  const end = round.end_tick

  // (1) Round-window filter — events.
  const inRound = events.filter((e) => e.tick >= start && e.tick <= end)

  // (2) Grenade correlation.
  const grenades = correlateGrenades(inRound)
  const grenadeThrowIds = new Set(grenades.map((g) => g.source.id))
  // Drop the raw throw event so we don't double-render alongside the marker.
  // Also drop bounces/detonates — they're folded into the marker.
  const eventsForLane = inRound.filter((e) => {
    if (e.event_type === "grenade_throw" && grenadeThrowIds.has(e.id)) {
      return false
    }
    if (
      e.event_type === "grenade_bounce" ||
      e.event_type === "grenade_detonate" ||
      e.event_type === "smoke_start" ||
      e.event_type === "smoke_expired" ||
      e.event_type === "decoy_start" ||
      e.event_type === "fire_start" ||
      e.event_type === "weapon_fire"
    ) {
      return false
    }
    return true
  })

  const top: TimelineEvent[] = []
  const bottom: TimelineEvent[] = []

  const pushToLane = (ev: TimelineEvent) => {
    if (ev.side === "ct" || ev.side === "caused") top.push(ev)
    else if (ev.side === "t" || ev.side === "affected") bottom.push(ev)
  }

  // (3) + (4) Filter + lane assignment for grenades.
  for (const grenade of grenades) {
    const kind: TimelineEventKind = "grenade"
    if (!passesFilters(grenade.source, kind, filters, selectedPlayerSteamId)) {
      continue
    }
    let side: LaneSide | null
    if (selectedPlayerSteamId) {
      side = playerSideForEvent(grenade.source, selectedPlayerSteamId)
    } else {
      side = teamToLaneSide(grenade.throwerTeam)
    }
    if (!side) continue
    pushToLane(projectGrenade(grenade, side))
  }

  // (3) + (4) Filter + lane assignment for non-grenade events.
  for (const event of eventsForLane) {
    const kind = eventKindOf(event)
    if (!kind) continue
    // bomb_explode never renders as a lane icon — it's the right edge of the
    // bomb spine bar.
    if (kind === "bomb_explode") continue
    if (!passesFilters(event, kind, filters, selectedPlayerSteamId)) continue
    const side = selectedPlayerSteamId
      ? playerSideForEvent(event, selectedPlayerSteamId)
      : teamSideForEvent(event)
    if (!side) continue
    pushToLane(projectEvent(event, kind, side))
  }

  // (5) Cluster each lane independently.
  const topLane = clusterLane(top, start, end, laneWidthPx, "top")
  const bottomLane = clusterLane(bottom, start, end, laneWidthPx, "bottom")

  // (6) Spine geometry.
  const spine = buildSpine(round, inRound)

  // (7) Mistakes lane.
  const roundMistakes: MistakeMarker[] = []
  if (selectedPlayerSteamId) {
    for (const m of mistakes) {
      if (m.tick < start || m.tick > end) continue
      roundMistakes.push({
        id: m.id,
        kind: m.kind,
        title: m.title,
        severity: m.severity,
        tick: m.tick,
      })
    }
    // Sort by severity ascending so the highest-severity marker renders last
    // (on top in DOM z-order).
    roundMistakes.sort((a, b) => a.severity - b.severity)
  }

  return {
    topLane,
    bottomLane,
    mistakes: roundMistakes,
    spine,
    roundStartTick: start,
    roundEndTick: end,
    selectedPlayerSteamId,
  }
}
