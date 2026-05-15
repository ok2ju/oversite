import type { GameEvent } from "@/types/demo"
import type { main } from "@wailsjs/go/models"

// Logical event-type vocabulary the round timeline thinks in. This is a
// projection of GameEventType — multiple wire-level event types collapse to a
// single timeline event (e.g. grenade_throw + grenade_bounce* + grenade_detonate
// become one GrenadeMarker keyed by entity_id).
export type TimelineEventKind =
  | "kill"
  | "grenade"
  | "bomb_plant"
  | "bomb_defuse"
  | "bomb_explode"
  | "player_hurt"
  | "player_flashed"

// Which lane an event/cluster belongs to.
// In team mode: "ct" = top lane, "t" = bottom lane.
// In player mode: "caused" = top lane, "affected" = bottom lane.
export type LaneSide = "ct" | "t" | "caused" | "affected"

// FilterSet drives the multi-select chip group in the controls strip.
// myEvents is inert when no player is selected.
export interface FilterSet {
  kills: boolean
  utility: boolean
  bomb: boolean
  myEvents: boolean
}

// A single event ready to render on a lane. Pixel position is derived in the
// lane component, not here, so the model stays width-agnostic.
export interface TimelineEvent {
  // Stable id; matches GameEvent.id for events, synthesized for grenade
  // markers as `grenade:<entity_id>:<throwTick>`.
  id: string
  kind: TimelineEventKind
  tick: number
  side: LaneSide
  // Sprite path (from getWeaponIconPath or static path). Null = caller renders
  // a fallback (small dot for player_hurt, etc.).
  iconPath: string | null
  // Optional headshot adornment for kills.
  headshot?: boolean
  // For grenades: detonateTick (when known) so the lane can draw a duration
  // bar for smokes / fires.
  detonateTick?: number
  // Original wire event used for tooltip rendering. For grenades, this is the
  // throw event (with entity_id) — bounces/detonates are folded into
  // detonateTick.
  source: GameEvent
}

// A grenade lifecycle correlated across grenade_throw + bounces + detonate.
// Returned by buildLanes for callers that need lifecycle data directly
// (e.g. drawing smoke/fire duration bars); also re-projected as a TimelineEvent
// for the lane renderer.
export interface GrenadeMarker {
  entityId: string
  throwerSteamId: string | null
  throwerTeam: string
  weapon: string | null
  throwTick: number
  detonateTick: number | null
  source: GameEvent
}

// A cluster groups overlapping events on the same lane into a single +N badge.
// A single-event cluster (count = 1) is the common case — the renderer can
// short-circuit to a plain marker.
export interface EventCluster {
  id: string
  side: LaneSide
  // Anchor tick for positioning (median tick of constituents).
  tick: number
  events: TimelineEvent[]
}

// Spine model — bomb-window geometry as a tick range. The events track
// renders this as a thin accent strip behind the event markers to keep the
// plant→end duration visible after the dual-lane stack was collapsed.
export interface SpineModel {
  // Bomb bar: plant tick → defuse/explode/end tick.
  bombBar: { startTick: number; endTick: number } | null
}

// Contacts projected onto the contacts lane (player mode only). Sorted
// by worstSeverity ascending so the most severe marker renders last
// (on top in DOM z-order). Embeds the full mistakes list so the tooltip
// can render without an additional query.
export interface ContactMarker {
  id: number
  subjectSteam: string
  // Tick the marker sits at on the lane (= t_first).
  tFirst: number
  // Lead-up tick — where the click handler seeks playback to.
  tPre: number
  tLast: number
  tPost: number
  outcome: main.ContactOutcome
  enemies: string[]
  // Mistakes attached to this contact. Sorted (phase ASC, severity DESC,
  // tick ASC) by the SQL ORDER BY in ListContactMistakesByContact.
  mistakes: main.ContactMistake[]
  // Max severity across mistakes (0 for clean contacts). Drives the
  // marker color.
  worstSeverity: number
}

// Full model the <RoundTimeline /> component consumes.
export interface RoundTimelineModel {
  // Unified event track. Each cluster carries the per-event side (ct/t in
  // team mode, caused/affected in player mode) which the renderer uses to
  // tint the icon chip — team-as-color replaces the old team-as-Y layout.
  events: EventCluster[]
  // Contacts (player mode only — empty when no player is selected).
  contacts: ContactMarker[]
  // Bomb-window geometry rendered as a thin accent strip inside the events
  // track. Empty when the round has no plant.
  spine: SpineModel
  // The round bounds the lanes are positioned against.
  roundStartTick: number
  roundEndTick: number
  // Echo of the selectedPlayerSteamId the model was built for — convenient for
  // the renderer to switch labels ("CT/T" vs "Caused/Affected") without having
  // to re-read the store.
  selectedPlayerSteamId: string | null
}
