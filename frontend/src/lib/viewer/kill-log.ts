import type { GameEvent } from "@/types/demo"
import type { TeamSide } from "@/types/roster"

// One entry in the on-screen kill log. Preserves the original event tick so
// callers can compute fade/age and key React lists stably.
export interface KillEntry {
  id: string
  tick: number
  attackerName: string
  attackerSide: TeamSide | null
  victimName: string
  victimSide: TeamSide | null
  weapon: string | null
  headshot: boolean
  noScope: boolean
  penetrated: boolean
  throughSmoke: boolean
  attackerBlind: boolean
}

const DEFAULT_WINDOW_SECS = 7
const DEFAULT_MAX_ENTRIES = 5

export interface SelectKillsOptions {
  windowSecs?: number
  maxEntries?: number
  tickRate?: number
}

function readString(
  extra: Record<string, unknown> | null,
  key: string,
): string {
  const v = extra?.[key]
  return typeof v === "string" ? v : ""
}

function readBool(extra: Record<string, unknown> | null, key: string): boolean {
  return extra?.[key] === true
}

function readSide(
  extra: Record<string, unknown> | null,
  key: string,
): TeamSide | null {
  const v = extra?.[key]
  if (v === "CT" || v === "T") return v
  return null
}

// Filter all kill events to those that should be visible at currentTick: kills
// from up to `windowSecs` ago (no future kills), capped at the `maxEntries`
// most recent and returned oldest-first so the latest kill renders at the
// bottom of the on-screen feed. Suicides and team-kills with no attacker are
// skipped.
export function selectVisibleKills(
  events: GameEvent[] | undefined,
  currentTick: number,
  options: SelectKillsOptions = {},
): KillEntry[] {
  if (!events?.length) return []
  const tickRate =
    options.tickRate && options.tickRate > 0 ? options.tickRate : 64
  const windowSecs = options.windowSecs ?? DEFAULT_WINDOW_SECS
  const maxEntries = options.maxEntries ?? DEFAULT_MAX_ENTRIES
  const minTick = currentTick - windowSecs * tickRate

  const recent: KillEntry[] = []
  for (const e of events) {
    if (e.event_type !== "kill") continue
    if (e.tick > currentTick) continue
    if (e.tick < minTick) continue
    if (!e.attacker_steam_id || !e.victim_steam_id) continue

    recent.push({
      id: e.id,
      tick: e.tick,
      attackerName: readString(e.extra_data, "attacker_name"),
      attackerSide: readSide(e.extra_data, "attacker_team"),
      victimName: readString(e.extra_data, "victim_name"),
      victimSide: readSide(e.extra_data, "victim_team"),
      weapon: e.weapon,
      headshot: readBool(e.extra_data, "headshot"),
      noScope: readBool(e.extra_data, "no_scope"),
      penetrated:
        ((e.extra_data?.["penetrated"] as number | undefined) ?? 0) > 0,
      throughSmoke: readBool(e.extra_data, "through_smoke"),
      attackerBlind: readBool(e.extra_data, "attacker_blind"),
    })
  }

  recent.sort((a, b) => a.tick - b.tick)
  return recent.slice(-maxEntries)
}
