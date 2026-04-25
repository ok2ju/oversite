import type { CSSProperties } from "react"

const BRAND_ACCENT = "oklch(0.72 0.18 45)"

interface ReticleGlyphProps {
  size?: number
  color?: string
  accent?: string
  stroke?: number
  className?: string
  title?: string
}

export function ReticleGlyph({
  size = 22,
  color = "currentColor",
  accent = BRAND_ACCENT,
  stroke,
  className,
  title,
}: ReticleGlyphProps) {
  const c = size / 2
  const s = stroke ?? Math.max(1.6, size * 0.09)
  const r = (size - s) / 2 - size * 0.04
  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      fill="none"
      className={className}
      role={title ? "img" : "presentation"}
      aria-hidden={title ? undefined : true}
      aria-label={title}
    >
      {title ? <title>{title}</title> : null}
      <circle cx={c} cy={c} r={r} stroke={color} strokeWidth={s} />
      <circle
        cx={c}
        cy={c}
        r={r * 0.42}
        stroke={color}
        strokeWidth={s * 0.7}
        opacity={0.55}
      />
      <line
        x1={c}
        y1={0}
        x2={c}
        y2={size * 0.22}
        stroke={color}
        strokeWidth={s}
        strokeLinecap="square"
      />
      <line
        x1={c}
        y1={size}
        x2={c}
        y2={size * 0.78}
        stroke={color}
        strokeWidth={s}
        strokeLinecap="square"
      />
      <line
        x1={0}
        y1={c}
        x2={size * 0.22}
        y2={c}
        stroke={color}
        strokeWidth={s}
        strokeLinecap="square"
      />
      <line
        x1={size}
        y1={c}
        x2={size * 0.78}
        y2={c}
        stroke={color}
        strokeWidth={s}
        strokeLinecap="square"
      />
      <circle cx={c} cy={c} r={size * 0.08} fill={accent} />
    </svg>
  )
}

interface LogoProps {
  iconSize?: number
  fontSize?: number
  accent?: string
  color?: string
  className?: string
  style?: CSSProperties
}

export function Logo({
  iconSize = 22,
  fontSize = 14,
  accent = BRAND_ACCENT,
  color = "currentColor",
  className,
  style,
}: LogoProps) {
  return (
    <span
      className={className}
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: Math.max(6, iconSize * 0.32),
        color,
        fontFamily: "'Inter Tight', 'Inter', sans-serif",
        fontWeight: 700,
        letterSpacing: "-0.025em",
        fontSize,
        lineHeight: 1,
        whiteSpace: "nowrap",
        ...style,
      }}
    >
      <ReticleGlyph size={iconSize} color={color} accent={accent} />
      <span>Oversite</span>
    </span>
  )
}
