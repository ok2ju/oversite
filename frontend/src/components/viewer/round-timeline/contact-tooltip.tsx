import { useMemo, useState } from "react"
import { cn } from "@/lib/utils"
import { formatElapsedTime } from "@/lib/viewer/timeline-utils"
import type { ContactMarker } from "@/lib/timeline/types"
import type { main } from "@wailsjs/go/models"

interface ContactTooltipProps {
  contact: ContactMarker
  tickRate: number
  roundStartTick: number
}

type Phase = "pre" | "during" | "post"

const PHASE_ORDER: readonly Phase[] = ["pre", "during", "post"] as const

const SEVERITY_TEXT_CLASS: Record<number, string> = {
  1: "text-yellow-300",
  2: "text-orange-300",
  3: "text-red-300",
}

const CONTACT_MISTAKE_TITLES: Record<string, string> = {
  slow_reaction: "Slow reaction",
  missed_first_shot: "Missed first shot",
  isolated_peek: "Isolated peek",
  bad_crosshair_height: "Bad crosshair height",
  peek_while_reloading: "Peeked while reloading",
  shot_while_moving: "Shot while moving",
  aim_while_flashed: "Aimed while flashed",
  lost_hp_advantage: "Lost HP advantage",
  no_reposition_after_kill: "No reposition after kill",
  no_reload_with_cover: "No reload with cover",
}

function mistakeTitle(kind: string): string {
  return CONTACT_MISTAKE_TITLES[kind] ?? kind
}

function phaseLabel(phase: Phase): string {
  return phase === "pre"
    ? "Pre-engagement"
    : phase === "during"
      ? "Engagement"
      : "Post-engagement"
}

function outcomeLabel(outcome: main.ContactOutcome): string {
  switch (outcome) {
    case "won_clean":
      return "Won (clean)"
    case "won_damaged":
      return "Won (damaged)"
    case "traded_win":
      return "Traded win"
    case "traded_death":
      return "Traded death"
    case "untraded_death":
      return "Untraded death"
    case "disengaged":
      return "Disengaged"
    case "partial_win":
      return "Partial win"
    case "mutual_damage_no_kill":
      return "Mutual damage"
    default:
      return String(outcome)
  }
}

function mistakeKey(m: main.ContactMistake): string {
  return `${m.kind}:${m.phase}:${m.tick ?? "null"}`
}

function groupMistakesByPhase(
  mistakes: main.ContactMistake[],
): Record<Phase, main.ContactMistake[]> {
  const out: Record<Phase, main.ContactMistake[]> = {
    pre: [],
    during: [],
    post: [],
  }
  for (const m of mistakes) {
    const p = m.phase as Phase
    if (p === "pre" || p === "during" || p === "post") {
      out[p].push(m)
    }
  }
  return out
}

// Top-3 by severity DESC, expand reveals all. Phase 5 may pivot the policy
// to per-category coverage; the override lives here.
function pickVisible(
  mistakes: main.ContactMistake[],
  expanded: boolean,
): main.ContactMistake[] {
  if (expanded) return mistakes
  if (mistakes.length <= 3) return mistakes
  const bySeverity = [...mistakes].sort((a, b) => b.severity - a.severity)
  return bySeverity.slice(0, 3)
}

interface PhaseGroupsProps {
  grouped: Record<Phase, main.ContactMistake[]>
  visible: main.ContactMistake[]
}

function PhaseGroups({ grouped, visible }: PhaseGroupsProps) {
  const visibleSet = new Set(visible.map(mistakeKey))
  return (
    <div className="space-y-1.5">
      {PHASE_ORDER.map((phase) => {
        const all = grouped[phase] ?? []
        const shown = all.filter((m) => visibleSet.has(mistakeKey(m)))
        if (shown.length === 0) return null
        return (
          <div key={phase} data-testid={`contact-tooltip-phase-${phase}`}>
            <div className="text-[10px] uppercase tracking-wider text-white/40">
              {phaseLabel(phase)}
            </div>
            <ul className="space-y-0.5">
              {shown.map((m) => (
                <li
                  key={mistakeKey(m)}
                  className={cn(
                    "flex items-center gap-2",
                    SEVERITY_TEXT_CLASS[m.severity] ?? SEVERITY_TEXT_CLASS[1],
                  )}
                >
                  <span className="inline-block h-1.5 w-1.5 rounded-full bg-current" />
                  <span>{mistakeTitle(m.kind)}</span>
                </li>
              ))}
            </ul>
          </div>
        )
      })}
    </div>
  )
}

export function ContactTooltip({
  contact,
  tickRate,
  roundStartTick,
}: ContactTooltipProps) {
  const [expanded, setExpanded] = useState(false)
  const grouped = useMemo(
    () => groupMistakesByPhase(contact.mistakes),
    [contact.mistakes],
  )
  const visible = useMemo(
    () => pickVisible(contact.mistakes, expanded),
    [contact.mistakes, expanded],
  )
  const hiddenCount = contact.mistakes.length - visible.length

  return (
    <div
      className="contact-tooltip space-y-2 text-xs"
      data-testid="contact-tooltip"
    >
      <header className="flex items-baseline justify-between gap-3">
        <span className="font-semibold">{outcomeLabel(contact.outcome)}</span>
        <span className="text-white/50">
          @ {formatElapsedTime(contact.tFirst - roundStartTick, tickRate)}
        </span>
      </header>

      {contact.enemies.length > 0 && (
        <div className="text-white/60">
          vs{" "}
          {contact.enemies.length === 1
            ? "1 enemy"
            : `${contact.enemies.length} enemies`}
        </div>
      )}

      {contact.mistakes.length === 0 ? (
        <div className="italic text-white/40">No mistakes — clean contact</div>
      ) : (
        <PhaseGroups grouped={grouped} visible={visible} />
      )}

      {hiddenCount > 0 && !expanded && (
        <button
          type="button"
          data-testid="contact-tooltip-expand"
          onClick={() => setExpanded(true)}
          className="text-[10px] text-orange-400/80 underline hover:text-orange-300"
        >
          +{hiddenCount} more
        </button>
      )}
    </div>
  )
}
