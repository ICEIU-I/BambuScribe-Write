import type { CalculationRecord } from '../services/calculationTypes'

export function calculationAt(text: string, records: CalculationRecord[], index: number) {
  for (const record of records) {
    let from = text.indexOf(record.text)
    while (from >= 0) {
      if (index >= from && index <= from + record.text.length)
        return { record, from, to: from + record.text.length }
      from = text.indexOf(record.text, from + 1)
    }
  }
}
export function formulaRanges(text: string) {
  const ranges: { start: number; end: number; source: string }[] = []
  for (let i = 0; i + 1 < text.length; i++) {
    if (text[i] !== '\\') continue
    if (text[i + 1] === '\\') {
      i++
      continue
    }
    if (!'(['.includes(text[i + 1])) continue
    const start = i,
      stack: string[] = []
    let end = 0
    for (; i + 1 < text.length && !'\n\f'.includes(text[i]); i++) {
      if (text[i] !== '\\') continue
      const delimiter = text[++i]
      if (delimiter === '(' || delimiter === '[') stack.push(delimiter === '(' ? ')' : ']')
      else if (delimiter === ')' || delimiter === ']') {
        if (stack.pop() !== delimiter) break
        if (!stack.length) {
          end = i + 1
          break
        }
      }
    }
    if (end) ranges.push({ start, end, source: text.slice(start + 2, end - 2) })
  }
  return ranges
}
export function insertCalculation(text: string, from: number, to: number, generated: string) {
  const prefix = from && !'\n\f'.includes(text[from - 1]) ? '\n' : ''
  const suffix = to < text.length && !'\n\f'.includes(text[to]) ? '\n' : ''
  return prefix + generated + suffix
}
