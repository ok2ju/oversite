import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { renderWithProviders } from "@/test/render"
import { mockAppBindings, mockRuntime } from "@/test/mocks/bindings"
import SettingsPage from "@/routes/settings"

vi.mock("@wailsjs/go/main/App", () => mockAppBindings)
vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

describe("SettingsPage", () => {
  it("renders heading", () => {
    renderWithProviders(<SettingsPage />, { initialRoute: "/settings" })
    expect(
      screen.getByRole("heading", { name: "Settings" }),
    ).toBeInTheDocument()
  })

  it("renders description text", () => {
    renderWithProviders(<SettingsPage />)
    expect(
      screen.getByText("Manage your account and preferences"),
    ).toBeInTheDocument()
  })

  it("displays the logs directory path from the backend", async () => {
    mockAppBindings.LogsDir.mockResolvedValueOnce(
      "C:\\Users\\test\\AppData\\Roaming\\oversite\\logs",
    )
    renderWithProviders(<SettingsPage />)
    await waitFor(() =>
      expect(screen.getByTestId("settings-logs-path")).toHaveTextContent(
        "C:\\Users\\test\\AppData\\Roaming\\oversite\\logs",
      ),
    )
  })

  it("opens the logs folder when the button is clicked", async () => {
    const user = userEvent.setup()
    mockAppBindings.LogsDir.mockResolvedValueOnce("/tmp/oversite/logs")
    renderWithProviders(<SettingsPage />)
    await waitFor(() =>
      expect(screen.getByTestId("settings-logs-path")).toHaveTextContent(
        "/tmp/oversite/logs",
      ),
    )
    await user.click(screen.getByRole("button", { name: /open logs folder/i }))
    expect(mockAppBindings.OpenLogsFolder).toHaveBeenCalledTimes(1)
  })

  it("copies the logs path to the clipboard", async () => {
    const user = userEvent.setup()
    mockAppBindings.LogsDir.mockResolvedValueOnce("/tmp/oversite/logs")
    renderWithProviders(<SettingsPage />)
    await waitFor(() =>
      expect(screen.getByTestId("settings-logs-path")).toHaveTextContent(
        "/tmp/oversite/logs",
      ),
    )
    await user.click(screen.getByRole("button", { name: /copy path/i }))
    expect(mockRuntime.ClipboardSetText).toHaveBeenCalledWith(
      "/tmp/oversite/logs",
    )
  })
})
