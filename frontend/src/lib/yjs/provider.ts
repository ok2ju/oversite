import { WebsocketProvider } from "y-websocket"
import * as Y from "yjs"
import { Awareness } from "y-protocols/awareness"

export interface StratProviderOptions {
  stratId: string
  doc: Y.Doc
  awareness?: Awareness
  connect?: boolean
}

export interface StratProvider {
  provider: WebsocketProvider
  doc: Y.Doc
  awareness: Awareness
  destroy: () => void
}

export function buildWsUrl(host: string, protocol: string): string {
  const wsProtocol = protocol === "https:" ? "wss:" : "ws:"
  return `${wsProtocol}//${host}/ws/strat`
}

export function createStratProvider(
  options: StratProviderOptions,
): StratProvider {
  const { stratId, doc, awareness, connect = true } = options
  const url = buildWsUrl(window.location.host, window.location.protocol)

  const provider = new WebsocketProvider(url, stratId, doc, {
    connect,
    awareness: awareness ?? new Awareness(doc),
    maxBackoffTime: 10000,
  })

  return {
    provider,
    doc,
    awareness: provider.awareness,
    destroy: () => {
      provider.awareness.destroy()
      provider.destroy()
    },
  }
}
