import { formulaRanges } from './calculationText'

// Math is an atomic rich-text span in the hidden textarea. It is edited through
// the calculation panel, so arrow/delete/cut cannot leave half a delimiter.
export function formulaSelection(text: string, start: number, end: number) {
  for (const f of formulaRanges(text)) {
    if (start === end && start > f.start && start < f.end) {
      start = end = start - f.start < f.end - start ? f.start : f.end
    } else if (start < end) {
      if (start > f.start && start < f.end) start = f.start
      if (end > f.start && end < f.end) end = f.end
    }
  }
  return { start, end }
}
export function formulaKey(text: string, start: number, end: number, key: string) {
  if (start !== end) return undefined
  const f = formulaRanges(text).find((f) =>
    key === 'ArrowLeft' || key === 'Backspace'
      ? start > f.start && start <= f.end
      : (key === 'ArrowRight' || key === 'Delete') && start >= f.start && start < f.end,
  )
  if (!f) return undefined
  if (key === 'Backspace' || key === 'Delete') return { start: f.start, end: f.end, remove: true }
  const to = key === 'ArrowLeft' ? f.start : f.end
  return { start: to, end: to, remove: false }
}
