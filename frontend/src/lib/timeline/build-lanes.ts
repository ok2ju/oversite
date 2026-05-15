import type { GameEvent } from "@/types/demo"
import type { Round } from "@/types/round"
import type { main } from "@wailsjs/go/models"
import { getEventIconPath } from "./icons"
import type {
  ContactMarker,
  EventCluster,
  FilterSet,
  GrenadeMarker,
  LaneSide,
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
  contacts: main.ContactMoment[]
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

// Compute the bomb-window geometry. The timeline starts at the live-phase
// tick, so freeze is excluded entirely. Returns the plant→end range so the
// events track can render a single accent strip; the dual phase tints from
// the legacy spine were dropped along with the dual-lane layout.
function buildSpine(round: Round, events: GameEvent[]): SpineModel {
  const start = liveStart(round)
  const end = round.end_tick
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
  const bombBar =
    plantTick !== null ? { startTick: plantTick, endTick: bombEnd } : null

  return { bombBar }
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
//   7. Project contacts into the contacts lane (player mode only).
export function buildLanes(input: BuildLanesInput): RoundTimelineModel {
  const {
    events,
    contacts,
    round,
    selectedPlayerSteamId,
    filters,
    laneWidthPx,
  } = input
  // Timeline spans the live phase only — events during freezetime are dropped
  // and the leftmost edge of the track is freeze_end_tick.
  const start = liveStart(round)
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
    // Round-mode lane shows only grenades + bomb. Drop kill / player_hurt /
    // player_flashed in round mode (no player selected); the per-player
    // contact lane encodes those signals in player mode.
    if (
      !selectedPlayerSteamId &&
      (e.event_type === "kill" ||
        e.event_type === "player_hurt" ||
        e.event_type === "player_flashed")
    ) {
      return false
    }
    return true
  })

  // Unified event collection. Side is kept per-event so the renderer can
  // tint the icon chip; events are no longer split into top/bottom lists.
  const unified: TimelineEvent[] = []

  // (3) + (4) Filter + side assignment for grenades.
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
    unified.push(projectGrenade(grenade, side))
  }

  // (3) + (4) Filter + side assignment for non-grenade events.
  for (const event of eventsForLane) {
    const kind = eventKindOf(event)
    if (!kind) continue
    // bomb_explode never renders as a marker — it's the right edge of the
    // bomb-window accent strip.
    if (kind === "bomb_explode") continue
    if (!passesFilters(event, kind, filters, selectedPlayerSteamId)) continue
    const side = selectedPlayerSteamId
      ? playerSideForEvent(event, selectedPlayerSteamId)
      : teamSideForEvent(event)
    if (!side) continue
    unified.push(projectEvent(event, kind, side))
  }

  // (5) Cluster across all events — mixed-team neighbours collapse into a
  // single +N badge whose popout disambiguates with per-row team chips.
  const eventsClusters = clusterLane(unified, start, end, laneWidthPx, "events")

  // (6) Spine geometry.
  const spine = buildSpine(round, inRound)

  // (7) Contacts lane.
  const roundContacts: ContactMarker[] = []
  if (selectedPlayerSteamId) {
    for (const c of contacts) {
      // Only render contacts inside the live round window. The Phase 2
      // builder doesn't emit contacts outside the round, but defensive
      // bounds mirror the legacy mistakes projection.
      if (c.t_first < start || c.t_first > end) continue
      roundContacts.push({
        id: c.id,
        subjectSteam: c.subject_steam,
        tFirst: c.t_first,
        tPre: c.t_pre,
        tLast: c.t_last,
        tPost: c.t_post,
        outcome: c.outcome,
        enemies: c.enemies ?? [],
        mistakes: c.mistakes ?? [],
        worstSeverity: worstSeverity(c.mistakes ?? []),
      })
    }
    // Sort by worstSeverity ascending so the highest-severity marker renders
    // last (on top in DOM z-order).
    roundContacts.sort((a, b) => a.worstSeverity - b.worstSeverity)
  }

  return {
    events: eventsClusters,
    contacts: roundContacts,
    spine,
    roundStartTick: start,
    roundEndTick: end,
    selectedPlayerSteamId,
  }
}

function worstSeverity(mistakes: main.ContactMistake[]): number {
  let max = 0
  for (const m of mistakes) {
    if (m.severity > max) max = m.severity
  }
  return max
}
