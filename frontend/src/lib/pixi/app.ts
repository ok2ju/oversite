import { Application, Container } from "pixi.js"

export interface ViewerAppOptions {
  container: HTMLElement
  background?: number
}

export class ViewerApp {
  private app: Application
  private layers = new Map<
    string,
    { container: Container; parent: Container }
  >()
  private _initialized = false

  constructor(app: Application) {
    this.app = app
  }

  async init(options: ViewerAppOptions): Promise<void> {
    await this.app.init({
      background: options.background ?? 0x1a1a2e,
      resizeTo: options.container,
      antialias: true,
      resolution: window.devicePixelRatio,
      autoDensity: true,
    })

    options.container.appendChild(this.app.canvas)
    this._initialized = true
  }

  get stage() {
    return this.app.stage
  }

  get canvas() {
    return this.app.canvas
  }

  get ticker() {
    return this.app.ticker
  }

  get initialized() {
    return this._initialized
  }

  addLayer(name: string, parent?: Container): Container {
    if (this.layers.has(name)) {
      throw new Error(`Layer "${name}" already exists`)
    }

    const container = new Container()
    container.label = name
    const target = parent ?? this.app.stage
    target.addChild(container)
    this.layers.set(name, { container, parent: target })
    return container
  }

  getLayer(name: string): Container | undefined {
    return this.layers.get(name)?.container
  }

  removeLayer(name: string): void {
    const entry = this.layers.get(name)
    if (!entry) return

    entry.parent.removeChild(entry.container)
    this.layers.delete(name)
  }

  destroy(): void {
    this.app.destroy(
      { removeView: true },
      { children: true, texture: true, textureSource: true },
    )
    this.layers.clear()
    this._initialized = false
  }
}

export async function createViewerApp(
  options: ViewerAppOptions,
): Promise<ViewerApp> {
  const app = new Application()
  const viewer = new ViewerApp(app)
  await viewer.init(options)
  return viewer
}
