import { Outlet } from "react-router-dom"
import { Sidebar } from "@/components/layout/sidebar"
import { Header } from "@/components/layout/header"

export default function RootLayout() {
  return (
    <div className="app-shell">
      <div className="app-body">
        <Sidebar />
        <div className="main-wrap">
          <Header />
          <main className="main-body">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  )
}
