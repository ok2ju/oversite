import { render, type RenderOptions } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ThemeProvider } from "next-themes"
import { AuthProvider } from "@/components/providers/auth-provider"

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

interface ProvidersOptions {
  withAuth?: boolean
}

function createAllProviders({ withAuth = false }: ProvidersOptions = {}) {
  return function AllProviders({ children }: { children: React.ReactNode }) {
    const queryClient = createTestQueryClient()
    const content = withAuth ? <AuthProvider>{children}</AuthProvider> : children
    return (
      <QueryClientProvider client={queryClient}>
        <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false}>
          {content}
        </ThemeProvider>
      </QueryClientProvider>
    )
  }
}

interface RenderWithProvidersOptions extends Omit<RenderOptions, "wrapper"> {
  withAuth?: boolean
}

export function renderWithProviders(ui: React.ReactElement, options?: RenderWithProvidersOptions) {
  const { withAuth, ...renderOptions } = options ?? {}
  return render(ui, { wrapper: createAllProviders({ withAuth }), ...renderOptions })
}

export { render } from "@testing-library/react"
export { default as userEvent } from "@testing-library/user-event"
