import type { Document } from '../types'
import type { LocatedCaret } from './caretGeometry'
import { blankRows } from './blankCaret'

export interface TextInsertionAnchor {
  from: number
  to: number
  blankLines: number
}

export function textInsertionAnchor(
  text: string,
  document: Document | null,
  pageIndex: number,
  start: number,
  end: number,
  caret: LocatedCaret,
  focused: boolean,
): TextInsertionAnchor {
  if (focused || start !== end || caret.page === pageIndex)
    return { from: start, to: end, blankLines: start === end ? (caret.blankLines ?? 0) : 0 }
  const page = document?.pages[pageIndex]
  const blank = blankRows(document, text.includes('\f')).find((r) => r.page === pageIndex)
  const hasText = !!page?.text.replace(/\f$/, '')
  const target = hasText ? page?.carets[0] : blank?.stops[0]
  const from = target?.index ?? page?.carets[0]?.index ?? text.length
  return {
    from,
    to: from,
    blankLines: hasText ? 0 : (blank?.stops[0].blankLines ?? 0),
  }
}
