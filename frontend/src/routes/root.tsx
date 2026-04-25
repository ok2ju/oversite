import { useMemo, useState, type ReactNode } from "react"
import { Outlet } from "react-router-dom"
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

  return (
    <div className="app-shell">
      <div className="app-body">
        <Sidebar />
        <div className="main-wrap">
          <Header actions={actions} />
          <main className="main-body">
            <Outlet context={outletContext} />
          </main>
        </div>
      </div>
    </div>
  )
}
