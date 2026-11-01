import type { Point } from '../types'
import type { TableAsset, TablePlacement } from './tableTypes'
import { cellBounds, tableLines } from './tableGeometry'
import { fitTableFont } from './tableFontFit'

export function nativeObjectStrokes(
  asset: TableAsset,
  preview?: TablePlacement,
  clip: (strokes: Point[][], part: string) => Point[][] = (s) => s,
): Point[][] {
  const strokes = clip(tableLines(asset), 'grid')
  const size = fitTableFont(asset, preview).fontSize
  for (const c of asset.cells) {
    const old = preview?.cells.find((v) => v.row === c.row && v.col === c.col)
    if (!old || old.text !== c.text) continue
    const bounds = cellBounds(asset, c),
      ratio = preview?.fontSize ? size / preview.fontSize : 1,
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
    for (const s of clip(old.strokes ?? [], `cell:${c.row}:${c.col}`))
      strokes.push(
        s.map((p) =>
          ratio === 1
            ? { x: p.x + (newX - oldX), y: p.y + (newY - oldY) }
            : { x: newX + (p.x - oldX) * ratio, y: newY + (p.y - oldY) * ratio },
        ),
      )
  }
  return strokes
}
export const nativeObjectPath = (strokes: Point[][]) =>
  strokes.map((s) => s.map((p, i) => `${i ? 'L' : 'M'}${p.x},${p.y}`).join(' ')).join(' ')
