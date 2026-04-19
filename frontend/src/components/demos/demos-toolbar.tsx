import { Search, SlidersHorizontal, Download } from "lucide-react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export type DemosFilter = "all" | "wins" | "losses" | "parsing"

const CHIPS: Array<{ value: DemosFilter; label: string }> = [
  { value: "all", label: "All" },
  { value: "wins", label: "Wins" },
  { value: "losses", label: "Losses" },
  { value: "parsing", label: "Parsing" },
]

interface DemosToolbarProps {
  search: string
  onSearchChange: (value: string) => void
  filter: DemosFilter
  onFilterChange: (value: DemosFilter) => void
}

export function DemosToolbar({
  search,
  onSearchChange,
  filter,
  onFilterChange,
}: DemosToolbarProps) {
  return (
    <div className="flex items-center gap-3">
      <div className="relative w-full max-w-[320px]">
        <Search className="absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-[var(--text-subtle)]" />
        <Input
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search map, file name, match ID"
          className="h-8 pl-8 text-[12.5px]"
          aria-label="Search demos"
        />
      </div>

      <div className="flex items-center gap-1">
        {CHIPS.map((chip) => (
          <button
            key={chip.value}
            type="button"
            onClick={() => onFilterChange(chip.value)}
            className={cn(
              "h-7 rounded-full px-3 text-[12px] font-medium transition-colors",
            )}
            style={
              filter === chip.value
                ? {
                    background: "var(--accent-soft)",
                    color: "var(--accent-ink)",
                  }
                : {
                    background: "var(--bg-sunken)",
                    color: "var(--text-muted)",
                  }
            }
            aria-pressed={filter === chip.value}
          >
            {chip.label}
          </button>
        ))}
      </div>

      <div className="ml-auto flex items-center gap-1">
        <Button variant="ghost" size="sm" className="gap-1.5" disabled>
          <SlidersHorizontal className="h-3.5 w-3.5" />
          More filters
        </Button>
        <Button variant="ghost" size="sm" className="gap-1.5" disabled>
          <Download className="h-3.5 w-3.5" />
          Export
        </Button>
      </div>
    </div>
  )
}
