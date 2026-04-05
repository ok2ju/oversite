"use client"

import { useDemos, useDeleteDemo } from "@/hooks/use-demos"
import { DemoList } from "@/components/demos/demo-list"
import { UploadDialog } from "@/components/demos/upload-dialog"

export default function DemosPage() {
  const { data, isLoading } = useDemos()
  const deleteDemo = useDeleteDemo()

  const demos = data?.data ?? []

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Demos</h1>
          <p className="mt-1 text-muted-foreground">
            Upload and manage your CS2 demo files
          </p>
        </div>
        <UploadDialog />
      </div>

      {!isLoading && demos.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed p-12 text-center">
          <p className="text-lg font-medium">No demos yet</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Upload a .dem file to get started
          </p>
        </div>
      ) : (
        <DemoList
          demos={demos}
          isLoading={isLoading}
          onDelete={(id) => deleteDemo.mutate(id)}
        />
      )}
    </div>
  )
}
