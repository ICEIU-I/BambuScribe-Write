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
  return [...text.matchAll(/^\\\(([^\n\f]*)\\\)$/gm)].map((m) => ({
    start: m.index!,
    end: m.index! + m[0].length,
    source: m[1],
  }))
}
export function insertCalculation(text: string, from: number, to: number, generated: string) {
  const prefix = from && !'\n\f'.includes(text[from - 1]) ? '\n' : ''
  const suffix = to < text.length && !'\n\f'.includes(text[to]) ? '\n' : ''
  return prefix + generated + suffix
}
