import { Swords } from "lucide-react"
import { cn } from "@/lib/utils"
import { getWeaponIconPath } from "@/lib/viewer/weapon-icons"

// Renders a CS2 weapon as an SVG sprite from /equipment. Falls back to a
// generic crossed-swords lucide icon for unmapped names (rare modded weapons,
// future additions, etc.) so layout is never empty.
export function WeaponIcon({
  name,
  className,
}: {
  name: string | null | undefined
  className?: string
}) {
  const path = getWeaponIconPath(name)
  if (!path) {
    return <Swords className={cn("opacity-70", className)} />
  }
  return (
    <img
      src={path}
      alt={name ?? ""}
      draggable={false}
      data-testid={`weapon-icon-${name}`}
      className={cn("h-3.5 w-auto select-none object-contain", className)}
    />
  )
}
