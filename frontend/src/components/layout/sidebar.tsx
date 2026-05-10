import { NavLink } from "react-router-dom"
import { Folder, Crosshair, Goal, Star, Settings, Target } from "lucide-react"
import { cn } from "@/lib/utils"
import { useDemoCount } from "@/hooks/use-demos"
import { Logo } from "@/components/brand/logo"

type NavItem = {
  href: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  badge?: number
  disabled?: boolean
}

const mainItems: NavItem[] = [
  { href: "/demos", label: "Demos", icon: Folder },
  { href: "/coaching", label: "Coaching", icon: Target },
  { href: "/heatmaps", label: "Heatmaps", icon: Crosshair, disabled: true },
  { href: "/strats", label: "Strategies", icon: Goal, disabled: true },
  { href: "/lineups", label: "Lineups", icon: Star, disabled: true },
]

const workspaceItems: NavItem[] = [
  { href: "/settings", label: "Settings", icon: Settings, disabled: true },
]

function SideNavLink({
  item,
  badge,
}: {
  item: NavItem
  badge?: number | null
}) {
  const Icon = item.icon

  if (item.disabled) {
    return (
      <span
        className="nav-item disabled"
        aria-disabled="true"
        title="Coming soon"
      >
        <Icon className="h-4 w-4" />
        <span>{item.label}</span>
        <span className="nav-badge tabular">Soon</span>
      </span>
    )
  }

  return (
    <NavLink
      to={item.href}
      className={({ isActive }) => cn("nav-item", isActive && "active")}
    >
      <Icon className="h-4 w-4" />
      <span>{item.label}</span>
      {badge != null && badge > 0 ? (
        <span className="nav-badge tabular">{badge}</span>
      ) : null}
    </NavLink>
  )
}

export function Sidebar() {
  const demoCountQuery = useDemoCount()
  const demoCount = demoCountQuery.data ?? 0

  return (
    <aside className="sidebar">
      <div className="side-brand">
        <Logo iconSize={22} fontSize={14} />
      </div>

      <div className="side-section">
        <div className="side-label">Main</div>
        {mainItems.map((item) => (
          <SideNavLink
            key={item.href}
            item={item}
            badge={item.href === "/demos" ? demoCount : null}
          />
        ))}
      </div>

      <div className="side-section">
        <div className="side-label">Workspace</div>
        {workspaceItems.map((item) => (
          <SideNavLink key={item.href} item={item} />
        ))}
      </div>
    </aside>
  )
}

export const navItems = [...mainItems, ...workspaceItems]
