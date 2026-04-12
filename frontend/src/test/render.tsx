import { render, type RenderOptions } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { MemoryRouter } from "react-router-dom"
import { ThemeProvider } from "@/components/providers/theme-provider"

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
  initialRoute?: string
}

function createAllProviders({ initialRoute = "/" }: ProvidersOptions = {}) {
  return function AllProviders({ children }: { children: React.ReactNode }) {
    const queryClient = createTestQueryClient()
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[initialRoute]}>
          <ThemeProvider defaultTheme="dark">{children}</ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>
    )
  }
}

interface RenderWithProvidersOptions extends Omit<RenderOptions, "wrapper"> {
  initialRoute?: string
}

export function renderWithProviders(
  ui: React.ReactElement,
  options?: RenderWithProvidersOptions,
) {
  const { initialRoute, ...renderOptions } = options ?? {}
  return render(ui, {
    wrapper: createAllProviders({ initialRoute }),
    ...renderOptions,
  })
}

export { render } from "@testing-library/react"
export { default as userEvent } from "@testing-library/user-event"
