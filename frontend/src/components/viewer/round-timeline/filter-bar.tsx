import { useCallback, useMemo } from "react"
import { Crosshair, Bomb, CircleAlert, User } from "lucide-react"
import { useViewerStore } from "@/stores/viewer"
import type { TimelineFilters } from "@/stores/viewer"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"

const FILTER_KEYS = ["kills", "utility", "bomb", "myEvents"] as const

const FILTER_META: Record<
  (typeof FILTER_KEYS)[number],
  { label: string; Icon: typeof Crosshair }
> = {
  kills: { label: "Kills", Icon: Crosshair },
  utility: { label: "Utility", Icon: CircleAlert },
  bomb: { label: "Bomb", Icon: Bomb },
  myEvents: { label: "My events", Icon: User },
}

export function FilterBar() {
  const filters = useViewerStore((s) => s.timelineFilters)
  const setTimelineFilter = useViewerStore((s) => s.setTimelineFilter)
  const selectedPlayerSteamId = useViewerStore((s) => s.selectedPlayerSteamId)

  const value = useMemo(() => {
    const out: string[] = []
    for (const key of FILTER_KEYS) {
      if (filters[key]) out.push(key)
    }
    return out
  }, [filters])

  const handleChange = useCallback(
    (next: string[]) => {
      const nextSet = new Set(next)
      for (const key of FILTER_KEYS) {
        const wanted = nextSet.has(key)
        if (filters[key] !== wanted) {
          setTimelineFilter(key as keyof TimelineFilters, wanted)
        }
      }
    },
    [filters, setTimelineFilter],
  )

  return (
    <ToggleGroup
      type="multiple"
      value={value}
      onValueChange={handleChange}
      data-testid="round-timeline-filter-bar"
      aria-label="Timeline filters"
    >
      {FILTER_KEYS.map((key) => {
        if (key === "myEvents" && !selectedPlayerSteamId) return null
        const { label, Icon } = FILTER_META[key]
        return (
          <ToggleGroupItem
            key={key}
            value={key}
            data-testid={`filter-chip-${key}`}
            aria-label={label}
          >
            <Icon size={11} className="mr-1" />
            {label}
          </ToggleGroupItem>
        )
      })}
    </ToggleGroup>
  )
}
