import { Search, SlidersHorizontal } from "lucide-react"
import { cn } from "@/lib/utils"

export type DemosFilter = "all" | "ready" | "parsing" | "error"

interface DemosToolbarProps {
  search: string
  onSearchChange: (value: string) => void
  filter: DemosFilter
  onFilterChange: (value: DemosFilter) => void
  totalCount?: number
}

export function DemosToolbar({
  search,
  onSearchChange,
  filter,
  onFilterChange,
  totalCount,
}: DemosToolbarProps) {
  const chips: Array<{ value: DemosFilter; label: string }> = [
    {
      value: "all",
      label: typeof totalCount === "number" ? `All · ${totalCount}` : "All",
    },
    { value: "ready", label: "Ready" },
    { value: "parsing", label: "Parsing" },
    { value: "error", label: "Failed" },
  ]

  return (
    <div className="demos-toolbar">
      <div className="demos-search">
        <Search className="h-3.5 w-3.5" />
        <input
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search by map, player, match ID…"
          aria-label="Search demos"
        />
      </div>

      <div className="flex items-center gap-1.5">
        {chips.map((chip) => (
          <button
            key={chip.value}
            type="button"
            onClick={() => onFilterChange(chip.value)}
            className={cn("filter-chip", filter === chip.value && "on")}
            aria-pressed={filter === chip.value}
          >
            {chip.label}
          </button>
        ))}
      </div>

      <div className="ml-auto">
        <button type="button" className="btn-sm ghost" disabled>
          <SlidersHorizontal className="h-3 w-3" />
          More filters
        </button>
      </div>
    </div>
  )
}
