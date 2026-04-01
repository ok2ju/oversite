import { render, type RenderOptions } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ThemeProvider } from "next-themes"

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  })
}

function AllProviders({ children }: { children: React.ReactNode }) {
  const queryClient = createTestQueryClient()
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false}>
        {children}
      </ThemeProvider>
    </QueryClientProvider>
  )
}

export function renderWithProviders(ui: React.ReactElement, options?: Omit<RenderOptions, "wrapper">) {
  return render(ui, { wrapper: AllProviders, ...options })
}

export { render } from "@testing-library/react"
export { default as userEvent } from "@testing-library/user-event"
