import { useState } from "react"
import { cn } from "@/lib/utils"

interface MapMeta {
  name: string
  code: string
  gradient: string
}

const MAP_GRADIENTS: Record<string, MapMeta> = {
  mirage: {
    name: "Mirage",
    code: "MIR",
    gradient: "linear-gradient(135deg, #e4c48b, #8a6a3a)",
  },
  inferno: {
    name: "Inferno",
    code: "INF",
    gradient: "linear-gradient(135deg, #e48b6a, #8a3a1a)",
  },
  nuke: {
    name: "Nuke",
    code: "NUK",
    gradient: "linear-gradient(135deg, #cfd2d6, #4a4d52)",
  },
  anubis: {
    name: "Anubis",
    code: "ANB",
    gradient: "linear-gradient(135deg, #6ab0e4, #1a4a8a)",
  },
  ancient: {
    name: "Ancient",
    code: "ANC",
    gradient: "linear-gradient(135deg, #6ac48a, #1a5a3a)",
  },
  dust2: {
    name: "Dust II",
    code: "D2",
    gradient: "linear-gradient(135deg, #e4cd8b, #8a6f3a)",
  },
  vertigo: {
    name: "Vertigo",
    code: "VRT",
    gradient: "linear-gradient(135deg, #b5b5b5, #3d3d3d)",
  },
  train: {
    name: "Train",
    code: "TRN",
    gradient: "linear-gradient(135deg, #8ab0c4, #2d4a5a)",
  },
  overpass: {
    name: "Overpass",
    code: "OVR",
    gradient: "linear-gradient(135deg, #b0d06a, #3a5a1a)",
  },
}

const MAP_ICONS: Record<string, string> = {
  mirage: "/map_icons/map_icon_de_mirage.svg",
  inferno: "/map_icons/map_icon_de_inferno.svg",
  nuke: "/map_icons/map_icon_de_nuke.svg",
  anubis: "/map_icons/map_icon_de_anubis.svg",
  ancient: "/map_icons/map_icon_de_ancient.svg",
  dust2: "/map_icons/map_icon_de_dust2.svg",
  train: "/map_icons/map_icon_de_train.svg",
  overpass: "/map_icons/map_icon_de_overpass.svg",
}

function mapKey(mapName: string): string {
  return mapName.toLowerCase().replace(/^de_/, "")
}

function fallback(name: string): MapMeta {
  return {
    name,
    code: name.slice(0, 3).toUpperCase(),
    gradient: "linear-gradient(135deg, #9aa1ab, #414750)",
  }
}

export function resolveMap(mapName: string): MapMeta {
  const key = mapKey(mapName)
  return MAP_GRADIENTS[key] ?? fallback(mapName)
}

interface MapTileProps {
  mapName: string
  size?: number
  className?: string
}

export function MapTile({ mapName, size = 36, className }: MapTileProps) {
  const meta = resolveMap(mapName)
  const iconSrc = MAP_ICONS[mapKey(mapName)]
  const [iconFailed, setIconFailed] = useState(false)

  if (iconSrc && !iconFailed) {
    return (
      <img
        src={iconSrc}
        alt={meta.name}
        width={size}
        height={size}
        onError={() => setIconFailed(true)}
        className={cn("shrink-0 rounded-[4px] object-contain", className)}
        style={{ width: size, height: size }}
      />
    )
  }

  return (
    <div
      className={cn(
        "grid place-items-center rounded-[4px] font-semibold text-white",
        className,
      )}
      style={{
        width: size,
        height: size,
        background: meta.gradient,
        fontSize: Math.round(size * 0.28),
        letterSpacing: "0.04em",
      }}
      aria-hidden
    >
      {meta.code}
    </div>
  )
}
