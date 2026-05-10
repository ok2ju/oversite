// MousePath renders a polar SVG of the player's mouse trajectory leading up
// to a fire-related mistake. Each tick is plotted at polar coordinates
// (yaw_delta_to_target_deg, speed). The center is the ideal fire moment —
// crosshair on target, player perfectly stopped — so the closer the last
// dot is to center, the cleaner the duel. Concentric rings mark common
// speed bands; dots are colored by speed status (good = stopped, warn =
// slow, bad = moving) using the same thresholds as TickSpeedBar so the two
// forensic panels speak the same color language. Falls back to yaw-only
// when pitch is missing (older demos pre-P3-1).
//
// See plans/analysis-overhaul.md §4.5, §7 (image 10).

interface MousePathProps {
  // Per-tick yaw delta to target in degrees. 0 = pointing at target.
  yaws: number[]
  // Per-tick planar speed in u/s. Same array as TickSpeedBar consumes.
  speeds: number[]
  // Optional per-tick pitch delta in degrees. When omitted the plot
  // continues to render with yaw + speed; only the aria description and a
  // data-attribute change. Used by callers as a forward-compat hook so we
  // can layer pitch into the visualization later without a breaking API.
  pitches?: number[] | null
  // Weapon stationary-cap (u/s). Speeds above this trigger inaccuracy.
  weaponSpeedCap: number
  // Concentric speed rings (u/s). Default mirrors the plan §3.2 spec.
  rings?: number[]
}

const STATUS_COLOR = {
  good: "#9bbc5a",
  warn: "#ffc233",
  bad: "#f87171",
} as const

type Status = keyof typeof STATUS_COLOR

const STILL_SPEED_THRESHOLD = 10
const DEFAULT_RINGS = [10, 50, 100, 150]
const VIEWBOX = 240
const CENTER = VIEWBOX / 2
const PADDING = 28
const PLOT_RADIUS = CENTER - PADDING

function statusForSpeed(speed: number, cap: number): Status {
  if (speed > cap) return "bad"
  if (speed > STILL_SPEED_THRESHOLD) return "warn"
  return "good"
}

function formatSpeed(speed: number): string {
  return Number.isInteger(speed) ? `${speed}` : speed.toFixed(0)
}

export function MousePath({
  yaws,
  speeds,
  pitches,
  weaponSpeedCap,
  rings = DEFAULT_RINGS,
}: MousePathProps) {
  if (!yaws || !speeds || yaws.length === 0 || speeds.length === 0) {
    return null
  }
  const n = Math.min(yaws.length, speeds.length)
  const ringMax = rings.length > 0 ? Math.max(...rings) : 150
  // Scale so the outermost ring fits, but stretch when an outlier tick
  // exceeds the largest configured ring — better to show the spike than
  // clip it into the boundary and lie about how fast the player was.
  const maxSpeed = Math.max(ringMax, ...speeds.slice(0, n))
  const pitchAvailable = Boolean(pitches && pitches.length > 0)

  const project = (yawDeg: number, speed: number) => {
    const theta = (yawDeg * Math.PI) / 180
    const r = (speed / maxSpeed) * PLOT_RADIUS
    return {
      x: CENTER + r * Math.sin(theta),
      y: CENTER - r * Math.cos(theta),
    }
  }

  const points = Array.from({ length: n }, (_, i) => {
    const projected = project(yaws[i], speeds[i])
    return {
      index: i,
      yaw: yaws[i],
      speed: speeds[i],
      status: statusForSpeed(speeds[i], weaponSpeedCap),
      x: projected.x,
      y: projected.y,
    }
  })

  const pathD = points
    .map((p, i) => `${i === 0 ? "M" : "L"} ${p.x.toFixed(2)} ${p.y.toFixed(2)}`)
    .join(" ")

  const ariaLabel = `Mouse path: ${n} tick${n === 1 ? "" : "s"} leading to fire. ${
    pitchAvailable
      ? "Yaw and pitch tracked."
      : "Yaw tracked; pitch unavailable."
  }`

  return (
    <section
      data-testid="mouse-path"
      data-pitch={pitchAvailable ? "available" : "missing"}
      aria-label="Mouse path leading to fire"
      className="flex flex-col gap-1.5"
    >
      <header className="flex items-center justify-between text-[10px] uppercase tracking-wide text-white/50">
        <span>Mouse path</span>
        <span className="font-mono">cap {formatSpeed(weaponSpeedCap)} u/s</span>
      </header>
      <svg
        role="img"
        aria-label={ariaLabel}
        viewBox={`0 0 ${VIEWBOX} ${VIEWBOX}`}
        className="aspect-square w-full max-w-[280px] self-center"
      >
        <line
          x1={CENTER}
          y1={PADDING}
          x2={CENTER}
          y2={VIEWBOX - PADDING}
          stroke="rgba(255,255,255,0.06)"
          strokeWidth={0.5}
        />
        <line
          x1={PADDING}
          y1={CENTER}
          x2={VIEWBOX - PADDING}
          y2={CENTER}
          stroke="rgba(255,255,255,0.06)"
          strokeWidth={0.5}
        />

        {rings.map((speed) => {
          const r = (speed / maxSpeed) * PLOT_RADIUS
          return (
            <g key={speed}>
              <circle
                data-testid={`mouse-path-ring-${speed}`}
                cx={CENTER}
                cy={CENTER}
                r={r}
                fill="none"
                stroke="rgba(255,255,255,0.14)"
                strokeWidth={0.6}
                strokeDasharray="2,3"
              />
              <text
                x={CENTER + 3}
                y={CENTER - r - 2}
                fontSize={7}
                fill="rgba(255,255,255,0.45)"
                fontFamily="ui-monospace, SFMono-Regular, monospace"
              >
                {formatSpeed(speed)} u/s
              </text>
            </g>
          )
        })}

        <path
          data-testid="mouse-path-line"
          d={pathD}
          fill="none"
          stroke="rgba(255,255,255,0.35)"
          strokeWidth={0.8}
        />

        {points.map((p) => (
          <g key={p.index}>
            <circle
              data-testid={`mouse-path-dot-${p.index}`}
              data-status={p.status}
              cx={p.x}
              cy={p.y}
              r={4}
              fill={STATUS_COLOR[p.status]}
              stroke="rgba(0,0,0,0.6)"
              strokeWidth={0.8}
            />
            <text
              x={p.x + 5}
              y={p.y - 5}
              fontSize={8}
              fill="rgba(255,255,255,0.85)"
              fontFamily="ui-monospace, SFMono-Regular, monospace"
            >
              {p.index + 1}
            </text>
          </g>
        ))}

        <circle cx={CENTER} cy={CENTER} r={1.5} fill="rgba(255,122,26,0.9)" />
      </svg>
      <p className="text-[10px] leading-snug text-white/45">
        First bullet at center. Outer dots = mouse path leading up to the fire
        (yaw delta vs. target).
      </p>
      <span className="flex items-center gap-2 text-[10px] tabular-nums text-white/40">
        <Legend color={STATUS_COLOR.good} label="stopped" />
        <Legend color={STATUS_COLOR.warn} label="slow" />
        <Legend color={STATUS_COLOR.bad} label="moving" />
      </span>
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
