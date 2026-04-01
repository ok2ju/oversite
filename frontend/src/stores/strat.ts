import { create } from "zustand"

type StratTool = "select" | "draw" | "line" | "arrow" | "text" | "icon" | "eraser"

interface StratState {
  currentTool: StratTool
  boardId: string | null
  mapName: string | null
  color: string
  strokeWidth: number
  setTool: (tool: StratTool) => void
  setBoard: (id: string | null, mapName: string | null) => void
  setColor: (color: string) => void
  setStrokeWidth: (width: number) => void
  reset: () => void
}

const initialState = {
  currentTool: "select" as StratTool,
  boardId: null as string | null,
  mapName: null as string | null,
  color: "#ff0000",
  strokeWidth: 2,
}

export const useStratStore = create<StratState>((set) => ({
  ...initialState,
  setTool: (tool) => set({ currentTool: tool }),
  setBoard: (id, mapName) => set({ boardId: id, mapName }),
  setColor: (color) => set({ color }),
  setStrokeWidth: (width) => set({ strokeWidth: width }),
  reset: () => set(initialState),
}))
