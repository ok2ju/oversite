import { describe, it, expect, beforeEach } from "vitest"
import { useStratStore } from "./strat"

describe("stratStore", () => {
  beforeEach(() => {
    useStratStore.getState().reset()
  })

  it("has correct initial state", () => {
    const state = useStratStore.getState()
    expect(state.currentTool).toBe("select")
    expect(state.boardId).toBeNull()
    expect(state.mapName).toBeNull()
    expect(state.color).toBe("#ff0000")
    expect(state.strokeWidth).toBe(2)
  })

  it("setTool updates currentTool", () => {
    useStratStore.getState().setTool("draw")
    expect(useStratStore.getState().currentTool).toBe("draw")
  })

  it("setBoard updates boardId and mapName", () => {
    useStratStore.getState().setBoard("board-1", "de_mirage")
    expect(useStratStore.getState().boardId).toBe("board-1")
    expect(useStratStore.getState().mapName).toBe("de_mirage")
  })

  it("setColor updates color", () => {
    useStratStore.getState().setColor("#00ff00")
    expect(useStratStore.getState().color).toBe("#00ff00")
  })

  it("setStrokeWidth updates strokeWidth", () => {
    useStratStore.getState().setStrokeWidth(4)
    expect(useStratStore.getState().strokeWidth).toBe(4)
  })
})
