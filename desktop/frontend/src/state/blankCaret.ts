import type { Document } from '../types'
import type { CaretRow } from './caretGeometry'

// Native trailing rows include paragraph spacing, object avoidance and section
// boundaries. Keep the fallback for empty, object-only pages without text flow.
export function blankRows(doc: Document | null, explicit = false): CaretRow[] {
  if (!doc) return []
  const rows: CaretRow[] = []
  const s = doc.settings
  const x = s.marginLeft + s.indent * ((s.fontSize * s.letterSpacing) / 100 + s.letterGap)
  const bottom = s.paperHeight - s.marginBottom
  let offset = 0
  const empty = doc.pages.every((page) => page.text === '')
  for (const [page, layout] of doc.pages.entries()) {
    if (rows.length >= 20000) break
    const index = explicit ? offset : 0
    offset += layout.text.length
    if (!empty && layout.blankCarets) {
      rows.push(
        ...layout.blankCarets.map((stop) => ({
          page,
          y: stop.y,
          stops: [{ ...stop, page }],
        })),
      )
      continue
    }
    if (!empty && !explicit) continue
    if (explicit && layout.text.replace(/\f$/, '') !== '') continue
    const step = layout.lineHeight + s.paragraphGap
    if (!Number.isFinite(step) || step <= 0) continue
    for (let row = 0; row < layout.lineCount && rows.length < 20000; row++) {
      const y = s.marginTop + row * step
      if (y >= bottom - 1e-8) break
      rows.push({
        page,
        y,
        stops: [{ page, x, y, index, blankLines: explicit ? row : rows.length }],
      })
    }
  }
  return rows
}
