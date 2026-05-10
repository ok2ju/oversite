import { useMemo } from "react"

export interface SparklinePoint {
  value: number
  // Optional label (used as title/aria for the point); not rendered visually.
  label?: string
}

interface SparklineProps {
  points: SparklinePoint[]
  width?: number
  height?: number
  color?: string
  // When set, the sparkline reverses its input so newest points render on the
  // right. Coaching trends arrive newest-first; the chart reads naturally
  // left-to-right when reversed.
  newestFirst?: boolean
  className?: string
  ariaLabel?: string
}

// Sparkline — minimal in-flow trend chart. SVG path on a normalized [0,1] grid;
// hidden / collapsed when fewer than 2 points (no line to draw). Accessible:
// role="img" + ariaLabel; the bundled <title> echoes the label for screen
// readers and tooltip-on-hover.
export function Sparkline({
  points,
  width = 80,
  height = 18,
  color = "#9bbc5a",
  newestFirst = true,
  className,
  ariaLabel,
}: SparklineProps) {
  const ordered = useMemo(() => {
    if (!points || points.length === 0) return []
    return newestFirst ? [...points].reverse() : points
  }, [points, newestFirst])

  if (ordered.length < 2) {
    return (
      <span
        data-testid="sparkline-empty"
        aria-hidden="true"
        className={className}
        style={{
          display: "inline-block",
          width,
          height,
          opacity: 0.25,
          fontFamily: "monospace",
          fontSize: 10,
          color,
        }}
      >
        —
      </span>
    )
  }

  const values = ordered.map((p) => p.value)
  const min = Math.min(...values)
  const max = Math.max(...values)
  const range = max - min || 1

  const stepX = ordered.length > 1 ? width / (ordered.length - 1) : 0
  const path = ordered
    .map((p, i) => {
      const x = i * stepX
      const y = height - ((p.value - min) / range) * height
      return `${i === 0 ? "M" : "L"}${x.toFixed(2)},${y.toFixed(2)}`
    })
    .join(" ")

  const lastX = (ordered.length - 1) * stepX
  const lastY =
    height - ((ordered[ordered.length - 1].value - min) / range) * height

  return (
    <svg
      data-testid="sparkline"
      role="img"
      aria-label={ariaLabel ?? "trend"}
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      className={className}
    >
      <title>{ariaLabel ?? "trend"}</title>
      <path
        d={path}
        stroke={color}
        strokeWidth={1.25}
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx={lastX} cy={lastY} r={1.5} fill={color} />
    </svg>
  )
}
