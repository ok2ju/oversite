import { Link, useLocation } from "react-router-dom"
import { navItems } from "@/components/layout/sidebar"

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
      {actions ? (
        <div className="flex items-center gap-2">{actions}</div>
      ) : null}
    </div>
  )
}
