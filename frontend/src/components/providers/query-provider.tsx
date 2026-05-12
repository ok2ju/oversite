import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { useState } from "react"

const __viteEnv = (import.meta as unknown as { env?: { DEV?: boolean } }).env

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(() => {
    const client = new QueryClient({
      defaultOptions: {
        queries: {
          staleTime: 60 * 1000,
          gcTime: 60 * 1000,
          refetchOnWindowFocus: false,
          retry: 1,
        },
      },
    })
    // DEV-only: expose for Playwright e2e specs to read cached
    // contact-moments data without re-fetching through the binding.
    if (__viteEnv?.DEV && typeof window !== "undefined") {
      ;(window as unknown as { __queryClient: QueryClient }).__queryClient =
        client
    }
    return client
  })

  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}
