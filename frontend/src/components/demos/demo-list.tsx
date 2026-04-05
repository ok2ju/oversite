"use client"

import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { DemoCard } from "@/components/demos/demo-card"
import type { Demo } from "@/types/demo"

interface DemoListProps {
  demos: Demo[]
  isLoading: boolean
  onDelete: (id: string) => void
}

export function DemoList({ demos, isLoading, onDelete }: DemoListProps) {
  if (isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Card key={i} data-testid="demo-skeleton">
            <CardHeader>
              <Skeleton className="h-5 w-32" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-4 w-24" />
              <Skeleton className="mt-2 h-4 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {demos.map((demo) => (
        <DemoCard key={demo.id} demo={demo} onDelete={onDelete} />
      ))}
    </div>
  )
}
