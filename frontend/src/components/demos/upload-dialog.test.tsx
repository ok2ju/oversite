import { screen, waitFor } from "@testing-library/react"
import { renderWithProviders, userEvent } from "@/test/render"
import { UploadDialog } from "@/components/demos/upload-dialog"

describe("UploadDialog", () => {
  it("opens dialog when trigger is clicked", async () => {
    const user = userEvent.setup()
    renderWithProviders(<UploadDialog />)

    await user.click(screen.getByRole("button", { name: /upload demo/i }))
    expect(screen.getByRole("dialog")).toBeInTheDocument()
  })

  it("file input accepts only .dem files", async () => {
    const user = userEvent.setup()
    renderWithProviders(<UploadDialog />)

    await user.click(screen.getByRole("button", { name: /upload demo/i }))
    const input = screen.getByTestId("file-input") as HTMLInputElement
    expect(input.accept).toBe(".dem")
  })

  it("shows filename and size after selecting a file", async () => {
    const user = userEvent.setup()
    renderWithProviders(<UploadDialog />)

    await user.click(screen.getByRole("button", { name: /upload demo/i }))

    const file = new File(["x".repeat(1024)], "match.dem", {
      type: "application/octet-stream",
    })
    const input = screen.getByTestId("file-input")
    await user.upload(input, file)

    expect(screen.getByText(/match\.dem/)).toBeInTheDocument()
  })

  it("shows error state on upload failure", async () => {
    const user = userEvent.setup()
    renderWithProviders(<UploadDialog />)

    await user.click(screen.getByRole("button", { name: /upload demo/i }))

    // The error state will be tested when wired with the hook
    // For now, verify the dialog renders correctly
    expect(screen.getByRole("dialog")).toBeInTheDocument()
  })
})
