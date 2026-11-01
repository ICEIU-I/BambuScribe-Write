import type { Document, ImageAsset, TableAsset } from '../types'

// Use the engine's placement only for the source geometry it measured. Edits
// awaiting the next preview must retain their new location and dimensions.
export function placedObjects<T extends ImageAsset | TableAsset>(
  assets: T[],
  doc: Document | null,
): T[] {
  const positions = new Map(doc?.objectLayout?.map((p) => [p.id, p]))
  return assets.map((asset) => {
    const p = positions.get(asset.id)
    if (!p) return asset
    const image = 'width' in asset
    const width = image ? asset.width : asset.columnWidths.reduce((n, v) => n + v, 0)
    const height = image ? asset.height : asset.rowHeights.reduce((n, v) => n + v, 0)
    if (
      asset.page !== p.sourcePage ||
      asset.x !== p.source.x ||
      asset.y !== p.source.y ||
      Math.abs(width - p.source.width) > 1e-8 ||
      Math.abs(height - p.source.height) > 1e-8 ||
      (image && asset.rotation !== p.rotation)
    )
      return asset
    return { ...asset, page: p.page, x: p.x, y: p.y }
  })
}

// Cell/region edits keep the requested anchor. Only moving an object changes
// it, so automatic avoidance remains reversible when content is removed.
export function keepObjectAnchor<T extends ImageAsset | TableAsset>(
  next: T,
  assets: T[],
  placed: T[],
): T {
  const source = assets.find((a) => a.id === next.id)
  const shown = placed.find((a) => a.id === next.id)
  if (source && shown && next.page === shown.page && next.x === shown.x && next.y === shown.y)
    return { ...next, page: source.page, x: source.x, y: source.y }
  return next
}
