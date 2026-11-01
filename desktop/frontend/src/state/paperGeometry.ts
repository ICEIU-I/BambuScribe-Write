import type { Point } from '../types'

export interface PaperPosition extends Point {
  page: number
}

export function nearestPaper(root: HTMLElement, y: number): HTMLElement | undefined {
  const distance = (paper: HTMLElement) => {
    const r = paper.getBoundingClientRect()
    return Math.max(r.top - y, y - r.bottom, 0)
  }
  return [...root.querySelectorAll<HTMLElement>('[data-paper]')].reduce<HTMLElement | undefined>(
    (nearest, paper) => (!nearest || distance(paper) < distance(nearest) ? paper : nearest),
    undefined,
  )
}

export function paperPosition(
  root: HTMLElement,
  point: Point,
  paperWidth: number,
): PaperPosition | undefined {
  const paper = nearestPaper(root, point.y)
  if (!paper) return
  const r = paper.getBoundingClientRect(),
    scale = r.width / paperWidth
  return {
    page: Number(paper.dataset.paper),
    x: (point.x - r.left) / scale,
    y: (point.y - r.top) / scale,
  }
}
