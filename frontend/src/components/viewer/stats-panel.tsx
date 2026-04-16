import { useWeaponStats } from "@/hooks/use-heatmap"

interface StatsPanelProps {
  demoId: string | null
  visible: boolean
}

export function StatsPanel({ demoId, visible }: StatsPanelProps) {
  const { data: weaponStats, isLoading } = useWeaponStats(demoId)

  if (!visible) return null

  return (
    <div
      className="absolute right-0 top-0 z-20 h-full w-80 overflow-y-auto border-l border-border bg-background/95 p-4 backdrop-blur"
      data-testid="stats-panel"
    >
      <h3 className="mb-4 text-lg font-semibold">Weapon Stats</h3>

      {isLoading && <p className="text-sm text-muted-foreground">Loading...</p>}

      {!isLoading && (!weaponStats || weaponStats.length === 0) && (
        <p className="text-sm text-muted-foreground">No kill data available</p>
      )}

      {weaponStats && weaponStats.length > 0 && (
        <div className="flex flex-col gap-3">
          {weaponStats.map((stat) => {
            const hsPercent =
              stat.kill_count > 0
                ? Math.round((stat.hs_count / stat.kill_count) * 100)
                : 0
            const maxKills = weaponStats[0].kill_count

            return (
              <div key={stat.weapon} className="flex flex-col gap-1">
                <div className="flex items-center justify-between text-sm">
                  <span className="font-medium">{stat.weapon}</span>
                  <span className="text-muted-foreground">
                    {stat.kill_count} kills ({hsPercent}% HS)
                  </span>
                </div>
                <div className="h-2 w-full overflow-hidden rounded-full bg-secondary">
                  <div
                    className="h-full rounded-full bg-primary transition-all"
                    style={{
                      width: `${maxKills > 0 ? (stat.kill_count / maxKills) * 100 : 0}%`,
                    }}
                  />
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
