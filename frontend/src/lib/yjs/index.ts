export {
  createStratDoc,
  getBoardSettings,
  getDrawingElements,
  createDrawingElement,
  removeDrawingElement,
  getStrokeData,
  type DrawingElement,
  type DrawingElementType,
  type BoardSettings,
} from "./doc"

export {
  setLocalUser,
  updateCursorPosition,
  clearCursor,
  getRemoteStates,
  onAwarenessChange,
  COLLABORATION_COLORS,
  type AwarenessUserState,
} from "./awareness"

export {
  buildWsUrl,
  createStratProvider,
  type StratProviderOptions,
  type StratProvider,
} from "./provider"
