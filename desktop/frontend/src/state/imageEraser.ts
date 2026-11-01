import type { ImageTextRegion, Point } from '../types'

export const eraserRadius = (region: ImageTextRegion) => region.eraseRadius ?? 0.018

// The unit square is only for storage. Draw the brush in paper millimetres,
// then map it back to the image, so a wide/tall image cannot stretch the brush.
export function eraserPath(region: ImageTextRegion, width: number, height: number): string {
  const points = region.erasePath ?? []
  return points.map((p, i) => `${i ? 'L' : 'M'}${p.x * width} ${p.y * height}`).join(' ')
}

export function freeEraseRegion(points: Point[], radius: number): ImageTextRegion {
  const xs = points.map((p) => p.x),
    ys = points.map((p) => p.y)
  const x = Math.min(Math.min(...xs), 1 - 1e-6)
  const y = Math.min(Math.min(...ys), 1 - 1e-6)
  return {
    id: 'draft',
    text: '',
    source: 'manual',
    mode: 'erase',
    x,
    y,
    width: Math.max(1e-6, Math.max(...xs) - x),
    height: Math.max(1e-6, Math.max(...ys) - y),
    erasePath: points,
    eraseRadius: radius,
  }
}
