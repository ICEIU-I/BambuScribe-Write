import type { ImageTextRegion, Point } from '../types'

export const isTextBox = (region: ImageTextRegion) =>
  region.source === 'manual' && region.mode === 'handwriting' && !region.sourceBounds

// A covering textbox uses the same rectangle clipping as an eraser, but only
// affects layers beneath it; its own ink is always drawn afterwards.
export function imageMasks(regions: ImageTextRegion[]): ImageTextRegion[] {
  return regions.flatMap((region) =>
    region.mode === 'erase'
      ? [region]
      : isTextBox(region)
        ? [{ ...region, mode: 'erase' as const }]
        : [],
  )
}

export function transformTextBox(
  region: ImageTextRegion,
  delta: Point,
  resize: boolean,
): ImageTextRegion {
  if (resize)
    return {
      ...region,
      width: Math.max(0.005, Math.min(1 - region.x, region.width + delta.x)),
      height: Math.max(0.005, Math.min(1 - region.y, region.height + delta.y)),
    }
  return {
    ...region,
    x: Math.max(0, Math.min(1 - region.width, region.x + delta.x)),
    y: Math.max(0, Math.min(1 - region.height, region.y + delta.y)),
  }
}
