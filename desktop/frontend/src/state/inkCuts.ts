import type { InkCut, InkLayer, PageErasure, Point } from '../types'
type Interval = [number, number]
function rectangle(
  a: Point,
  b: Point,
  left: number,
  top: number,
  right: number,
  bottom: number,
): Interval[] {
  let lo = 0,
    hi = 1
  for (const [start, delta, min, max] of [
    [a.x, b.x - a.x, left, right],
    [a.y, b.y - a.y, top, bottom],
  ]) {
    if (Math.abs(delta!) < 1e-12) {
      if (start! < min! || start! > max!) return []
      continue
    }
    const x = (min! - start!) / delta!,
      y = (max! - start!) / delta!
    lo = Math.max(lo, Math.min(x, y))
    hi = Math.min(hi, Math.max(x, y))
    if (hi <= lo) return []
  }
  return [[lo, hi]]
}
function circle(a: Point, b: Point, p: Point, r: number): Interval[] {
  const dx = b.x - a.x,
    dy = b.y - a.y,
    x = a.x - p.x,
    y = a.y - p.y,
    aa = dx * dx + dy * dy,
    bb = 2 * (x * dx + y * dy),
    cc = x * x + y * y - r * r
  if (aa < 1e-20) return cc <= 0 ? [[0, 1]] : []
  const det = bb * bb - 4 * aa * cc
  if (det <= 0) return []
  const root = Math.sqrt(det),
    lo = Math.max(0, (-bb - root) / (2 * aa)),
    hi = Math.min(1, (-bb + root) / (2 * aa))
  return hi > lo ? [[lo, hi]] : []
}
function intervals(a: Point, b: Point, e: PageErasure): Interval[] {
  if (!e.path?.length) return rectangle(a, b, e.x, e.y, e.x + e.width, e.y + e.height)
  const out: Interval[] = [],
    radius = e.radius ?? 0
  e.path.forEach((p, i) => {
    out.push(...circle(a, b, p, radius))
    if (!i) return
    const q = e.path![i - 1]!,
      dx = p.x - q.x,
      dy = p.y - q.y,
      length = Math.hypot(dx, dy)
    if (length < 1e-10) return
    const local = (v: Point) => ({
      x: ((v.x - q.x) * dx + (v.y - q.y) * dy) / length,
      y: (-(v.x - q.x) * dy + (v.y - q.y) * dx) / length,
    })
    out.push(...rectangle(local(a), local(b), 0, -radius, length, radius))
  })
  return out
}
function merged(ranges: Interval[]): Interval[] {
  const out: Interval[] = []
  for (const c of ranges.sort((a, b) => a[0] - b[0])) {
    const last = out.at(-1)
    if (last && c[0] <= last[1] + 1e-10) last[1] = Math.max(last[1], c[1])
    else out.push([...c])
  }
  return out
}
// Resolve a gesture exactly once to portions of specific source vectors.
export function captureInk(e: PageErasure, layers: InkLayer[]): InkCut[] {
  const out: InkCut[] = []
  for (const layer of layers) {
    if (layer.objectId && e.objectIds && !e.objectIds.includes(layer.objectId)) continue
    layer.strokes.forEach((stroke, index) => {
      const ranges: Interval[] = []
      for (let i = 1; i < stroke.length; i++)
        for (const [lo, hi] of intervals(stroke[i - 1]!, stroke[i]!, e))
          ranges.push([i - 1 + lo, i - 1 + hi])
      if (ranges.length)
        out.push({
          objectId: layer.objectId,
          part: layer.part,
          source: layer.source,
          stroke: index,
          ranges: merged(ranges),
        })
    })
  }
  return out
}
// The same vertex fractions apply after translation, scaling or text reflow.
export function cutInk(
  strokes: Point[][],
  layer: Pick<InkLayer, 'objectId' | 'part' | 'source'> | undefined,
  erasures: PageErasure[],
): Point[][] {
  if (!layer) return strokes
  const byStroke = new Map<number, Interval[]>()
  for (const e of erasures)
    for (const c of e.cuts ?? []) {
      if (c.objectId !== layer.objectId || c.part !== layer.part || c.source !== layer.source)
        continue
      byStroke.set(c.stroke, [...(byStroke.get(c.stroke) ?? []), ...c.ranges])
    }
  if (!byStroke.size) return strokes
  return strokes.flatMap((stroke, i) => {
    if (!byStroke.has(i)) return [stroke]
    if (stroke.length < 2) return []
    const out: Point[][] = [],
      end = stroke.length - 1
    const at = (v: number) => {
      const n = Math.min(Math.floor(v), stroke.length - 2),
        t = v - n,
        a = stroke[n]!,
        b = stroke[n + 1]!
      return { x: a.x + t * (b.x - a.x), y: a.y + t * (b.y - a.y) }
    }
    const add = (lo: number, hi: number) => {
      if (hi - lo < 1e-10) return
      const path = [at(lo)]
      for (let j = Math.floor(lo) + 1; j < hi; j++) path.push(stroke[j]!)
      path.push(at(hi))
      out.push(path)
    }
    let cursor = 0
    for (const c of merged(byStroke.get(i)!)) {
      const lo = Math.min(end, c[0]),
        hi = Math.min(end, c[1])
      if (lo > cursor) add(cursor, lo)
      cursor = Math.max(cursor, hi)
    }
    add(cursor, end)
    return out
  })
}
export function validInkCuts(cuts: unknown): boolean {
  return (
    cuts === undefined ||
    (Array.isArray(cuts) &&
      cuts.length <= 2000000 &&
      cuts.every(
        (c) =>
          c &&
          typeof c.objectId === 'string' &&
          c.objectId.length <= 1000 &&
          typeof c.part === 'string' &&
          c.part.length > 0 &&
          c.part.length <= 1000 &&
          typeof c.source === 'string' &&
          c.source.length > 0 &&
          c.source.length <= 1000 &&
          Number.isInteger(c.stroke) &&
          c.stroke >= 0 &&
          c.stroke <= 2000000 &&
          Array.isArray(c.ranges) &&
          c.ranges.every(
            (v: number[]) =>
              Array.isArray(v) &&
              v.length === 2 &&
              Number.isFinite(v[0]) &&
              Number.isFinite(v[1]) &&
              v[0]! >= 0 &&
              v[1]! > v[0]! &&
              v[1]! <= 2000000,
          ),
      ))
  )
}
