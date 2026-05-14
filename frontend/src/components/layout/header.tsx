import { Link, useLocation } from "react-router-dom"
import { ChevronRight } from "lucide-react"
import { navItems } from "@/components/layout/sidebar"

export interface HeaderProps {
  title?: string
  subtitle?: string
  actions?: React.ReactNode
}

function deriveTitle(pathname: string): string {
  if (pathname.startsWith("/demos/")) return "Demo viewer"
  if (pathname === "/demos") return "Demos library"
  const match = navItems.find(
    (item) =>
      item.href === pathname ||
      (item.href !== "/" && pathname.startsWith(`${item.href}/`)),
  )
  if (match) return match.label
  return "Oversite"
}

export function Header({ title, subtitle, actions }: HeaderProps = {}) {
  const { pathname } = useLocation()
  const resolvedTitle = title ?? deriveTitle(pathname)
  const isHome = pathname === "/demos"

  return (
    <div className="main-header">
      <div className="min-w-0">
        {isHome ? null : (
          <div className="page-crumbs" aria-label="Breadcrumb">
            <Link to="/demos" className="hover:text-[var(--text)]">
              Home
            </Link>
            <span className="crumb-sep" aria-hidden>
              <ChevronRight className="h-3 w-3" />
            </span>
            <span className="crumb-current">{resolvedTitle}</span>
          </div>
        )}
        <div className="page-title">{resolvedTitle}</div>
        {subtitle ? <div className="page-subtitle">{subtitle}</div> : null}
      </div>
      {actions ? (
        <div className="flex items-center gap-2">{actions}</div>
      ) : null}
    </div>
  )
}
