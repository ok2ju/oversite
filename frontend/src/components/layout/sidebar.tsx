import { useEffect, useState } from "react"
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
  ExternalLink,
} from "lucide-react"
import { BrowserOpenURL } from "@wailsjs/runtime/runtime"
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
  disabled?: boolean
}

const mainItems: NavItem[] = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/demos", label: "Demos", icon: Folder },
  { href: "/heatmaps", label: "Heatmaps", icon: Crosshair, disabled: true },
  { href: "/strats", label: "Strategies", icon: Goal, disabled: true },
  { href: "/lineups", label: "Lineups", icon: Star, disabled: true },
]

const workspaceItems: NavItem[] = [
  { href: "/settings", label: "Settings", icon: Settings, disabled: true },
]

function faceitProfileUrl(nickname: string | null | undefined): string | null {
  if (!nickname) return null
  return `https://www.faceit.com/en/players/${encodeURIComponent(nickname)}`
}

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
  const avatarUrl = profile?.avatar_url ?? null
  const [avatarBroken, setAvatarBroken] = useState(false)
  useEffect(() => {
    setAvatarBroken(false)
  }, [avatarUrl])
  const showAvatar = Boolean(avatarUrl) && !avatarBroken

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
        {(() => {
          const url = faceitProfileUrl(profile?.nickname)
          const disabled = !url
          return (
            <button
              type="button"
              className={cn(
                "nav-item w-full appearance-none border-0 bg-transparent text-left font-[inherit]",
                disabled && "disabled",
              )}
              onClick={() => url && BrowserOpenURL(url)}
              disabled={disabled}
              title={
                disabled ? "Sign in to view your Faceit profile" : undefined
              }
            >
              <Link2 className="h-4 w-4" />
              <span>Faceit account</span>
              <ExternalLink className="ml-auto h-3.5 w-3.5 opacity-70" />
            </button>
          )
        })()}
      </div>

      <div className="side-user">
        {showAvatar ? (
          <img
            src={avatarUrl as string}
            alt={`${profile?.nickname ?? "Player"} avatar`}
            className="side-user-avatar object-cover"
            referrerPolicy="no-referrer"
            onError={() => setAvatarBroken(true)}
          />
        ) : (
          <div className="side-user-avatar">{initial}</div>
        )}
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
