import type { Document, ImageAsset, RegionBounds, TableAsset } from '../types'
import type { ObjectSelection } from './objectSelection'
import type { PaperPosition } from './paperGeometry'
import { imageBounds, objectBounds } from './objectSelection'
import { placedObjects } from './objectLayout'
import { displayRegions } from './imageEditing'
import { copyImage, imageMime, readImage, writeImage } from './imageClipboard'
import { readTable, tableMime, writeTable } from './tableClipboard'

export const objectGroupMime = 'application/x-bambuscribe-objects-v1'
const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value))
const newId = () =>
  globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`
export interface CopiedObjects {
  version: 1
  token: string
  bounds: RegionBounds
  images: ImageAsset[]
  tables: TableAsset[]
}
export function copyObjects(
  doc: Document,
  images: ImageAsset[],
  tables: TableAsset[],
  selection: ObjectSelection,
): CopiedObjects | undefined {
  const ids = new Set(selection.objects.map((o) => o.id))
  const objects = objectBounds(doc, images, tables).filter(
    (o) => o.page === selection.page && ids.has(o.id),
  )
  if (!objects.length) return
  const placedImages = placedObjects(images, doc).filter((o) => ids.has(o.id))
  // The engine keeps both rotated ink and the original image frame on paper.
  // Include both so edge clamping cannot move one image away from its group.
  const frames = [...objects, ...placedImages]
  const x = Math.min(...frames.map((o) => o.x)),
    y = Math.min(...frames.map((o) => o.y))
  return {
    version: 1,
    token: `BambuScribe:objects:${newId()}`,
    bounds: {
      x,
      y,
      width: Math.max(...frames.map((o) => o.x + o.width)) - x,
      height: Math.max(...frames.map((o) => o.y + o.height)) - y,
    },
    images: placedImages.map((image) => {
      const preview = doc.pages[image.page]?.images?.find((im) => im.id === image.id)
      return clone({ ...image, textRegions: displayRegions(image, preview) })
    }),
    tables: clone(placedObjects(tables, doc).filter((o) => ids.has(o.id))),
  }
}
export function writeObjects(data: DataTransfer, copied: CopiedObjects) {
  if (copied.images.length === 1 && !copied.tables.length)
    writeImage(data, copyImage(copied.images[0]!))
  if (copied.tables.length === 1 && !copied.images.length) writeTable(data, copied.tables[0]!)
  const content =
    copied.tables.length === 1 && !copied.images.length
      ? data.getData('text/html')
      : '<span>图片、表格和文本框</span>'
  data.setData('text/plain', copied.token)
  data.setData(
    'text/html',
    `<div data-bambuscribe-objects="${encodeURIComponent(JSON.stringify(copied))}">${content}</div>`,
  )
  try {
    data.setData(objectGroupMime, JSON.stringify(copied))
  } catch {
    /* HTML and token survive native WebViews. */
  }
}
export function readObjects(
  data: Pick<DataTransfer, 'getData'>,
  fallback?: CopiedObjects,
): CopiedObjects | undefined {
  const custom = data.getData(objectGroupMime),
    html = data.getData('text/html')
  const embedded = html.match(/data-bambuscribe-objects="([^"]+)"/)
  if (!custom && !embedded)
    return data.getData('text/plain') === fallback?.token ? fallback : undefined
  try {
    const raw = custom || decodeURIComponent(embedded![1]!)
    if (raw.length > 64 * 1024 * 1024) return
    const copied = JSON.parse(raw) as CopiedObjects,
      b = copied.bounds
    if (
      copied.version !== 1 ||
      typeof copied.token !== 'string' ||
      !copied.token.startsWith('BambuScribe:objects:') ||
      !b ||
      ![b.x, b.y, b.width, b.height].every(Number.isFinite) ||
      b.width <= 0 ||
      b.height <= 0 ||
      !Array.isArray(copied.images) ||
      !Array.isArray(copied.tables) ||
      copied.images.length > 50 ||
      copied.tables.length > 50 ||
      (!copied.images.length && !copied.tables.length)
    )
      return
    const bounds: RegionBounds[] = []
    for (const im of copied.images) {
      if (
        !readImage({
          getData: (type) =>
            type === imageMime
              ? JSON.stringify({ version: 1, token: copied.token, image: im })
              : '',
        })
      )
        return
      bounds.push(imageBounds(im), im)
    }
    const assets = [...copied.images, ...copied.tables]
    if (
      new Set(assets.map((o) => o.id)).size !== assets.length ||
      assets.some((o) => o.page !== assets[0]!.page)
    )
      return
    for (const table of copied.tables) {
      if (!readTable({ getData: (type) => (type === tableMime ? JSON.stringify(table) : '') }))
        return
      bounds.push({
        x: table.x,
        y: table.y,
        width: table.columnWidths.reduce((a, v) => a + v, 0),
        height: table.rowHeights.reduce((a, v) => a + v, 0),
      })
    }
    if (
      bounds.some(
        (r) =>
          r.x < b.x - 1e-6 ||
          r.y < b.y - 1e-6 ||
          r.x + r.width > b.x + b.width + 1e-6 ||
          r.y + r.height > b.y + b.height + 1e-6,
      )
    )
      return
    return copied
  } catch {
    return
  }
}
export function pasteObjects(
  copied: CopiedObjects,
  doc: Document,
  _images: ImageAsset[],
  tables: TableAsset[],
  position: Pick<PaperPosition, 'page'> & Partial<PaperPosition>,
) {
  const s = doc.settings,
    b = copied.bounds,
    bottom = s.paperHeight - s.marginBottom
  if (tables.length + copied.tables.length > 50)
    throw new Error('表格和文本框共最多 50 个，请减少选中内容。')
  if (b.width > s.paperWidth || b.height > bottom)
    throw new Error('选中内容超过纸张，请减少选中内容或使用更大的纸张。')
  let page = position.page,
    // Clicking chooses the page and vertical placement. Retain the source
    // columns; only a narrower destination paper may require an X adjustment.
    x = Math.max(0, Math.min(b.x, s.paperWidth - b.width)),
    y = Math.max(0, position.y ?? b.y + 2)
  if (y + b.height > bottom + 1e-8) {
    page++
    y = Math.min(s.marginTop, bottom - b.height)
  }
  if (page > 20000) throw new Error('页面数量过多，请拆分文档。')
  const dx = x - b.x,
    dy = y - b.y
  const groups = new Map(
    copied.tables.filter((t) => t.groupId).map((t) => [t.groupId!, `group-${newId()}`]),
  )
  return {
    page,
    x,
    y,
    images: copied.images.map((im) => ({
      ...clone(im),
      id: `image-${newId()}`,
      page,
      x: im.x + dx,
      y: im.y + dy,
    })),
    tables: copied.tables.map((table) => ({
      ...clone(table),
      id: `table-${newId()}`,
      groupId: table.groupId ? groups.get(table.groupId) : undefined,
      page,
      x: table.x + dx,
      y: table.y + dy,
    })),
  }
}
