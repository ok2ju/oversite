"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import type { DemoListResponse } from "@/types/demo"
import { useCallback, useRef, useState } from "react"

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, { credentials: "include", ...init })
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`)
  }
  return res.json()
}

export function useDemos(page = 1, perPage = 20) {
  return useQuery({
    queryKey: ["demos", page, perPage],
    queryFn: () =>
      fetchJSON<DemoListResponse>(
        `/api/v1/demos?page=${page}&per_page=${perPage}`,
      ),
    select: (res) => res,
    refetchInterval: (query) => {
      const data = query.state.data
      if (!data) return false
      const hasActive = data.data.some(
        (d) => d.status === "uploaded" || d.status === "parsing",
      )
      return hasActive ? 5000 : false
    },
  })
}

export function useUploadDemo() {
  const queryClient = useQueryClient()
  const [progress, setProgress] = useState(0)
  const xhrRef = useRef<XMLHttpRequest | null>(null)

  const mutation = useMutation({
    mutationFn: (file: File) => {
      return new Promise<void>((resolve, reject) => {
        const xhr = new XMLHttpRequest()
        xhrRef.current = xhr
        const formData = new FormData()
        formData.append("file", file)

        xhr.upload.addEventListener("progress", (e) => {
          if (e.lengthComputable) {
            setProgress(Math.round((e.loaded / e.total) * 100))
          }
        })

        xhr.addEventListener("load", () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            resolve()
          } else {
            reject(new Error(`Upload failed: ${xhr.status}`))
          }
        })

        xhr.addEventListener("error", () => reject(new Error("Upload failed")))
        xhr.addEventListener("abort", () => reject(new Error("Upload aborted")))

        xhr.open("POST", "/api/v1/demos")
        xhr.withCredentials = true
        xhr.send(formData)
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["demos"] })
    },
    onSettled: () => {
      xhrRef.current = null
    },
  })

  const reset = useCallback(() => {
    setProgress(0)
    mutation.reset()
  }, [mutation])

  return {
    upload: mutation.mutate,
    progress,
    isUploading: mutation.isPending,
    error: mutation.error,
    isSuccess: mutation.isSuccess,
    reset,
  }
}

export function useDeleteDemo() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) =>
      fetch(`/api/v1/demos/${id}`, {
        method: "DELETE",
        credentials: "include",
      }).then((res) => {
        if (!res.ok) throw new Error(`Delete failed: ${res.status}`)
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["demos"] })
    },
  })
}
