import { useMemo } from "react"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Slider } from "@/components/ui/slider"
import { useHeatmapStore } from "@/stores/heatmap"
import { useDemos } from "@/hooks/use-demos"
import { useUniqueWeapons, useUniquePlayers } from "@/hooks/use-heatmap"
import type { CS2MapName } from "@/lib/maps/calibration"

const CS2_MAPS: CS2MapName[] = [
  "de_dust2",
  "de_mirage",
  "de_inferno",
  "de_nuke",
  "de_ancient",
  "de_vertigo",
  "de_anubis",
]

export function FilterPanel() {
  const selectedMap = useHeatmapStore((s) => s.selectedMap)
  const selectedDemoIds = useHeatmapStore((s) => s.selectedDemoIds)
  const selectedWeapons = useHeatmapStore((s) => s.selectedWeapons)
  const selectedPlayer = useHeatmapStore((s) => s.selectedPlayer)
  const selectedSide = useHeatmapStore((s) => s.selectedSide)
  const bandwidth = useHeatmapStore((s) => s.bandwidth)
  const opacity = useHeatmapStore((s) => s.opacity)
  const setMap = useHeatmapStore((s) => s.setMap)
  const setDemoIds = useHeatmapStore((s) => s.setDemoIds)
  const setWeapons = useHeatmapStore((s) => s.setWeapons)
  const setPlayer = useHeatmapStore((s) => s.setPlayer)
  const setSide = useHeatmapStore((s) => s.setSide)
  const setBandwidth = useHeatmapStore((s) => s.setBandwidth)
  const setOpacity = useHeatmapStore((s) => s.setOpacity)

  // Fetch all demos to filter by map
  const { data: demoListData } = useDemos(1, 200)
  const { data: weapons } = useUniqueWeapons(selectedDemoIds)
  const { data: players } = useUniquePlayers(selectedDemoIds)

  // Filter demos by selected map
  const demosForMap = useMemo(() => {
    if (!demoListData?.data || !selectedMap) return []
    return demoListData.data.filter(
      (d) => d.map_name === selectedMap && d.status === "ready",
    )
  }, [demoListData?.data, selectedMap])

  const handleMapChange = (value: string) => {
    setMap(value === "all" ? null : (value as CS2MapName))
  }

  const handleDemoToggle = (demoId: number) => {
    const current = selectedDemoIds
    if (current.includes(demoId)) {
      setDemoIds(current.filter((id) => id !== demoId))
    } else {
      setDemoIds([...current, demoId])
    }
  }

  const handleSelectAllDemos = () => {
    setDemoIds(demosForMap.map((d) => d.id))
  }

  const handleWeaponToggle = (weapon: string) => {
    const current = selectedWeapons
    if (current.includes(weapon)) {
      setWeapons(current.filter((w) => w !== weapon))
    } else {
      setWeapons([...current, weapon])
    }
  }

  return (
    <div
      className="flex h-full w-[280px] shrink-0 flex-col gap-5 overflow-y-auto border-r border-border bg-background p-4"
      data-testid="heatmap-filter-panel"
    >
      <h2 className="text-lg font-semibold">Filters</h2>

      {/* Map Select */}
      <div className="flex flex-col gap-2">
        <Label>Map</Label>
        <Select value={selectedMap ?? "all"} onValueChange={handleMapChange}>
          <SelectTrigger>
            <SelectValue placeholder="Select map" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Maps</SelectItem>
            {CS2_MAPS.map((map) => (
              <SelectItem key={map} value={map}>
                {map.replace("de_", "").replace(/^\w/, (c) => c.toUpperCase())}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Demo Selection */}
      {selectedMap && (
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <Label>Demos ({selectedDemoIds.length})</Label>
            {demosForMap.length > 0 && (
              <Button
                variant="ghost"
                size="sm"
                className="h-6 px-2 text-xs"
                onClick={handleSelectAllDemos}
              >
                Select All
              </Button>
            )}
          </div>
          <div className="flex max-h-40 flex-col gap-1 overflow-y-auto rounded-md border border-input p-2">
            {demosForMap.length === 0 ? (
              <p className="text-xs text-muted-foreground">
                No demos for this map
              </p>
            ) : (
              demosForMap.map((demo) => (
                <label
                  key={demo.id}
                  className="flex cursor-pointer items-center gap-2 rounded px-1 py-0.5 text-sm hover:bg-accent"
                >
                  <input
                    type="checkbox"
                    checked={selectedDemoIds.includes(demo.id)}
                    onChange={() => handleDemoToggle(demo.id)}
                    className="rounded"
                  />
                  <span className="truncate">
                    {new Date(demo.match_date).toLocaleDateString()}
                  </span>
                </label>
              ))
            )}
          </div>
        </div>
      )}

      {/* Side Filter */}
      {selectedDemoIds.length > 0 && (
        <div className="flex flex-col gap-2">
          <Label>Side</Label>
          <div className="flex gap-1">
            {[
              { value: "", label: "Both" },
              { value: "CT", label: "CT" },
              { value: "T", label: "T" },
            ].map((option) => (
              <Button
                key={option.value}
                variant={selectedSide === option.value ? "default" : "outline"}
                size="sm"
                className="flex-1"
                onClick={() => setSide(option.value)}
              >
                {option.label}
              </Button>
            ))}
          </div>
        </div>
      )}

      {/* Weapon Filter */}
      {selectedDemoIds.length > 0 && weapons && weapons.length > 0 && (
        <div className="flex flex-col gap-2">
          <Label>Weapons</Label>
          <div className="flex max-h-32 flex-col gap-1 overflow-y-auto rounded-md border border-input p-2">
            {weapons.map((weapon) => (
              <label
                key={weapon}
                className="flex cursor-pointer items-center gap-2 rounded px-1 py-0.5 text-sm hover:bg-accent"
              >
                <input
                  type="checkbox"
                  checked={selectedWeapons.includes(weapon)}
                  onChange={() => handleWeaponToggle(weapon)}
                  className="rounded"
                />
                <span>{weapon}</span>
              </label>
            ))}
          </div>
        </div>
      )}

      {/* Player Filter */}
      {selectedDemoIds.length > 0 && players && players.length > 0 && (
        <div className="flex flex-col gap-2">
          <Label>Player</Label>
          <Select
            value={selectedPlayer || "all"}
            onValueChange={(v) => setPlayer(v === "all" ? "" : v)}
          >
            <SelectTrigger>
              <SelectValue placeholder="All players" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Players</SelectItem>
              {players.map((p) => (
                <SelectItem key={p.steam_id} value={p.steam_id}>
                  {p.player_name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      {/* Bandwidth Slider */}
      <div className="flex flex-col gap-2">
        <Label>Bandwidth: {bandwidth}</Label>
        <Slider
          min={5}
          max={50}
          step={1}
          value={[bandwidth]}
          onValueChange={([v]) => setBandwidth(v)}
        />
      </div>

      {/* Opacity Slider */}
      <div className="flex flex-col gap-2">
        <Label>Opacity: {Math.round(opacity * 100)}%</Label>
        <Slider
          min={10}
          max={100}
          step={5}
          value={[Math.round(opacity * 100)]}
          onValueChange={([v]) => setOpacity(v / 100)}
        />
      </div>
    </div>
  )
}
