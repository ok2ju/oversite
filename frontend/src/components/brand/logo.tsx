import type { CSSProperties } from "react"

const BRAND_ACCENT = "#E89B2A"

interface ReticleGlyphProps {
  size?: number
  color?: string
  accent?: string
  /** kept for API compatibility; the aperture mark is solid, not stroked */
  stroke?: number
  className?: string
  title?: string
}

export function ReticleGlyph({
  size = 22,
  color = "currentColor",
  accent = BRAND_ACCENT,
  className,
  title,
}: ReticleGlyphProps) {
  // 5-blade aperture/iris pinwheel around a centered accent dot.
  // Single-path blade, rotated 72° five times in a 0–64 viewBox.
  const blade = "M 32 32 L 32 6 L 22 14 Z"
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      className={className}
      role={title ? "img" : "presentation"}
      aria-hidden={title ? undefined : true}
      aria-label={title}
    >
      {title ? <title>{title}</title> : null}
      <g>
        {[0, 1, 2, 3, 4].map((i) => (
          <path
            key={i}
            d={blade}
            fill={color}
            transform={`rotate(${i * 72} 32 32)`}
          />
        ))}
      </g>
      <circle cx="32" cy="32" r="4.5" fill={accent} />
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
