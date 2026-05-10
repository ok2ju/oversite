import { useEffect } from "react"
import { useParams } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { useDemo } from "@/hooks/use-demo"
import { useViewerStore } from "@/stores/viewer"
import { AnalysisOverallGauge } from "@/components/viewer/analysis-overall-gauge"
import { CategoryCard } from "@/components/viewer/category-card"
import { RoundTradeBars } from "@/components/viewer/round-trade-bars"
import { DemoRouteTabs } from "@/components/viewer/demo-route-tabs"

// Standalone analysis page mounted at /demos/:id/analysis. Reuses the slice-5
// gauge and Trades card so the per-round bar chart, the overall score, and
// the category card stay rendered against the same store-backed
// (demoId, selectedPlayerSteamId) pair without component duplication.
export default function DemoAnalysisPage() {
  const { id } = useParams<{ id: string }>()
  const { data: demo, isLoading, isError } = useDemo(id)
  const initDemo = useViewerStore((s) => s.initDemo)
  const reset = useViewerStore((s) => s.reset)

  useEffect(() => {
    if (!demo) return
    initDemo({
      id: String(demo.id),
      mapName: demo.map_name,
      totalTicks: demo.total_ticks,
      tickRate: demo.tick_rate,
    })
  }, [demo, initDemo])

  useEffect(() => {
    return () => {
      reset()
    }
  }, [reset])

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError || !demo) {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-muted-foreground">
          Demo not found or failed to load.
        </p>
      </div>
    )
  }

  if (demo.status !== "ready") {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-muted-foreground">
          Demo is not ready for viewing (status: {demo.status}).
        </p>
      </div>
    )
  }

  return (
    <main data-testid="demo-analysis" className="flex flex-col gap-6 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold">Analysis</h1>
        <DemoRouteTabs demoId={String(demo.id)} />
      </div>
      <section className="rounded-md border border-border bg-card p-4">
        <AnalysisOverallGauge />
      </section>
      <section className="rounded-md border border-border bg-card p-4">
        <CategoryCard category="trade" />
      </section>
      <section className="rounded-md border border-border bg-card p-4">
        <CategoryCard category="aim" />
      </section>
      <section className="rounded-md border border-border bg-card p-4">
        <CategoryCard category="movement" />
      </section>
      <section className="rounded-md border border-border bg-card p-4">
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          Round trade %
        </h2>
        <RoundTradeBars />
      </section>
    </main>
  )
}
