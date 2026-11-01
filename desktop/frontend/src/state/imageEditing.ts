import type { ImageAsset, ImagePlacement, ImageTextRegion, RegionBounds } from '../types'
export const regionBounds = (r: RegionBounds): RegionBounds => ({
  x: r.x,
  y: r.y,
  width: r.width,
  height: r.height,
})
export function displayRegions(image: ImageAsset, preview?: ImagePlacement): ImageTextRegion[] {
  let regions = image.textRegions ?? preview?.textRegions ?? []
  if (image.textRegions && preview?.textRegions?.some((r) => r.ocrVersion)) {
    // The engine may have regenerated legacy OCR. Use that geometry while
    // keeping the current manual edits, including changes awaiting a preview.
    const manual = new Map(
      image.textRegions.filter((r) => r.source === 'manual').map((r) => [r.id, r]),
    )
    regions = preview.textRegions.flatMap((r) => {
      if (r.source !== 'manual') return [r]
      const current = manual.get(r.id)
      manual.delete(r.id)
      return current ? [current] : []
    })
    regions.push(...manual.values())
  }
  return regions.map(({ strokes: _strokes, sourceStrokes: _sourceStrokes, ...r }) => ({
    ...r,
    sourceBounds: r.sourceBounds
      ? { ...r.sourceBounds }
      : r.source === 'manual'
        ? undefined
        : regionBounds(r),
  }))
}
// Apply the inverse image rotation to pointer deltas. Zoom and rotation must
// not change how far an image moves in its parent paper.
export function imageDelta(
  dx: number,
  dy: number,
  rotation: number,
  width: number,
  height: number,
) {
  const angle = (rotation * Math.PI) / 180
  return {
    x: (dx * Math.cos(angle) + dy * Math.sin(angle)) / width,
    y: (-dx * Math.sin(angle) + dy * Math.cos(angle)) / height,
  }
}
