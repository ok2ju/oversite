import { useNavigate } from "react-router-dom"
import { ArrowLeft, ExternalLink, Play } from "lucide-react"
import { Button } from "@/components/ui/button"
import type { DemoStatus } from "@/types/demo"

interface MatchToolbarProps {
  matchId: string
  demoId?: number | null
  demoStatus?: DemoStatus | null
}

export function MatchToolbar({
  matchId,
  demoId,
  demoStatus,
}: MatchToolbarProps) {
  const navigate = useNavigate()
  const canPlay = demoId != null && demoStatus === "ready"

  return (
    <div className="flex items-center gap-3">
      <Button
        variant="ghost"
        size="sm"
        className="gap-1.5"
        onClick={() => navigate(-1)}
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Back
      </Button>

      <span
        className="font-mono rounded-full px-2.5 py-1 text-[11.5px]"
        style={{
          background: "var(--bg-sunken)",
          color: "var(--text-muted)",
        }}
      >
        {matchId}
      </span>

      <div className="ml-auto flex items-center gap-2">
        <Button variant="ghost" size="sm" className="gap-1.5" disabled>
          <ExternalLink className="h-3.5 w-3.5" />
          Faceit page
        </Button>
        <Button
          size="sm"
          className="gap-1.5"
          onClick={() => canPlay && navigate(`/demos/${demoId}`)}
          disabled={!canPlay}
        >
          <Play className="h-3.5 w-3.5" />
          Play demo
        </Button>
      </div>
    </div>
  )
}
