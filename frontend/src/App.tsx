import { lazy, Suspense } from "react"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { ThemeProvider } from "@/components/providers/theme-provider"
import { QueryProvider } from "@/components/providers/query-provider"
import RootLayout from "@/routes/root"

const DemosPage = lazy(() => import("@/routes/demos"))
const DemoViewerPage = lazy(() => import("@/routes/demo-viewer"))
const HeatmapsPage = lazy(() => import("@/routes/heatmaps"))
const StratsPage = lazy(() => import("@/routes/strats"))
const StratBoardPage = lazy(() => import("@/routes/strat-board"))
const LineupsPage = lazy(() => import("@/routes/lineups"))
const SettingsPage = lazy(() => import("@/routes/settings"))

function RouteFallback() {
  return (
    <div className="flex h-screen items-center justify-center">
      <div className="h-8 w-8 animate-spin rounded-full border-2 border-muted border-t-foreground" />
    </div>
  )
}

function App() {
  return (
    <ThemeProvider>
      <QueryProvider>
        <BrowserRouter>
          <Suspense fallback={<RouteFallback />}>
            <Routes>
              <Route path="/" element={<RootLayout />}>
                <Route index element={<Navigate to="/demos" replace />} />
                <Route path="demos" element={<DemosPage />} />
                <Route path="demos/:id" element={<DemoViewerPage />} />
                <Route path="heatmaps" element={<HeatmapsPage />} />
                <Route path="strats" element={<StratsPage />} />
                <Route path="strats/:id" element={<StratBoardPage />} />
                <Route path="lineups" element={<LineupsPage />} />
                <Route path="settings" element={<SettingsPage />} />
              </Route>
            </Routes>
          </Suspense>
        </BrowserRouter>
      </QueryProvider>
    </ThemeProvider>
  )
}

export default App
