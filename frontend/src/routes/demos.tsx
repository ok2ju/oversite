import { useDemos, useDeleteDemo, useImportFolder } from "@/hooks/use-demos"
import { DemoList } from "@/components/demos/demo-list"
import { UploadDialog } from "@/components/demos/upload-dialog"
import { Button } from "@/components/ui/button"
import { FolderOpen } from "lucide-react"

export default function DemosPage() {
  const { data, isLoading } = useDemos()
  const deleteDemo = useDeleteDemo()
  const { importFolder, isImporting: isFolderImporting } = useImportFolder()

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
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => importFolder()}
            disabled={isFolderImporting}
          >
            <FolderOpen className="mr-2 h-4 w-4" />
            {isFolderImporting ? "Importing..." : "Import Folder"}
          </Button>
          <UploadDialog />
        </div>
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
