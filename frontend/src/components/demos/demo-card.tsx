import { useNavigate } from "react-router-dom"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Trash2 } from "lucide-react"
import type { Demo } from "@/types/demo"

interface DemoCardProps {
  demo: Demo
  onDelete: (id: number) => void
}

const statusVariant: Record<
  Demo["status"],
  "default" | "secondary" | "outline" | "destructive"
> = {
  ready: "default",
  parsing: "secondary",
  imported: "outline",
  failed: "destructive",
}

export function formatFileSize(bytes: number): string {
  if (bytes >= 1_000_000_000) return `${(bytes / 1_000_000_000).toFixed(1)} GB`
  if (bytes >= 1_000_000) return `${(bytes / 1_000_000).toFixed(1)} MB`
  return `${(bytes / 1_000).toFixed(1)} KB`
}

export function formatDuration(secs: number): string {
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return `${m}:${s.toString().padStart(2, "0")}`
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  })
}

export function DemoCard({ demo, onDelete }: DemoCardProps) {
  const navigate = useNavigate()

  return (
    <Card
      className={
        demo.status === "ready"
          ? "cursor-pointer transition-colors hover:border-primary"
          : ""
      }
      onClick={() => {
        if (demo.status === "ready") {
          navigate(`/demos/${demo.id}`)
        }
      }}
    >
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-base font-medium">
          {demo.map_name || "Unknown Map"}
        </CardTitle>
        <Badge
          variant={statusVariant[demo.status]}
          className={demo.status === "parsing" ? "animate-pulse" : undefined}
        >
          {demo.status}
        </Badge>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <div className="space-y-1">
            <p>{formatFileSize(demo.file_size)}</p>
            {demo.duration_secs > 0 && (
              <p>{formatDuration(demo.duration_secs)}</p>
            )}
            {demo.match_date ? (
              <p>{formatDate(demo.match_date)}</p>
            ) : (
              <p>{formatDate(demo.created_at)}</p>
            )}
          </div>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                aria-label="Delete"
                onClick={(e) => e.stopPropagation()}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent onClick={(e) => e.stopPropagation()}>
              <AlertDialogHeader>
                <AlertDialogTitle>Are you sure?</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently delete this demo and its associated
                  data.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  aria-label="Confirm"
                  onClick={() => onDelete(demo.id)}
                >
                  Confirm
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </CardContent>
    </Card>
  )
}
