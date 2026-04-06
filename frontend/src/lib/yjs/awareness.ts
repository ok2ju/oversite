import { Awareness } from "y-protocols/awareness"

export interface AwarenessUserState {
  user: {
    name: string
    color: string
    userId: string
  }
  cursor: { x: number; y: number } | null
}

export const COLLABORATION_COLORS: string[] = [
  "#f94144",
  "#f3722c",
  "#f8961e",
  "#f9c74f",
  "#90be6d",
  "#43aa8b",
  "#577590",
  "#6d6875",
]

export function setLocalUser(
  awareness: Awareness,
  user: AwarenessUserState["user"]
): void {
  awareness.setLocalStateField("user", user)
  awareness.setLocalStateField("cursor", null)
}

export function updateCursorPosition(
  awareness: Awareness,
  x: number,
  y: number
): void {
  awareness.setLocalStateField("cursor", { x, y })
}

export function clearCursor(awareness: Awareness): void {
  awareness.setLocalStateField("cursor", null)
}

export function getRemoteStates(
  awareness: Awareness
): Map<number, AwarenessUserState> {
  const localId = awareness.clientID
  const result = new Map<number, AwarenessUserState>()
  awareness.getStates().forEach((state, clientId) => {
    if (clientId !== localId) {
      result.set(clientId, state as AwarenessUserState)
    }
  })
  return result
}

export function onAwarenessChange(
  awareness: Awareness,
  callback: () => void
): () => void {
  awareness.on("change", callback)
  return () => awareness.off("change", callback)
}
