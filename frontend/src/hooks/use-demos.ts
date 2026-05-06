import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  ListDemos,
  ImportDemoFile,
  ImportDemoByPath,
  DeleteDemo,
} from "@wailsjs/go/main/App"
import type { DemoListResponse } from "@/types/demo"

export function useDemos(page = 1, perPage = 20) {
  return useQuery({
    queryKey: ["demos", page, perPage],
    queryFn: () => ListDemos(page, perPage) as Promise<DemoListResponse>,
    refetchInterval: (query) => {
      const data = query.state.data
      if (!data) return false
      const hasActive = data.data.some(
        (d) => d.status === "imported" || d.status === "parsing",
      )
      return hasActive ? 5000 : false
    },
  })
}

export function useImportDemo() {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => ImportDemoFile(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["demos"] })
    },
  })

  return {
    importDemo: mutation.mutate,
    isImporting: mutation.isPending,
    error: mutation.error,
    isSuccess: mutation.isSuccess,
    reset: mutation.reset,
  }
}

export function useImportDemoByPath() {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: (filePath: string) => ImportDemoByPath(filePath),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["demos"] })
    },
  })

  return {
    importByPath: mutation.mutateAsync,
    isImporting: mutation.isPending,
  }
}

export function useDeleteDemo() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => DeleteDemo(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["demos"] })
    },
  })
}
