import { useState } from "react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { useImportDemo } from "@/hooks/use-demos"
import { Upload } from "lucide-react"

export function UploadDialog() {
  const [open, setOpen] = useState(false)
  const { importDemo, isImporting, error, reset } = useImportDemo()

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen)
    if (!nextOpen) {
      reset()
    }
  }

  function handleImport() {
    importDemo(undefined, {
      onSuccess: () => {
        setOpen(false)
        reset()
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button>
          <Upload className="mr-2 h-4 w-4" />
          Import Demo
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Import Demo</DialogTitle>
          <DialogDescription>
            Select a CS2 .dem file from your computer to import and parse.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          {error && <p className="text-sm text-destructive">{error.message}</p>}
          <Button
            onClick={handleImport}
            disabled={isImporting}
            className="w-full"
          >
            {isImporting ? "Importing..." : "Select & Import .dem File"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
