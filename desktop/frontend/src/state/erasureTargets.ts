import type { Page, PageErasure, Point } from '../types'
import { captureInk } from './inkCuts'

const cross = (a: Point, b: Point, c: Point) =>
  (b.x - a.x) * (c.y - a.y) - (b.y - a.y) * (c.x - a.x)
function pointDistance(p: Point, a: Point, b: Point) {
  const dx = b.x - a.x,
    dy = b.y - a.y
  const t = Math.max(
    0,
    Math.min(1, ((p.x - a.x) * dx + (p.y - a.y) * dy) / (dx * dx + dy * dy || 1)),
  )
  return Math.hypot(p.x - a.x - t * dx, p.y - a.y - t * dy)
}
function segmentDistance(a: Point, b: Point, c: Point, d: Point) {
  if (
    Math.max(a.x, b.x) >= Math.min(c.x, d.x) &&
    Math.max(c.x, d.x) >= Math.min(a.x, b.x) &&
    Math.max(a.y, b.y) >= Math.min(c.y, d.y) &&
    Math.max(c.y, d.y) >= Math.min(a.y, b.y) &&
    cross(a, b, c) * cross(a, b, d) <= 0 &&
    cross(c, d, a) * cross(c, d, b) <= 0
  )
    return 0
  return Math.min(
    pointDistance(a, c, d),
    pointDistance(b, c, d),
    pointDistance(c, a, b),
    pointDistance(d, a, b),
  )
}

// Test actual vectors, not a brush's bounding rectangle: an untouched object
// inside a bent stroke must remain free to move through that old gesture.
export function erasureTouches(e: PageErasure, strokes: Point[][]) {
  const radius = e.path?.length ? (e.radius ?? 0) : 0
  const inside = (p: Point) =>
    p.x >= e.x && p.x <= e.x + e.width && p.y >= e.y && p.y <= e.y + e.height
  const edge = [
    { x: e.x, y: e.y },
    { x: e.x + e.width, y: e.y },
    { x: e.x + e.width, y: e.y + e.height },
    { x: e.x, y: e.y + e.height },
    { x: e.x, y: e.y },
  ]
  const path = e.path?.length ? e.path : edge
  for (const stroke of strokes) {
    for (let i = 0; i < stroke.length; i++) {
      const a = stroke[Math.max(0, i - 1)]!,
        b = stroke[i]!
      if (
        Math.max(a.x, b.x) < e.x - radius ||
        Math.min(a.x, b.x) > e.x + e.width + radius ||
        Math.max(a.y, b.y) < e.y - radius ||
        Math.min(a.y, b.y) > e.y + e.height + radius
      )
        continue
      if (!e.path?.length && (inside(a) || inside(b))) return true
      for (let j = 0; j < path.length; j++)
        if (segmentDistance(a, b, path[Math.max(0, j - 1)]!, path[j]!) <= radius + 1e-8) return true
    }
  }
  return false
}

export function targetErasure(e: PageErasure, page?: Page): PageErasure {
  if (e.cuts || !page) return e
  if (page.ink) {
    const cuts = captureInk(e, page.ink)
    return { ...e, cuts, objectIds: [...new Set(cuts.map((c) => c.objectId).filter(Boolean))] }
  }
  if (e.objectIds) return e
  return {
    ...e,
    objectIds: [
      ...(page.images ?? [])
        .filter((image) => erasureTouches(e, image.strokes))
        .map((image) => image.id),
      ...(page.tables ?? [])
        .filter((table) =>
          erasureTouches(
            e,
            table.strokes.map((stroke) =>
              stroke.map((p) => ({ x: p.x + table.x, y: p.y + table.y })),
            ),
          ),
        )
        .map((table) => table.id),
    ],
  }
}
