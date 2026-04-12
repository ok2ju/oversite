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
    const res = await fetch("/api/v1/auth/me")
    const data = await res.json()

    expect(data.nickname).toBe("TestPlayer")
    expect(data.faceit_id).toBe("test-faceit-id")
  })

  it("supports per-test handler overrides", async () => {
    server.use(
      http.get("/api/v1/auth/me", () => {
        return HttpResponse.json(
          { data: null, error: "unauthorized" },
          { status: 401 },
        )
      }),
    )

    const res = await fetch("/api/v1/auth/me")
    expect(res.status).toBe(401)
  })
})
