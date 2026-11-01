import type { Point } from '../types'
import type { TableAsset, TablePlacement } from './tableTypes'
import { cellBounds, tableLines } from './tableGeometry'
import { previewTableRules } from './tablePreviewRules'
import { fitTableCellFont } from './tableFontFit'

export function nativeObjectStrokes(
  asset: TableAsset,
  preview?: TablePlacement,
  clip: (strokes: Point[][], part: string) => Point[][] = (s) => s,
): Point[][] {
  const strokes = clip(previewTableRules(asset, preview) ?? tableLines(asset), 'grid')
  for (const c of asset.cells) {
    const old = preview?.cells.find((v) => v.row === c.row && v.col === c.col)
    if (!old || old.text !== c.text) continue
    const bounds = cellBounds(asset, c),
      previousSize = old.fontSize ?? preview?.fontSize ?? asset.fontSize,
      size = fitTableCellFont(asset, c, old, previousSize, preview?.maxFontSize),
      ratio = Math.abs(size - previousSize) < 1e-10 ? 1 : size / previousSize,
      oldPadding = preview?.padding ?? asset.padding,
      a = c.align ?? asset.align,
      v = c.verticalAlign ?? asset.verticalAlign,
      oldX =
        old.x +
        (a === 'center' ? old.width / 2 : a === 'right' ? old.width - oldPadding : oldPadding),
      newX =
        bounds.x +
        (a === 'center'
          ? bounds.width / 2
          : a === 'right'
            ? bounds.width - asset.padding
            : asset.padding),
      oldY =
        old.y +
        (v === 'middle' ? old.height / 2 : v === 'bottom' ? old.height - oldPadding : oldPadding),
      newY =
        bounds.y +
        (v === 'middle'
          ? bounds.height / 2
          : v === 'bottom'
            ? bounds.height - asset.padding
            : asset.padding)
    const transformed = (old.strokes ?? []).map((s, index) => {
      const gap = old.textSpacing?.strokeGaps[index] ?? [0, 0]
      return s.map((p, j) =>
        ratio === 1
          ? { x: p.x + (newX - oldX), y: p.y + (newY - oldY) }
          : {
              x:
                newX +
                (p.x - oldX) * ratio +
                (gap[0] + ((gap[1] - gap[0]) * j) / Math.max(1, s.length - 1)) * (1 - ratio),
              y: newY + (p.y - oldY) * ratio,
            },
      )
    })
    strokes.push(...clip(transformed, `cell:${c.row}:${c.col}`))
  }
  return strokes
}
export const nativeObjectPath = (strokes: Point[][]) =>
  strokes.map((s) => s.map((p, i) => `${i ? 'L' : 'M'}${p.x},${p.y}`).join(' ')).join(' ')
