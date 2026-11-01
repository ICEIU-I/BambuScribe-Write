import { computed, inject, type InjectionKey, type Ref } from 'vue'
import type { PageErasure, Point } from '../types'

export const pageErasuresKey: InjectionKey<Ref<PageErasure[]>> = Symbol('page-erasures')
export interface InkFrame {
  id?: string
  page: number
  x: number
  y: number
  width: number
  height: number
  rotation?: number
}
export const paperBrushRadius = (value: number) => value * 100
export function erasureBounds(points: Point[]) {
  const xs = points.map((p) => p.x),
    ys = points.map((p) => p.y)
  const x = Math.min(...xs),
    y = Math.min(...ys)
  return { x, y, width: Math.max(...xs) - x, height: Math.max(...ys) - y }
}

// The SVG mask is measured in paper mm, then transformed into each ink
// layer's local frame. Selection handles and paper rules stay outside it.
export function paperMask(erasures: PageErasure[], frame: InkFrame) {
  const regions = erasures.filter(
    (e) =>
      !e.cuts &&
      e.page === frame.page &&
      (!frame.id || !e.objectIds || e.objectIds.includes(frame.id)),
  )
  if (!regions.length || frame.width <= 0 || frame.height <= 0) return {}
  const shapes = regions
    .map((e) =>
      e.path?.length
        ? `<path d="${e.path.map((p, i) => `${i ? 'L' : 'M'}${p.x} ${p.y}`).join(' ')}" fill="none" stroke="black" stroke-width="${2 * (e.radius ?? 0)}" stroke-linecap="round" stroke-linejoin="round"/><circle cx="${e.path[0]!.x}" cy="${e.path[0]!.y}" r="${e.radius}" fill="black"/>`
        : `<rect x="${e.x}" y="${e.y}" width="${e.width}" height="${e.height}" fill="black"/>`,
    )
    .join('')
  const { width, height, x, y, rotation = 0 } = frame
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${width} ${height}" preserveAspectRatio="none"><mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="${width}" height="${height}"><rect width="${width}" height="${height}" fill="white"/><g transform="rotate(${-rotation} ${width / 2} ${height / 2}) translate(${-x} ${-y})">${shapes}</g></mask><rect width="${width}" height="${height}" fill="white" mask="url(#m)"/></svg>`
  const url = `url("data:image/svg+xml,${encodeURIComponent(svg)}")`
  return {
    maskImage: url,
    WebkitMaskImage: url,
    maskSize: '100% 100%',
    WebkitMaskSize: '100% 100%',
    maskRepeat: 'no-repeat',
    WebkitMaskRepeat: 'no-repeat',
  }
}
export function usePageMask(frame: () => InkFrame) {
  const erasures = inject(pageErasuresKey, undefined)
  return computed(() => paperMask(erasures?.value ?? [], frame()))
}
