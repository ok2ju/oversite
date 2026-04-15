import { useState, useEffect, useCallback } from "react"
import { OnFileDrop, OnFileDropOff } from "@wailsjs/runtime/runtime"

interface DropZoneProps {
  onFilesDropped: (filePaths: string[]) => void
  children: React.ReactNode
}

export function DropZone({ onFilesDropped, children }: DropZoneProps) {
  const [isDragging, setIsDragging] = useState(false)

  const handleDrop = useCallback(
    (_x: number, _y: number, paths: string[]) => {
      setIsDragging(false)
      const demFiles = paths.filter((p) => p.toLowerCase().endsWith(".dem"))
      if (demFiles.length > 0) {
        onFilesDropped(demFiles)
      }
    },
    [onFilesDropped],
  )

  useEffect(() => {
    OnFileDrop(handleDrop, false)
    return () => {
      OnFileDropOff()
    }
  }, [handleDrop])

  return (
    <div
      className={
        isDragging
          ? "rounded-lg border-2 border-dashed border-primary bg-primary/5 transition-colors"
          : ""
      }
    >
      {children}
    </div>
  )
}
