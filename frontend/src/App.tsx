import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { ThemeProvider } from "@/components/providers/theme-provider"
import { QueryProvider } from "@/components/providers/query-provider"
import RootLayout from "@/routes/root"
import DemosPage from "@/routes/demos"
import DemoViewerPage from "@/routes/demo-viewer"
import HeatmapsPage from "@/routes/heatmaps"
import StratsPage from "@/routes/strats"
import StratBoardPage from "@/routes/strat-board"
import LineupsPage from "@/routes/lineups"
import SettingsPage from "@/routes/settings"

function App() {
  return (
    <ThemeProvider>
      <QueryProvider>
        <BrowserRouter>
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
        </BrowserRouter>
      </QueryProvider>
    </ThemeProvider>
  )
}

export default App
