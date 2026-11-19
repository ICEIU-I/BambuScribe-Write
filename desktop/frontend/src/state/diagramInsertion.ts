import type { Document } from '../types'
import type { LocatedCaret } from './caretGeometry'

export type DiagramAnchor = { page: number; x: number; y: number }

// The initial textarea selection is zero even when the user has never placed
// a cursor. Importing from that stale selection would prepend every figure.
export function diagramInsertionAnchor(
  doc: Document | null,
  text: string,
  caret: LocatedCaret,
  focused: boolean,
  pageIndex: number,
): DiagramAnchor {
  if (focused && (caret.blankLines !== undefined || caret.index < text.trimEnd().length))
    return { page: caret.page, x: caret.x, y: caret.y }
  const x = doc?.settings.marginLeft ?? 8
  for (let page = (doc?.pages.length ?? 0) - 1; page >= 0; page--) {
    const layout = doc!.pages[page]
    const ink = [...layout.cells, ...(layout.formulas ?? [])]
    if (!ink.length) continue
    const bottom = Math.max(...ink.map((c) => c.y + c.height))
    const blank = layout.blankCarets?.find((c) => c.y >= bottom - 1e-8)
    const lastRow = Math.max(...layout.carets.map((c) => c.y), doc!.settings.marginTop)
    return {
      page,
      x,
      // Leave room for the engine's one-millimeter exclusion gap when tight
      // line spacing puts the next blank row directly against the ink.
      y: Math.max(bottom + 2, blank?.y ?? lastRow + layout.lineHeight + doc!.settings.paragraphGap),
    }
  }
  return { page: pageIndex, x, y: doc?.settings.marginTop ?? 8 }
}
