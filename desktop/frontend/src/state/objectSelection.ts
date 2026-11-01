import type { Document, ImageAsset, Point, RegionBounds, TableAsset } from '../types'
import { placedObjects } from './objectLayout'

export interface SelectedObject extends RegionBounds {
  id: string
  kind: 'image' | 'table'
}
export interface ObjectSelection extends RegionBounds {
  page: number
  objects: SelectedObject[]
}
export const intersects = (a: RegionBounds, b: RegionBounds) =>
  a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height

export function imageBounds(image: ImageAsset): RegionBounds {
  const a = (image.rotation * Math.PI) / 180,
    width = Math.abs(image.width * Math.cos(a)) + Math.abs(image.height * Math.sin(a)),
    height = Math.abs(image.width * Math.sin(a)) + Math.abs(image.height * Math.cos(a))
  return {
    x: image.x + (image.width - width) / 2,
    y: image.y + (image.height - height) / 2,
    width,
    height,
  }
}
export function objectBounds(doc: Document, images: ImageAsset[], tables: TableAsset[]) {
  return [
    ...placedObjects(images, doc).map((im) => ({
      ...imageBounds(im),
      id: im.id,
      page: im.page,
      kind: 'image' as const,
    })),
    ...placedObjects(tables, doc).map((table) => {
      const preview = doc.pages[table.page]?.tables?.find((t) => t.id === table.id)
      return {
        id: table.id,
        page: table.page,
        kind: 'table' as const,
        x: table.x,
        y: table.y,
        width: table.columnWidths.reduce((a, b) => a + b, 0),
        height: preview?.height ?? table.rowHeights.reduce((a, b) => a + b, 0),
      }
    }),
  ]
}
export function selectionAt(
  doc: Document,
  images: ImageAsset[],
  tables: TableAsset[],
  page: number,
  a: Point,
  b: Point,
): ObjectSelection {
  const rect = {
    x: Math.min(a.x, b.x),
    y: Math.min(a.y, b.y),
    width: Math.abs(a.x - b.x),
    height: Math.abs(a.y - b.y),
  }
  return {
    ...rect,
    page,
    objects: objectBounds(doc, images, tables).filter(
      (o) => o.page === page && intersects(rect, o),
    ),
  }
}
export function onBodyText(doc: Document, text: string, page: number, p: Point) {
  return (
    doc.pages[page]?.cells.some(
      (c) =>
        p.x >= c.x &&
        p.x <= c.x + c.width &&
        p.y >= c.y &&
        p.y <= c.y + c.height &&
        text.slice(c.start, c.end).trim() !== '',
    ) ?? false
  )
}
