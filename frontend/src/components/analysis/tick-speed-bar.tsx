import { cn } from "@/lib/utils"

// TickSpeedBar renders one segment per sampled tick in a fire window with the
// player's planar speed at that tick. Status colors mirror habit-checklist
// (good = green, warn = yellow, bad = red). The bar is the heart of the
// mistake-detail forensic view: a glance answers "were you stopped when you
// fired?" without parsing the raw extras blob.
//
// Color rules (per plan §3.2):
//   - speed > weaponSpeedCap → bad (still moving on the trigger)
//   - speed ≤ weaponSpeedCap, > 10 → warn (decelerating but not stopped)
//   - speed ≤ 10 → good (effectively stopped)
//
// Hidden when the speeds array is missing or empty — the parent should not
// render the section header in that case either, but the component degrades
// to null defensively so a malformed extras blob never crashes the panel.

interface TickSpeedBarProps {
  speeds: number[]
  weaponSpeedCap: number
  // Optional matching ticks_window — when provided, each segment is labeled
  // with its tick number above the speed value. Mostly useful for debugging;
  // the default rendering shows "1, 2, 3, 4" indices.
  ticksWindow?: number[]
}

const STATUS_COLOR = {
  good: "#9bbc5a",
  warn: "#ffc233",
  bad: "#f87171",
} as const

const STILL_SPEED_THRESHOLD = 10

function statusForSpeed(speed: number, cap: number): keyof typeof STATUS_COLOR {
  if (speed > cap) return "bad"
  if (speed > STILL_SPEED_THRESHOLD) return "warn"
  return "good"
}

function formatSpeed(speed: number): string {
  return Number.isInteger(speed) ? `${speed}` : speed.toFixed(0)
}

export function TickSpeedBar({
  speeds,
  weaponSpeedCap,
  ticksWindow,
}: TickSpeedBarProps) {
  if (!speeds || speeds.length === 0) {
    return null
  }
  return (
    <section
      data-testid="tick-speed-bar"
      aria-label="Speed at each tick before fire"
      className="flex flex-col gap-1.5"
    >
      <header className="flex items-center justify-between text-[10px] uppercase tracking-wide text-white/50">
        <span>Speed window</span>
        <span className="font-mono">cap {formatSpeed(weaponSpeedCap)} u/s</span>
      </header>
      <div
        role="img"
        aria-label={`Tick speeds: ${speeds.map(formatSpeed).join(", ")} u/s; cap ${formatSpeed(weaponSpeedCap)} u/s`}
        className="grid gap-1"
        style={{
          gridTemplateColumns: `repeat(${speeds.length}, minmax(0, 1fr))`,
        }}
      >
        {speeds.map((speed, i) => {
          const status = statusForSpeed(speed, weaponSpeedCap)
          return (
            <div
              key={i}
              data-testid={`tick-speed-segment-${i}`}
              data-status={status}
              className={cn(
                "flex flex-col items-center justify-center rounded border px-1.5 py-1.5 text-[11px] font-mono tabular-nums text-white/90",
              )}
              style={{
                backgroundColor: `${STATUS_COLOR[status]}26`,
                borderColor: `${STATUS_COLOR[status]}66`,
              }}
            >
              <span className="font-semibold leading-none">
                {formatSpeed(speed)}
              </span>
              <span className="text-[9px] uppercase tracking-wider text-white/50">
                u/s
              </span>
            </div>
          )
        })}
      </div>
      <footer className="flex items-center justify-between text-[10px] tabular-nums text-white/40">
        <span>
          {ticksWindow && ticksWindow.length === speeds.length
            ? `ticks ${ticksWindow[0]}–${ticksWindow[ticksWindow.length - 1]}`
            : `${speeds.length} sampled tick${speeds.length === 1 ? "" : "s"}`}
        </span>
        <span className="flex items-center gap-2">
          <Legend color={STATUS_COLOR.good} label="stopped" />
          <Legend color={STATUS_COLOR.warn} label="slow" />
          <Legend color={STATUS_COLOR.bad} label="moving" />
        </span>
      </footer>
    </section>
  )
}

function Legend({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1">
      <span
        aria-hidden
        className="inline-block h-1.5 w-1.5 rounded-sm"
        style={{ backgroundColor: color }}
      />
      <span>{label}</span>
    </span>
  )
}
