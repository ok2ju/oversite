import { useMemo, useState, type ReactNode } from "react"
import { Outlet, useMatch } from "react-router-dom"
import { Sidebar } from "@/components/layout/sidebar"
import { Header } from "@/components/layout/header"

export interface HeaderActionsContext {
  setHeaderActions: (actions: ReactNode | null) => void
}

export default function RootLayout() {
  const [actions, setActions] = useState<ReactNode | null>(null)
  const outletContext = useMemo<HeaderActionsContext>(
    () => ({ setHeaderActions: setActions }),
    [],
  )
  // The 2D viewer and match overview own their own chrome (back button +
  // contextual topbar), so the global app header is suppressed for both.
  // Only the viewer needs a flush, non-scrolling body (the map fills the
  // frame); the overview keeps the standard padded, scrollable body.
  const isDemoViewer = useMatch("/demos/:id") !== null
  const isMatchOverview = useMatch("/demos/:id/overview") !== null
  const hideHeader = isDemoViewer || isMatchOverview

  return (
    <div className="app-shell">
      <div className="app-body">
        <Sidebar />
        <div
          className={
            hideHeader ? "main-wrap main-wrap--no-header" : "main-wrap"
          }
        >
          {hideHeader ? null : <Header actions={actions} />}
          <main
            className={
              isDemoViewer ? "main-body main-body--flush" : "main-body"
            }
          >
            <Outlet context={outletContext} />
          </main>
        </div>
      </div>
    </div>
  )
}
