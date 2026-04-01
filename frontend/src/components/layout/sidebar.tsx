"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { LayoutDashboard, Film, Flame, Map, Crosshair, Settings } from "lucide-react"
import { cn } from "@/lib/utils"

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/demos", label: "Demos", icon: Film },
  { href: "/heatmaps", label: "Heatmaps", icon: Flame },
  { href: "/strats", label: "Strategies", icon: Map },
  { href: "/lineups", label: "Lineups", icon: Crosshair },
  { href: "/settings", label: "Settings", icon: Settings },
]

export function Sidebar() {
  const pathname = usePathname()

  return (
    <aside className="flex h-full w-64 flex-col border-r bg-card">
      <div className="flex h-14 items-center border-b px-6">
        <Link href="/dashboard" className="text-lg font-bold">Oversite</Link>
      </div>
      <nav className="flex-1 space-y-1 p-4">
        {navItems.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className={cn(
              "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
              pathname === item.href || pathname?.startsWith(item.href + "/")
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:bg-muted hover:text-foreground"
            )}
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </Link>
        ))}
      </nav>
    </aside>
  )
}

export { navItems }
