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
  // The 2D viewer owns its own chrome (MatchHeader + in-viewer back button),
  // so the global app header is suppressed here only. /demos/:id/analysis and
  // every other route keep the breadcrumbed header.
  const isDemoViewer = useMatch("/demos/:id") !== null

  return (
    <div className="app-shell">
      <div className="app-body">
        <Sidebar />
        <div
          className={
            isDemoViewer ? "main-wrap main-wrap--no-header" : "main-wrap"
          }
        >
          {isDemoViewer ? null : <Header actions={actions} />}
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
