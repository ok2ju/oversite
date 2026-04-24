# Coordinate Calibration

**Related:** [[pixijs-viewer]] · [architecture/crosscutting](../architecture/crosscutting.md) §9.5

## The transform

CS2 world coordinates (from demo tick data) and radar image pixel coordinates are two different spaces. We convert per-map using three constants: `originX`, `originY`, `scale`.

```
pixelX = (worldX - originX) / scale
pixelY = (originY - worldY) / scale     // Y is flipped: screen Y grows downward, world Y grows upward
```

Note the **Y flip**: it trips up most bugs in this area. World Y increases going north; image Y increases going south.

## Where calibration lives

`frontend/src/lib/maps/calibration.ts`:

```typescript
export const MAP_CALIBRATION = {
  de_dust2:  { originX: -2476, originY: 3239, scale: 4.4 },
  de_mirage: { originX: -3230, originY: 1713, scale: 5.0 },
  // ...
};
```

One entry per Active Duty map. Numbers come from matching known in-world landmarks (e.g., bombsite center) to pixel coordinates on the radar image.

## When calibration is wrong

Symptoms: players draw off-map, or all players cluster on one side, or movement looks mirrored.

Debugging:
1. Pick a well-known point (e.g. A-site bomb plant). Check `worldX, worldY` from a bomb_plant event.
2. Check the pixel position it renders at.
3. Apply the transform by hand with the constants — if it produces the wrong pixel, the constants are off.
4. Adjust `originX/Y` to shift; adjust `scale` to stretch.

## Radar images

Stored in `frontend/public/maps/`. One PNG per map, 1024×1024. Fetched by PixiJS MapLayer based on `demo.map_name`.
