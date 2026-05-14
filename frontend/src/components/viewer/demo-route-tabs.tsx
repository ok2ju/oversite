import { useLocation, useNavigate } from "react-router-dom"
import { cn } from "@/lib/utils"

interface DemoRouteTabsProps {
  demoId: string
}

type TabValue = "viewer" | "analysis"

// Two-tab strip mounted on both /demos/:id and /demos/:id/analysis. Active
// state derives from the current pathname so refreshing on either route
// highlights the right tab without a parent layout owning the state. Clicks
// route via react-router; the per-page useEffect cleanups in each route
// reset useViewerStore so component state doesn't leak across the swap.
export function DemoRouteTabs({ demoId }: DemoRouteTabsProps) {
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const active: TabValue = pathname.endsWith("/analysis")
    ? "analysis"
    : "viewer"

  const select = (value: TabValue) => {
    if (value === active) return
    if (value === "analysis") {
      navigate(`/demos/${demoId}/analysis`)
    } else {
      navigate(`/demos/${demoId}`)
    }
  }

  return (
    <div
      data-testid="demo-route-tabs"
      role="tablist"
      className="inline-flex h-7 items-center gap-0.5 rounded-md bg-white/[0.04] p-0.5"
    >
      <RouteTab
        value="viewer"
        active={active}
        onSelect={select}
        testId="demo-route-tab-viewer"
        label="Viewer"
      />
      <RouteTab
        value="analysis"
        active={active}
        onSelect={select}
        testId="demo-route-tab-analysis"
        label="Analysis"
      />
    </div>
  )
}

function RouteTab({
  value,
  active,
  onSelect,
  testId,
  label,
}: {
  value: TabValue
  active: TabValue
  onSelect: (v: TabValue) => void
  testId: string
  label: string
}) {
  const isActive = value === active
  return (
    <button
      type="button"
      role="tab"
      aria-selected={isActive}
      data-state={isActive ? "active" : "inactive"}
      data-testid={testId}
      onClick={() => onSelect(value)}
      className={cn(
        "inline-flex h-6 items-center justify-center rounded-[5px] px-3 text-[12px] font-medium leading-none transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-white/30",
        isActive
          ? "bg-white text-black shadow-[0_1px_2px_rgba(0,0,0,0.45)]"
          : "text-white/65 hover:bg-white/[0.06] hover:text-white",
      )}
    >
      {label}
    </button>
  )
}
