import { useState } from "react"
import { useDemos, useDeleteDemo, useImportDemoByPath } from "@/hooks/use-demos"
import { useParseProgress } from "@/hooks/use-parse-progress"
import { WatchBanner } from "@/components/demos/watch-banner"
import { ImportQueue } from "@/components/demos/import-queue"
import {
  DemosToolbar,
  type DemosFilter,
} from "@/components/demos/demos-toolbar"
import { LibraryTable } from "@/components/demos/library-table"
import { DropZone } from "@/components/demos/drop-zone"
import { Skeleton } from "@/components/ui/skeleton"

export default function DemosPage() {
  const { data, isLoading } = useDemos()
  const deleteDemo = useDeleteDemo()
  const { importByPath } = useImportDemoByPath()

  useParseProgress()

  const [search, setSearch] = useState("")
  const [filter, setFilter] = useState<DemosFilter>("all")

  const demos = data?.data ?? []
  const total = data?.meta.total ?? 0

  function handleFilesDropped(filePaths: string[]) {
    for (const path of filePaths) {
      importByPath(path)
    }
  }

  return (
    <DropZone onFilesDropped={handleFilesDropped}>
      <div className="flex flex-col gap-[18px]">
        <WatchBanner queuedCount={total} />
        <ImportQueue />
        <DemosToolbar
          search={search}
          onSearchChange={setSearch}
          filter={filter}
          onFilterChange={setFilter}
        />

        {isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : (
          <LibraryTable
            demos={demos}
            search={search}
            filter={filter}
            onDelete={(id) => deleteDemo.mutate(id)}
          />
        )}
      </div>
    </DropZone>
  )
}
