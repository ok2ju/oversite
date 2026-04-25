import { Folder, Plus } from "lucide-react"
import { useImportDemo, useImportFolder } from "@/hooks/use-demos"

export function DemosHeaderActions() {
  const { importDemo, isImporting } = useImportDemo()
  const { importFolder, isImporting: isFolderImporting } = useImportFolder()

  return (
    <>
      <button
        type="button"
        className="btn-sm ghost"
        onClick={() => importFolder()}
        disabled={isFolderImporting}
      >
        <Folder className="h-3 w-3" />
        {isFolderImporting ? "Scanning…" : "Choose folder"}
      </button>
      <button
        type="button"
        className="btn-sm primary"
        onClick={() => importDemo()}
        disabled={isImporting}
      >
        <Plus className="h-3 w-3" />
        {isImporting ? "Importing…" : "Import demos"}
      </button>
    </>
  )
}
