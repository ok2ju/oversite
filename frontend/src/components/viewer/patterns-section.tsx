import { useMemo } from "react"
import { useViewerStore } from "@/stores/viewer"
import { useMistakeTimeline } from "@/hooks/use-mistake-timeline"
import { cn } from "@/lib/utils"
import type { MistakeEntry } from "@/types/mistake"

// PATTERN_KINDS are the cross-duel signals that surface here instead of
// on the round-timeline duels lane. eco_misbuy and he_damage are
// inherently non-engagement; repeated_death_zone is a habit rolled up
// from multiple duels and reads better as a pattern.
//
// flash_assist is intentionally excluded — it attaches to the resolving
// kill's duel via cross-player attribution and renders on the duels lane
// instead.
const PATTERN_KINDS = new Set([
  "eco_misbuy",
  "he_damage",
  "repeated_death_zone",
])

const PATTERN_LABEL: Record<string, string> = {
  eco_misbuy: "Eco misbuy",
  he_damage: "HE damage",
  repeated_death_zone: "Death zone",
}

interface PatternsSectionProps {
  steamId?: string | null
}

// PatternsSection surfaces cross-duel signals — pattern-level mistakes
// that don't anchor to a single engagement (eco_misbuy, he_damage) or
// roll up across multiple duels (repeated_death_zone, flash_assist).
// Renders nothing when there are no pattern entries.
export function PatternsSection({
  steamId: steamIdProp,
}: PatternsSectionProps = {}) {
  const demoId = useViewerStore((s) => s.demoId)
  const selectedSteamId = useViewerStore((s) => s.selectedPlayerSteamId)
  const steamId = steamIdProp ?? selectedSteamId
  const { data: mistakes } = useMistakeTimeline(demoId, steamId)

  const patterns = useMemo(() => {
    if (!mistakes) return [] as MistakeEntry[]
    return mistakes.filter((m) => PATTERN_KINDS.has(m.kind))
  }, [mistakes])

  if (!steamId || patterns.length === 0) return null

  // Group by kind so the section reads "3 HE damage flags · 1 eco misbuy"
  // rather than a flat row-per-tick which is noisy for round-level signals.
  const grouped = patterns.reduce<Map<string, MistakeEntry[]>>((acc, m) => {
    const list = acc.get(m.kind) ?? []
    list.push(m)
    acc.set(m.kind, list)
    return acc
  }, new Map())

  return (
    <section
      data-testid="patterns-section"
      className="flex flex-col gap-1.5 rounded-md border border-white/[0.06] bg-white/[0.02] p-2.5"
    >
      <span className="hud-callsign text-[10px] font-semibold tracking-wider text-white/55">
        Patterns &amp; highlights
      </span>
      <ul className="flex flex-wrap gap-1.5">
        {[...grouped.entries()].map(([kind, rows]) => (
          <li key={kind}>
            <span
              data-testid={`pattern-chip-${kind}`}
              className={cn(
                "inline-flex items-center gap-1 rounded-sm bg-white/10 px-1.5 py-0.5 text-[11px] text-white/80",
              )}
            >
              {PATTERN_LABEL[kind] ?? kind}
              <span className="text-white/50">{rows.length}</span>
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
