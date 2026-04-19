import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { StatusPill, statusKey } from "@/components/demos/status-pill"

describe("statusKey", () => {
  it("maps domain statuses to pill variants", () => {
    expect(statusKey("ready")).toBe("ready")
    expect(statusKey("parsing")).toBe("parsing")
    expect(statusKey("failed")).toBe("error")
    expect(statusKey("imported")).toBe("pending")
  })
})

describe("StatusPill", () => {
  it.each([
    ["ready", "Ready"],
    ["parsing", "Parsing"],
    ["failed", "Error"],
    ["imported", "Pending"],
  ] as const)(
    "renders the %s label with the right variant",
    (status, label) => {
      render(<StatusPill status={status} />)
      expect(screen.getByText(label)).toBeInTheDocument()
    },
  )
})
