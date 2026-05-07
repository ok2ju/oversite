import { Assets, type Texture } from "pixi.js"
import { getWeaponIconPath } from "@/lib/viewer/weapon-icons"

// Cache loaded textures by source path. SVGs raster-decode once into a GPU
// texture and are reused for every PlayerSprite that holds the same weapon —
// PixiJS Text rasterization on each ammo change was the previous bottleneck.

const textureCache = new Map<string, Texture>()
const inFlight = new Map<string, Promise<Texture>>()

function resolveUrl(path: string): string {
  // PixiJS Assets prepends its baseURL ("/") and turns absolute paths like
  // "/equipment/foo.svg" into "//equipment/foo.svg" (a protocol-relative URL).
  // Resolve against location.href the same way map-layer.ts does.
  return new URL(path, globalThis.location.href).href
}

// Synchronous lookup for the render loop: returns the cached texture if loaded,
// otherwise kicks off a background load and returns null. Subsequent ticks pick
// up the texture once Assets.load resolves.
export function getWeaponTexture(
  weapon: string | null | undefined,
): Texture | null {
  const path = getWeaponIconPath(weapon)
  if (!path) return null

  const cached = textureCache.get(path)
  if (cached) return cached

  if (!inFlight.has(path)) {
    const url = resolveUrl(path)
    const promise = (Assets.load(url) as Promise<Texture>).then((tex) => {
      textureCache.set(path, tex)
      return tex
    })
    inFlight.set(path, promise)
  }
  return null
}

// Test/teardown helper: drop caches between specs so loaded textures don't
// leak across tests and the next getWeaponTexture call re-loads from the
// (possibly re-mocked) Assets module.
export function _resetWeaponTextureCache(): void {
  textureCache.clear()
  inFlight.clear()
}
