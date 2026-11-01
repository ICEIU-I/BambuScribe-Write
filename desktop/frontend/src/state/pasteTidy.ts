// Rich-text views build the plain-text clipboard flavour by collapsing block
// boundaries — the blank line that separates a heading from the paragraph above
// it — into a run of spaces, while the line breaks inside a block survive as
// newlines. A pasted heading therefore arrives glued to the line before it.
// Turning those runs back into line breaks is what restores the structure.
//
// The run is the only signal available: a single space can be deliberate (it
// sits inside a title), two or more inside Chinese prose cannot, which is why
// the caller keeps the rewrite undoable and reports how many runs it changed.

const RUN = /[ \t]{2,}/g

export interface TidyResult {
  text: string
  count: number
}

export function tidySpacing(text: string): TidyResult {
  let count = 0
  const protectedRanges = mathInputRanges(text)
  const tidied = text.replace(RUN, (run: string, offset: number) => {
    if (protectedRanges.some(([start, end]) => offset >= start && offset < end)) return run
    // A run that opens or closes a line is indentation or trailing space the
    // writer meant to keep: only a run between two pieces of text marks the
    // boundary the clipboard turned a line break into.
    const before = text[offset - 1],
      after = text[offset + run.length]
    if (!before || !after || before === '\n' || after === '\n') return run
    count++
    return '\n'
  })
  return { text: tidied, count }
}

// Scope delimiters are explicit: preserve spaces inside both complete and
// unfinished math input so a paste never inserts a newline into an argument.
function mathInputRanges(text: string): [number, number][] {
  const ranges: [number, number][] = []
  const starts = /\\(?:frac|sqrt|bar)\b|[_^](?=\{)|√(?=[(（{])|\\[([]/g
  for (const match of text.matchAll(starts)) {
    const start = match.index!
    if (ranges.some(([a, b]) => start >= a && start < b)) continue
    let i = start + match[0].length
    if (match[0] === '\\(' || match[0] === '\\[') {
      const close = match[0] === '\\(' ? '\\)' : '\\]'
      const at = text.indexOf(close, i)
      const newline = text.indexOf('\n', i)
      ranges.push([
        start,
        at >= 0 && (newline < 0 || at < newline) ? at + 2 : newline >= 0 ? newline : text.length,
      ])
      continue
    }
    while (i < text.length && '{(（['.includes(text[i])) {
      const stack: string[] = []
      const pairs: Record<string, string> = { '{': '}', '(': ')', '（': '）', '[': ']' }
      do {
        const c = text[i++]
        if (c === '\\' && i < text.length && '_^{}\\'.includes(text[i])) {
          i++
          continue
        }
        if (pairs[c]) stack.push(pairs[c])
        else if (c === stack.at(-1)) stack.pop()
        else if (c === '\n') {
          i--
          break
        }
      } while (i < text.length && stack.length)
      if (stack.length) break
    }
    ranges.push([start, i])
  }
  return ranges
}

// The range an edit inserted, found by trimming the unchanged prefix and
// suffix around it. A paste leaves everything outside its insertion point
// alone, so the remainder is exactly what the clipboard contributed.
export function insertedRange(before: string, after: string): [number, number] {
  const limit = Math.min(before.length, after.length)
  let start = 0
  while (start < limit && before[start] === after[start]) start++
  let tail = 0
  while (
    tail < limit - start &&
    before[before.length - 1 - tail] === after[after.length - 1 - tail]
  )
    tail++
  return [start, after.length - tail]
}
