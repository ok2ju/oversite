import { vi, describe, it, expect, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/render"
import { mockRuntime, resetRuntimeMocks } from "@/test/mocks/bindings"
import { DropZone } from "./drop-zone"

vi.mock("@wailsjs/runtime/runtime", () => mockRuntime)

describe("DropZone", () => {
  beforeEach(() => {
    resetRuntimeMocks()
  })

  it("renders children", () => {
    renderWithProviders(
      <DropZone onFilesDropped={vi.fn()}>
        <p>Drop demos here</p>
      </DropZone>,
    )

    expect(screen.getByText("Drop demos here")).toBeInTheDocument()
  })

  it("registers OnFileDrop on mount", () => {
    renderWithProviders(
      <DropZone onFilesDropped={vi.fn()}>
        <p>Content</p>
      </DropZone>,
    )

    expect(mockRuntime.OnFileDrop).toHaveBeenCalledTimes(1)
  })

  it("filters for .dem files and calls onFilesDropped", () => {
    const onFilesDropped = vi.fn()

    renderWithProviders(
      <DropZone onFilesDropped={onFilesDropped}>
        <p>Content</p>
      </DropZone>,
    )

    // Grab the callback passed to OnFileDrop
    const dropHandler = mockRuntime.OnFileDrop.mock.calls[0][0] as (
      x: number,
      y: number,
      paths: string[],
    ) => void

    dropHandler(0, 0, [
      "/demos/match1.dem",
      "/demos/notes.txt",
      "/demos/match2.DEM",
      "/demos/match3.dem.zst",
    ])

    expect(onFilesDropped).toHaveBeenCalledWith([
      "/demos/match1.dem",
      "/demos/match2.DEM",
      "/demos/match3.dem.zst",
    ])
  })

  it("does not call onFilesDropped when no .dem files are dropped", () => {
    const onFilesDropped = vi.fn()

    renderWithProviders(
      <DropZone onFilesDropped={onFilesDropped}>
        <p>Content</p>
      </DropZone>,
    )

    const dropHandler = mockRuntime.OnFileDrop.mock.calls[0][0] as (
      x: number,
      y: number,
      paths: string[],
    ) => void

    dropHandler(0, 0, ["/demos/readme.txt", "/demos/image.png"])

    expect(onFilesDropped).not.toHaveBeenCalled()
  })
})
