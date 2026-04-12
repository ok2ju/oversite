import { describe, it, expect, vi } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import { mockAppBindings } from "@/test/mocks/bindings"
import { UploadDialog } from "@/components/demos/upload-dialog"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)

describe("UploadDialog", () => {
  it("opens dialog when trigger is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<UploadDialog />)

    await user.click(screen.getByRole("button", { name: /import demo/i }))
    expect(screen.getByRole("dialog")).toBeInTheDocument()
  })

  it("calls ImportDemoFile binding on import click", async () => {
    const user = userEvent.setup()
    renderWithProviders(<UploadDialog />)

    await user.click(screen.getByRole("button", { name: /import demo/i }))
    await user.click(screen.getByRole("button", { name: /select & import/i }))
    expect(mockAppBindings.ImportDemoFile).toHaveBeenCalled()
  })

  it("shows error state on import failure", async () => {
    mockAppBindings.ImportDemoFile.mockRejectedValueOnce(
      new Error("No file selected"),
    )
    const user = userEvent.setup()
    renderWithProviders(<UploadDialog />)

    await user.click(screen.getByRole("button", { name: /import demo/i }))
    await user.click(screen.getByRole("button", { name: /select & import/i }))

    await waitFor(() => {
      expect(screen.getByText("No file selected")).toBeInTheDocument()
    })
  })

  it("shows importing state while import is pending", async () => {
    mockAppBindings.ImportDemoFile.mockImplementationOnce(
      () => new Promise(() => {}),
    )
    const user = userEvent.setup()
    renderWithProviders(<UploadDialog />)

    await user.click(screen.getByRole("button", { name: /import demo/i }))
    await user.click(screen.getByRole("button", { name: /select & import/i }))

    await waitFor(() => {
      expect(screen.getByText("Importing...")).toBeInTheDocument()
    })
  })
})
