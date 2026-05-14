import { useEffect, useState } from "react"
import { useOutletContext } from "react-router-dom"
import { useDemos, useDeleteDemo, useImportDemoByPath } from "@/hooks/use-demos"
import { useParseProgress } from "@/hooks/use-parse-progress"
import { ImportQueue } from "@/components/demos/import-queue"
import {
  DemosToolbar,
  type DemosFilter,
} from "@/components/demos/demos-toolbar"
import { LibraryTable } from "@/components/demos/library-table"
import { DropZone } from "@/components/demos/drop-zone"
import { Skeleton } from "@/components/ui/skeleton"
import { DemosHeaderActions } from "@/components/demos/demos-header-actions"
import { DemosEmptyHero } from "@/components/demos/empty-hero"
import type { HeaderActionsContext } from "@/routes/root"

export default function DemosPage() {
  const { data, isLoading } = useDemos()
  const deleteDemo = useDeleteDemo()
  const { importByPath } = useImportDemoByPath()
  const ctx = useOutletContext<HeaderActionsContext | undefined>()

  useParseProgress()

  const [search, setSearch] = useState("")
  const [filter, setFilter] = useState<DemosFilter>("all")

  useEffect(() => {
    ctx?.setHeaderActions(<DemosHeaderActions />)
    return () => ctx?.setHeaderActions(null)
  }, [ctx])

  const demos = data?.data ?? []

  function handleFilesDropped(filePaths: string[]) {
    for (const path of filePaths) {
      importByPath(path)
    }
  }

  const showEmptyHero = !isLoading && demos.length === 0

  return (
    <DropZone onFilesDropped={handleFilesDropped}>
      <div className="flex flex-col gap-4">
        <ImportQueue />
        {showEmptyHero ? (
          <DemosEmptyHero />
        ) : (
          <>
            <DemosToolbar
              search={search}
              onSearchChange={setSearch}
              filter={filter}
              onFilterChange={setFilter}
              totalCount={demos.length}
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
          </>
        )}
      </div>
    </DropZone>
  )
}
