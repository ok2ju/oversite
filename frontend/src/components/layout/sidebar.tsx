import { useEffect } from "react"
import { NavLink } from "react-router-dom"
import {
  Folder,
  Crosshair,
  Goal,
  Star,
  Settings,
  Link2,
  PanelLeftClose,
  PanelLeft,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useDemoCount } from "@/hooks/use-demos"
import { ReticleGlyph } from "@/components/brand/logo"
import { useUiStore } from "@/stores/ui"

type NavItem = {
  href: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  badge?: number
  disabled?: boolean
}

const mainItems: NavItem[] = [
  { href: "/demos", label: "Demos", icon: Folder },
  { href: "/heatmaps", label: "Heatmaps", icon: Crosshair, disabled: true },
  { href: "/strats", label: "Strategies", icon: Goal, disabled: true },
  { href: "/lineups", label: "Lineups", icon: Star, disabled: true },
]

const workspaceItems: NavItem[] = [
  { href: "/settings", label: "Settings", icon: Settings, disabled: true },
  { href: "/sources", label: "Sources", icon: Link2, disabled: true },
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
        title={item.label}
      >
        <Icon className="h-4 w-4 nav-icon" />
        <span className="nav-text">{item.label}</span>
      </span>
    )
  }

  return (
    <NavLink
      to={item.href}
      className={({ isActive }) => cn("nav-item", isActive && "active")}
      title={item.label}
    >
      <Icon className="h-4 w-4 nav-icon" />
      <span className="nav-text">{item.label}</span>
      {badge != null && badge > 0 ? (
        <span className="nav-badge tabular">{badge}</span>
      ) : null}
    </NavLink>
  )
}

export function Sidebar() {
  const demoCountQuery = useDemoCount()
  const demoCount = demoCountQuery.data ?? 0
  const sidebarCollapsed = useUiStore((s) => s.sidebarCollapsed)
  const toggleSidebarCollapsed = useUiStore((s) => s.toggleSidebarCollapsed)

  useEffect(() => {
    document.documentElement.dataset.sidebar = sidebarCollapsed
      ? "icons"
      : "labeled"
    return () => {
      delete document.documentElement.dataset.sidebar
    }
  }, [sidebarCollapsed])

  const CollapseIcon = sidebarCollapsed ? PanelLeft : PanelLeftClose

  return (
    <aside className="sidebar">
      <div className="side-brand">
        <ReticleGlyph size={22} color="var(--text)" accent="var(--accent)" />
        <span className="brand-label">Oversite</span>
        <button
          type="button"
          className="side-collapse-btn"
          onClick={toggleSidebarCollapsed}
          title={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
          aria-label={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
          aria-pressed={sidebarCollapsed}
        >
          <CollapseIcon className="h-3.5 w-3.5" />
        </button>
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
