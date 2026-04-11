"use client"

import { useRef, useState } from "react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { useUploadDemo } from "@/hooks/use-demos"
import { Upload } from "lucide-react"

function formatFileSize(bytes: number): string {
  if (bytes >= 1_000_000_000) return `${(bytes / 1_000_000_000).toFixed(1)} GB`
  if (bytes >= 1_000_000) return `${(bytes / 1_000_000).toFixed(1)} MB`
  return `${(bytes / 1_000).toFixed(1)} KB`
}

export function UploadDialog() {
  const [open, setOpen] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const { upload, progress, isUploading, error, reset } =
    useUploadDemo()

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen)
    if (!nextOpen) {
      setSelectedFile(null)
      reset()
    }
  }

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0] ?? null
    setSelectedFile(file)
  }

  function handleUpload() {
    if (!selectedFile) return
    upload(selectedFile, {
      onSuccess: () => {
        setOpen(false)
        setSelectedFile(null)
        reset()
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button>
          <Upload className="mr-2 h-4 w-4" />
          Upload Demo
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Upload Demo</DialogTitle>
          <DialogDescription>
            Select a CS2 .dem file to upload and parse.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <input
            ref={inputRef}
            type="file"
            accept=".dem"
            data-testid="file-input"
            className="block w-full text-sm file:mr-4 file:rounded-md file:border-0 file:bg-primary file:px-4 file:py-2 file:text-sm file:font-medium file:text-primary-foreground hover:file:bg-primary/90"
            onChange={handleFileChange}
            disabled={isUploading}
          />
          {selectedFile && (
            <p className="text-sm text-muted-foreground">
              {selectedFile.name} ({formatFileSize(selectedFile.size)})
            </p>
          )}
          {isUploading && <Progress value={progress} />}
          {error && (
            <p className="text-sm text-destructive">
              {error.message}
            </p>
          )}
          <Button
            onClick={handleUpload}
            disabled={!selectedFile || isUploading}
            className="w-full"
          >
            {isUploading ? `Uploading... ${progress}%` : "Upload"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
