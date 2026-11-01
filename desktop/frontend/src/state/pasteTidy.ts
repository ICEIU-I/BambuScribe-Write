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
  const tidied = text.replace(RUN, (run: string, offset: number) => {
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
