import { BadgeCheck } from "lucide-react"
import { Card } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import type { FaceitProfile } from "@/types/faceit"

const LEVEL_COLOR: Record<number, string> = {
  10: "var(--tier-10)",
  9: "var(--tier-9)",
  8: "var(--tier-8)",
  7: "var(--tier-7)",
  6: "var(--tier-6)",
  5: "var(--tier-5)",
  4: "var(--tier-4)",
  3: "var(--tier-3)",
  2: "var(--tier-2)",
  1: "var(--tier-1)",
}

// Simple level thresholds; real Faceit floors/ceilings are tiered per level
// and the API doesn't expose them directly.  These match the commonly-used
// public Faceit level boundaries.
const LEVEL_ELO: Array<[number, number, number]> = [
  [1, 1, 800],
  [2, 801, 950],
  [3, 951, 1100],
  [4, 1101, 1250],
  [5, 1251, 1400],
  [6, 1401, 1550],
  [7, 1551, 1700],
  [8, 1701, 1850],
  [9, 1851, 2000],
  [10, 2001, 99999],
]

function levelBounds(level: number | null, elo: number | null) {
  if (level == null) return { floor: 0, ceiling: 0, toNext: 0 }
  const row = LEVEL_ELO.find((r) => r[0] === level) ?? LEVEL_ELO[0]
  const floor = row[1]
  const ceiling = row[2] === 99999 ? Math.max(elo ?? floor, floor) : row[2]
  const toNext = Math.max(0, ceiling - (elo ?? 0))
  return { floor, ceiling, toNext }
}

interface ProfileHeroProps {
  profile: FaceitProfile | undefined
  isLoading: boolean
}

export function ProfileHero({ profile, isLoading }: ProfileHeroProps) {
  if (isLoading) {
    return (
      <Card
        className="grid gap-4 border border-[var(--border)] bg-[var(--bg-elevated)] px-5 py-4"
        data-testid="profile-hero-skeleton"
      >
        <div className="flex items-center gap-4">
          <Skeleton className="h-[68px] w-[68px] rounded-[10px]" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-5 w-40" />
            <Skeleton className="h-3 w-48" />
          </div>
          <Skeleton className="h-8 w-20" />
        </div>
        <Skeleton className="h-1.5 w-full" />
      </Card>
    )
  }

  if (!profile) return null

  const level = profile.level ?? 1
  const elo = profile.elo ?? 0
  const { floor, ceiling, toNext } = levelBounds(level, elo)
  const pct =
    ceiling > floor
      ? Math.max(0, Math.min(100, ((elo - floor) / (ceiling - floor)) * 100))
      : 0
  const levelColor = LEVEL_COLOR[level] ?? "var(--tier-5)"

  return (
    <Card className="grid gap-4 border border-[var(--border)] bg-[var(--bg-elevated)] px-5 py-4">
      <div className="grid grid-cols-[auto_1fr_auto] items-center gap-5">
        <div className="relative">
          <div
            className="grid h-[68px] w-[68px] place-items-center rounded-[10px] font-extrabold text-white"
            style={{
              fontSize: 24,
              background: "linear-gradient(135deg, #ff8a3d, #e11d48)",
            }}
          >
            {profile.nickname?.[0]?.toUpperCase() ?? "?"}
          </div>
          <div
            className="absolute -right-1.5 -bottom-1.5 grid h-7 w-7 place-items-center rounded-[8px] text-[11px] font-bold text-white"
            style={{ background: levelColor }}
            aria-label={`Level ${level}`}
          >
            {level}
          </div>
        </div>

        <div className="min-w-0">
          <div className="flex items-center gap-1.5">
            <span className="text-[20px] font-bold text-[var(--text)] leading-tight">
              {profile.nickname}
            </span>
            <BadgeCheck
              className="h-4 w-4"
              style={{ color: "var(--accent)" }}
              aria-hidden
            />
          </div>
          <div className="mt-1 flex items-center gap-2 text-[12.5px] text-[var(--text-muted)]">
            {profile.country && <span>{profile.country.toUpperCase()}</span>}
            {profile.country && <span aria-hidden>·</span>}
            <span>EU</span>
            <span aria-hidden>·</span>
            <span>faceit / {profile.nickname}</span>
          </div>
        </div>

        <div className="text-right">
          <div className="text-[10.5px] font-semibold uppercase tracking-wider text-[var(--text-subtle)]">
            Elo
          </div>
          <div className="tabular mt-0.5 text-[28px] font-bold leading-none text-[var(--text)]">
            {elo.toLocaleString()}
          </div>
        </div>
      </div>

      <div>
        <div
          className="h-1.5 w-full overflow-hidden rounded-full"
          style={{ background: "var(--bg-sunken)" }}
        >
          <div
            className="h-full rounded-full transition-[width] duration-[600ms] ease-out"
            style={{
              width: `${pct}%`,
              background: "linear-gradient(90deg, #e11d48, #ff8a3d)",
            }}
          />
        </div>
        <div
          className={cn(
            "tabular mt-2 flex justify-between text-[11px] text-[var(--text-subtle)]",
          )}
        >
          <span>
            Lv {level} · {floor.toLocaleString()}
          </span>
          <span>
            {toNext > 0 ? `${toNext.toLocaleString()} Elo to next` : "Max"}
          </span>
          <span>
            Lv {Math.min(10, level + 1)} · {ceiling.toLocaleString()}
          </span>
        </div>
      </div>
    </Card>
  )
}
