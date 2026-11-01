import type { Document, ImageAsset, Point, RegionBounds, TableAsset } from '../types'
import { objectBounds, type ObjectSelection } from './objectSelection'
import { placedObjects } from './objectLayout'

export function enclosingBounds(objects: RegionBounds[]): RegionBounds {
  const x = Math.min(...objects.map((o) => o.x)),
    y = Math.min(...objects.map((o) => o.y))
  return {
    x,
    y,
    width: Math.max(...objects.map((o) => o.x + o.width)) - x,
    height: Math.max(...objects.map((o) => o.y + o.height)) - y,
  }
}

export function selectedGeometry(
  doc: Document,
  images: ImageAsset[],
  tables: TableAsset[],
  selection: ObjectSelection,
) {
  const ids = new Set(selection.objects.map((o) => o.id)),
    objects = objectBounds(doc, images, tables).filter(
      (o) => ids.has(o.id) && o.page === selection.page,
    )
  if (!objects.length || objects.length !== ids.size) return null
  const selectedImages = placedObjects(images, doc).filter((im) => ids.has(im.id)),
    selectedTables = placedObjects(tables, doc).filter((t) => ids.has(t.id))
  return {
    selection: { ...enclosingBounds(objects), page: selection.page, objects },
    // Fit the rotated ink AND the original frame, as the engine requires both.
    bounds: enclosingBounds([...objects, ...selectedImages]),
    images: selectedImages,
    tables: selectedTables,
  }
}
export function clampSelectionMove(
  bounds: RegionBounds,
  delta: Point,
  paper: { paperWidth: number; paperHeight: number },
): Point {
  // An existing oversized group may be moved along the other axis without
  // distorting it or silently discarding objects.
  const axis = (value: number, start: number, size: number, limit: number) =>
    size > limit ? 0 : Math.max(-start, Math.min(limit - start - size, value))
  return {
    x: axis(delta.x, bounds.x, bounds.width, paper.paperWidth),
    y: axis(delta.y, bounds.y, bounds.height, paper.paperHeight),
  }
}
export function translateSelection(selection: ObjectSelection, delta: Point, page: number) {
  return {
    ...selection,
    page,
    x: selection.x + delta.x,
    y: selection.y + delta.y,
    objects: selection.objects.map((o) => ({ ...o, x: o.x + delta.x, y: o.y + delta.y })),
  }
}
export function translateObjects<T extends ImageAsset | TableAsset>(
  objects: T[],
  delta: Point,
  page: number,
): T[] {
  return objects.map((o) => ({ ...o, page, x: o.x + delta.x, y: o.y + delta.y }))
}
