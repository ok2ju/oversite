import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
} from "recharts"
import type { EloHistoryPoint } from "@/types/faceit"

interface EloChartProps {
  data: EloHistoryPoint[] | undefined
  isLoading: boolean
  days: number
  onDaysChange: (days: number) => void
}

const timeRanges = [
  { label: "30d", value: 30 },
  { label: "90d", value: 90 },
  { label: "180d", value: 180 },
  { label: "All", value: 0 },
]

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
  })
}

function CustomTooltip({
  active,
  payload,
}: {
  active?: boolean
  payload?: Array<{ payload: EloHistoryPoint }>
}) {
  if (!active || !payload?.length) return null
  const point = payload[0].payload
  return (
    <div className="rounded-md border bg-popover p-2 text-sm shadow-md">
      <p className="font-medium">ELO: {point.elo ?? "-"}</p>
      <p className="text-muted-foreground">{point.map_name}</p>
      <p className="text-muted-foreground">
        {new Date(point.played_at).toLocaleDateString("en-US", {
          month: "short",
          day: "numeric",
          year: "numeric",
        })}
      </p>
    </div>
  )
}

export function EloChart({
  data,
  isLoading,
  days,
  onDaysChange,
}: EloChartProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>ELO History</CardTitle>
          <div className="flex gap-1" role="group" aria-label="Time range">
            {timeRanges.map((range) => (
              <Button
                key={range.value}
                size="sm"
                variant={days === range.value ? "default" : "outline"}
                onClick={() => onDaysChange(range.value)}
              >
                {range.label}
              </Button>
            ))}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton
            className="h-[300px] w-full"
            data-testid="elo-chart-skeleton"
          />
        ) : !data || data.length === 0 ? (
          <div className="flex h-[300px] items-center justify-center text-muted-foreground">
            No ELO history available
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={data}>
              <XAxis
                dataKey="played_at"
                tickFormatter={formatDate}
                stroke="hsl(var(--muted-foreground))"
                fontSize={12}
              />
              <YAxis
                domain={["dataMin - 50", "dataMax + 50"]}
                stroke="hsl(var(--muted-foreground))"
                fontSize={12}
              />
              <Tooltip content={<CustomTooltip />} />
              <Line
                type="monotone"
                dataKey="elo"
                stroke="hsl(var(--primary))"
                strokeWidth={2}
                dot={{ r: 3 }}
                activeDot={{ r: 5 }}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  )
}
