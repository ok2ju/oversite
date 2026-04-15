import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { ThemeProvider } from "@/components/providers/theme-provider"
import { QueryProvider } from "@/components/providers/query-provider"
import { AuthProvider } from "@/components/providers/auth-provider"
import RootLayout from "@/routes/root"
import DashboardPage from "@/routes/dashboard"
import DemosPage from "@/routes/demos"
import DemoViewerPage from "@/routes/demo-viewer"
import HeatmapsPage from "@/routes/heatmaps"
import StratsPage from "@/routes/strats"
import StratBoardPage from "@/routes/strat-board"
import LineupsPage from "@/routes/lineups"
import SettingsPage from "@/routes/settings"
import LoginPage from "@/routes/login"
import CallbackPage from "@/routes/callback"

function App() {
  return (
    <ThemeProvider defaultTheme="dark">
      <QueryProvider>
        <BrowserRouter>
          <AuthProvider>
            <Routes>
              <Route path="/login" element={<LoginPage />} />
              <Route path="/callback" element={<CallbackPage />} />
              <Route path="/" element={<RootLayout />}>
                <Route index element={<Navigate to="/dashboard" replace />} />
                <Route path="dashboard" element={<DashboardPage />} />
                <Route path="demos" element={<DemosPage />} />
                <Route path="demos/:id" element={<DemoViewerPage />} />
                <Route path="heatmaps" element={<HeatmapsPage />} />
                <Route path="strats" element={<StratsPage />} />
                <Route path="strats/:id" element={<StratBoardPage />} />
                <Route path="lineups" element={<LineupsPage />} />
                <Route path="settings" element={<SettingsPage />} />
              </Route>
            </Routes>
          </AuthProvider>
        </BrowserRouter>
      </QueryProvider>
    </ThemeProvider>
  )
}

export default App
