import type { LaneSide } from "@/lib/timeline/types"

// Team / role color encoding for the unified events track. Replaces the old
// dual-lane Y-axis split: each marker carries its side, and the renderer
// tints a small chip behind the icon so the team is readable at a glance.
//
// Round mode  → ct/t      (sky / amber, matches the scoreboard accents)
// Player mode → caused/affected (orange / fuchsia, matches the legacy
//                                 caused/affected lane backgrounds)
export function sideChipClass(side: LaneSide): string {
  switch (side) {
    case "ct":
      return "bg-sky-400/15 ring-sky-300/40"
    case "t":
      return "bg-amber-400/15 ring-amber-300/40"
    case "caused":
      return "bg-orange-500/20 ring-orange-300/45"
    case "affected":
      return "bg-fuchsia-500/20 ring-fuchsia-300/45"
  }
}

export function sideTextClass(side: LaneSide): string {
  switch (side) {
    case "ct":
      return "text-sky-300"
    case "t":
      return "text-amber-300"
    case "caused":
      return "text-orange-300"
    case "affected":
      return "text-fuchsia-300"
  }
}

// Short label used as the team prefix in event tooltips. Keeping it concise
// (2–8 chars) matches the rest of the HUD callsign typography.
export function sideLabel(side: LaneSide): string {
  switch (side) {
    case "ct":
      return "CT"
    case "t":
      return "T"
    case "caused":
      return "Caused"
    case "affected":
      return "Affected"
  }
}
