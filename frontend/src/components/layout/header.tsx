import { Link, useLocation } from "react-router-dom"
import { RefreshCw } from "lucide-react"
import { cn } from "@/lib/utils"
import { navItems } from "@/components/layout/sidebar"
import { useAuth } from "@/components/providers/auth-provider"
import { useFaceitSync } from "@/hooks/use-faceit-sync"

export interface HeaderProps {
  title?: string
  subtitle?: string
  actions?: React.ReactNode
}

function deriveTitle(pathname: string): string {
  const match = navItems.find(
    (item) =>
      item.href === pathname ||
      (item.href !== "/" && pathname.startsWith(`${item.href}/`)),
  )
  if (match) return match.label
  if (pathname.startsWith("/matches/")) return "Match detail"
  if (pathname.startsWith("/demos/")) return "Demo viewer"
  return "Oversite"
}

function SyncButton() {
  const { isAuthenticated } = useAuth()
  const sync = useFaceitSync()

  return (
    <button
      type="button"
      className="header-sync"
      onClick={() => sync.mutate()}
      disabled={sync.isPending || !isAuthenticated}
      aria-label="Sync Faceit data"
    >
      <RefreshCw
        className={cn("h-3.5 w-3.5", sync.isPending && "animate-spin")}
      />
      <span>{sync.isPending ? "Syncing…" : "Sync"}</span>
    </button>
  )
}

export function Header({ title, subtitle, actions }: HeaderProps = {}) {
  const { pathname } = useLocation()
  const resolvedTitle = title ?? deriveTitle(pathname)
  const isHome = pathname === "/" || pathname === "/dashboard"

  return (
    <div className="main-header">
      <div className="min-w-0">
        <div className="page-crumbs" aria-label="Breadcrumb">
          <Link to="/dashboard" className="hover:text-[var(--text)]">
            Home
          </Link>
          {!isHome ? (
            <>
              <span className="crumb-sep">›</span>
              <span className="crumb-current">{resolvedTitle}</span>
            </>
          ) : (
            <>
              <span className="crumb-sep">›</span>
              <span className="crumb-current">Dashboard</span>
            </>
          )}
        </div>
        <div className="page-title">{resolvedTitle}</div>
        {subtitle ? <div className="page-subtitle">{subtitle}</div> : null}
      </div>
      <div className="flex items-center gap-2">
        <SyncButton />
        {actions}
      </div>
    </div>
  )
}
