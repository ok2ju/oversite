import { NavLink } from "react-router-dom"
import {
  LayoutDashboard,
  Folder,
  Crosshair,
  Goal,
  Star,
  Settings,
  Link2,
  LogOut,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useDemos } from "@/hooks/use-demos"
import { useFaceitProfile } from "@/hooks/use-faceit"
import { useAuth } from "@/components/providers/auth-provider"
import { Logo } from "@/components/brand/logo"

type NavItem = {
  href: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  badge?: number
}

const mainItems: NavItem[] = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/demos", label: "Demos", icon: Folder },
  { href: "/heatmaps", label: "Heatmaps", icon: Crosshair },
  { href: "/strats", label: "Strategies", icon: Goal },
  { href: "/lineups", label: "Lineups", icon: Star },
]

const workspaceItems: NavItem[] = [
  { href: "/settings", label: "Settings", icon: Settings },
  { href: "/login", label: "Faceit account", icon: Link2 },
]

function SideNavLink({
  item,
  badge,
}: {
  item: NavItem
  badge?: number | null
}) {
  const Icon = item.icon
  return (
    <NavLink
      to={item.href}
      end={item.href === "/dashboard"}
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
  const demos = useDemos(1, 1)
  const demoCount = demos.data?.meta.total ?? 0
  const { data: profile } = useFaceitProfile()
  const { isAuthenticated, logout } = useAuth()

  const initial = profile?.nickname?.[0]?.toUpperCase() ?? "?"

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

      <div className="side-user">
        <div className="side-user-avatar">{initial}</div>
        <div className="min-w-0">
          <div className="side-user-name truncate">
            {profile?.nickname ?? "Not connected"}
          </div>
          <div className="side-user-sub">
            {profile?.level != null && profile?.elo != null
              ? `Lv ${profile.level} · ${profile.elo.toLocaleString()} Elo`
              : "Sign in with Faceit"}
          </div>
        </div>
        <button
          type="button"
          className="side-user-logout"
          onClick={() => logout()}
          disabled={!isAuthenticated}
          aria-label="Log out"
          title="Log out"
        >
          <LogOut className="h-4 w-4" />
        </button>
      </div>
    </aside>
  )
}

export const navItems = [...mainItems, ...workspaceItems]
