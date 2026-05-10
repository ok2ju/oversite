import { useLocation, useNavigate } from "react-router-dom"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"

interface DemoRouteTabsProps {
  demoId: string
}

// Two-tab strip mounted on both /demos/:id and /demos/:id/analysis. Active
// state derives from the current pathname so refreshing on either route
// highlights the right tab without a parent layout owning the state. Clicks
// route via react-router; the per-page useEffect cleanups in each route
// reset useViewerStore so component state doesn't leak across the swap.
export function DemoRouteTabs({ demoId }: DemoRouteTabsProps) {
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const active = pathname.endsWith("/analysis") ? "analysis" : "viewer"

  return (
    <Tabs
      data-testid="demo-route-tabs"
      value={active}
      onValueChange={(value) => {
        if (value === active) return
        if (value === "analysis") {
          navigate(`/demos/${demoId}/analysis`)
        } else {
          navigate(`/demos/${demoId}`)
        }
      }}
    >
      <TabsList>
        <TabsTrigger value="viewer" data-testid="demo-route-tab-viewer">
          Viewer
        </TabsTrigger>
        <TabsTrigger value="analysis" data-testid="demo-route-tab-analysis">
          Analysis
        </TabsTrigger>
      </TabsList>
    </Tabs>
  )
}
