import { screen } from "@testing-library/react"
import { http, HttpResponse } from "msw"
import { describe, expect, it } from "vitest"
import { renderWithProviders, userEvent } from "@/test/render"
import { server } from "@/test/msw/server"
import { Button } from "./button"

describe("Button", () => {
  it("renders with text", () => {
    renderWithProviders(<Button>Click me</Button>)
    expect(screen.getByRole("button", { name: "Click me" })).toBeInTheDocument()
  })

  it("handles click events", async () => {
    const user = userEvent.setup()
    let clicked = false
    renderWithProviders(
      <Button
        onClick={() => {
          clicked = true
        }}
      >
        Click
      </Button>,
    )

    await user.click(screen.getByRole("button", { name: "Click" }))
    expect(clicked).toBe(true)
  })

  it("renders disabled state", () => {
    renderWithProviders(<Button disabled>Disabled</Button>)
    expect(screen.getByRole("button", { name: "Disabled" })).toBeDisabled()
  })
})

describe("MSW integration", () => {
  it("intercepts API calls with default handler", async () => {
    const res = await fetch("/api/v1/demos")
    const data = await res.json()

    expect(Array.isArray(data.data)).toBe(true)
    expect(data.meta).toMatchObject({ page: 1, per_page: 20 })
  })

  it("supports per-test handler overrides", async () => {
    server.use(
      http.get("/api/v1/demos", () => {
        return HttpResponse.json(
          { data: null, error: "unavailable" },
          { status: 503 },
        )
      }),
    )

    const res = await fetch("/api/v1/demos")
    expect(res.status).toBe(503)
  })
})
