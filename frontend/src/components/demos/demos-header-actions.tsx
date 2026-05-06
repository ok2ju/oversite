import { Plus } from "lucide-react"
import { useImportDemo } from "@/hooks/use-demos"

export function DemosHeaderActions() {
  const { importDemo, isImporting } = useImportDemo()

  return (
    <button
      type="button"
      className="btn-sm primary"
      onClick={() => importDemo()}
      disabled={isImporting}
    >
      <Plus className="h-3 w-3" />
      {isImporting ? "Importing…" : "Import demos"}
    </button>
  )
}
