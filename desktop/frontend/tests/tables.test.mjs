import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'
async function moduleURL(name, replacements = {}) {
  let source = ts.transpileModule(
    await readFile(new URL(`../src/state/${name}.ts`, import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
  ).outputText
  for (const [name, url] of Object.entries(replacements))
    source = source.replaceAll(`'${name}'`, JSON.stringify(url))
  return `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
}
const geometry = await moduleURL('tableGeometry')
const { cellAt, selectionBounds, tableLines } = await import(geometry)
const operations = await moduleURL('tableOperations', { './tableGeometry': geometry })
const { createTable, mergeCells, splitCells, changeAxis, pasteCells, setBorders, resizeTable } =
  await import(operations)
const { createFreeTextBox } = await import(
  await moduleURL('freeTextBox', { './tableOperations': operations })
)
const { parseTableText, tableText, readTable, writeTable, tableMime } = await import(
  await moduleURL('tableClipboard', { './tableGeometry': geometry })
)
const { createEditorHistory } = await import(await moduleURL('editorHistory'))
const { outputAvailability } = await import(await moduleURL('outputAvailability'))
const settings = { paperWidth: 163, paperHeight: 240, marginLeft: 0, marginTop: 0, fontSize: 6 }
const make = () =>
  createTable({ rows: 3, cols: 3, width: 90, style: 'grid' }, settings, { page: 0, x: 10, y: 20 })
test('independent captions fit the paper and retain identity through native clipboard fallback', () => {
  const box = createFreeTextBox(settings, { page: 3, x: 160, y: 239 })
  assert.equal(make().fontSize, settings.fontSize)
  assert.equal(box.fontSize, settings.fontSize)
  assert.equal(box.kind, 'textbox')
  assert.ok(box.x + box.columnWidths[0] <= settings.paperWidth)
  assert.ok(box.y + box.rowHeights[0] <= settings.paperHeight)
  assert.equal(box.page, 3)
  assert.deepEqual(tableLines(box), [])
  box.cells[0].text = '表1 I-U关系\n10⁻¹¹ A'
  const data = new Map()
  const clipboard = {
    setData: (type, text) => data.set(type, text),
    getData: (type) => data.get(type) ?? '',
  }
  writeTable(clipboard, box)
  data.delete(tableMime)
  assert.deepEqual(readTable(clipboard), box)
  box.style = 'grid'
  writeTable(clipboard, box)
  assert.equal(readTable(clipboard), undefined)
})
function topology(t) {
  const owners = new Set()
  for (const c of t.cells)
    for (let r = c.row; r < c.row + c.rowSpan; r++)
      for (let col = c.col; col < c.col + c.colSpan; col++) {
        assert.ok(r < t.rowHeights.length && col < t.columnWidths.length)
        assert.equal(owners.has(`${r}:${col}`), false)
        owners.add(`${r}:${col}`)
      }
  assert.equal(owners.size, t.columnWidths.length * t.rowHeights.length)
}
test('merge preserves all contents and selection expands to whole merged regions', () => {
  const original = pasteCells(make(), 0, 0, [
    ['A', 'B'],
    ['C', 'D'],
  ])
  const merged = mergeCells(original, { r0: 0, c0: 0, r1: 1, c1: 1 })
  assert.equal(cellAt(merged, 1, 1).text, 'A\nB\nC\nD')
  topology(merged)
  assert.deepEqual(selectionBounds(merged, { r0: 1, c0: 1, r1: 2, c1: 2 }), {
    r0: 0,
    c0: 0,
    r1: 2,
    c1: 2,
  })
  const split = splitCells(merged, { r0: 0, c0: 0, r1: 0, c1: 0 })
  topology(split)
  assert.equal(split.cells.length, 9)
  assert.equal(cellAt(split, 0, 0).text, 'A\nB\nC\nD')
  assert.equal(original.cells[0].text, 'A')
})
test('row and column insertion/deletion retains valid spans and merged contents', () => {
  const merged = mergeCells(
    pasteCells(make(), 0, 0, [
      ['A', 'B'],
      ['C', 'D'],
    ]),
    { r0: 0, c0: 0, r1: 1, c1: 1 },
  )
  for (const axis of ['row', 'col']) {
    const inserted = changeAxis(merged, axis, 1)
    topology(inserted)
    assert.equal(cellAt(inserted, 0, 0)[axis === 'row' ? 'rowSpan' : 'colSpan'], 3)
    const restored = changeAxis(inserted, axis, 1, true)
    topology(restored)
    assert.deepEqual(restored, merged)
    const deletedOrigin = changeAxis(merged, axis, 0, true)
    topology(deletedOrigin)
    assert.equal(cellAt(deletedOrigin, 0, 0).text, 'A\nB\nC\nD')
  }
})
test('TSV quoting round trips multiline/tab cells, blank columns and trailing newlines', () => {
  assert.deepEqual(parseTableText('"a\nb"\t"c\t\"\"d\"\""\t\n\t2\t3\n'), [
    ['a\nb', 'c\t"d"', ''],
    ['', '2', '3'],
  ])
  const pasted = pasteCells(make(), 0, 0, [
    ['a\nb', 'c\t"d"', ''],
    ['', '2', '3'],
  ])
  assert.deepEqual(parseTableText(tableText(pasted, { r0: 0, c0: 0, r1: 1, c1: 2 })), [
    ['a\nb', 'c\t"d"', ''],
    ['', '2', '3'],
  ])
  const expanded = pasteCells(make(), 2, 2, [
    ['A', 'B'],
    ['C', 'D'],
  ])
  topology(expanded)
  assert.equal(expanded.rowHeights.length, 4)
  assert.equal(expanded.columnWidths.length, 4)
  assert.throws(
    () =>
      pasteCells(make(), 49, 19, [
        ['A', 'B'],
        ['C', 'D'],
      ]),
    /超过/,
  )
})
test('resize keeps glyph size unless explicitly requested and shared border edits can be restored', () => {
  const original = make(),
    resized = resizeTable(original, 120, 60)
  assert.equal(resized.fontSize, original.fontSize)
  assert.ok(
    Math.abs(resizeTable(original, 120, 60, true).fontSize - (original.fontSize * 120) / 90) <
      1e-10,
  )
  const b = { r0: 0, c0: 0, r1: 0, c1: 0 },
    hidden = setBorders(original, b, 'none')
  assert.equal(cellAt(hidden, 1, 0).borders.top, false)
  const restored = setBorders(hidden, b, 'all')
  assert.equal(cellAt(restored, 1, 0).borders.top, true)
  const lines = tableLines(original)
  assert.equal(lines.length, 8)
  const merged = mergeCells(original, { r0: 0, c0: 0, r1: 0, c1: 1 })
  assert.ok(!tableLines(merged).some((s) => s[0].x === 30 && s[1].x === 30 && s[0].y === 0))
})
test('table object clipboard rejects malformed topology and retains editable styles and content', () => {
  const original = mergeCells(make(), { r0: 0, c0: 0, r1: 0, c1: 1 }),
    data = { getData: (type) => (type === tableMime ? JSON.stringify(original) : '') }
  assert.deepEqual(readTable(data), original)
  const bad = { ...original, cells: [...original.cells, original.cells[0]] }
  assert.equal(readTable({ getData: () => JSON.stringify(bad) }), undefined)
  assert.equal(readTable({ getData: () => '{bad json' }), undefined)
})
test('table edits join the same undo timeline as body and images', () => {
  const history = createEditorHistory(),
    initial = {
      text: '正文',
      images: [],
      tables: [],
      imageId: '',
      regionId: '',
      tableId: '',
      start: 0,
      end: 0,
    },
    inserted = { ...initial, tables: [make()], tableId: 'table' },
    typed = { ...inserted, text: '正文变化' }
  history.record(initial, inserted)
  history.record(inserted, typed, 'typing')
  assert.deepEqual(history.undo(), inserted)
  assert.deepEqual(history.undo(), initial)
  assert.deepEqual(history.redo(), inserted)
})
test('paper erasure is an independent history step between text and object edits', () => {
  const history = createEditorHistory()
  const initial = {
    text: '正文',
    images: [],
    tables: [],
    erasures: [],
    imageId: '',
    regionId: '',
    start: 0,
    end: 0,
  }
  const erased = {
    ...initial,
    erasures: [{ id: 'e', page: 0, x: 10, y: 20, width: 30, height: 40 }],
  }
  const inserted = { ...erased, tables: [make()] }
  history.record(initial, erased)
  history.record(erased, inserted)
  assert.deepEqual(history.undo(), erased)
  assert.deepEqual(history.undo(), initial)
  assert.deepEqual(history.redo(), erased)
  assert.deepEqual(history.redo(), inserted)
})
test('invalid table on a later page blocks all-page output while a valid current page stays available', () => {
  const valid = { number: 1, strokes: [[{}]], missing: [] },
    invalid = { number: 2, strokes: [[{}]], missing: [], issues: ['表格超出纸张'] }
  const state = {
    document: { pages: [valid, invalid] },
    index: 0,
    ready: true,
    loading: false,
    dirty: false,
    busy: false,
  }
  assert.deepEqual(outputAvailability(state), { current: '', all: '第 2 页：表格超出纸张' })
  assert.equal(outputAvailability({ ...state, index: 1 }).current, '第 2 页：表格超出纸张')
})

test('native HTML clipboard fallback retains merges without executing cell markup', () => {
  const table = mergeCells(make(), { r0: 0, c0: 0, r1: 0, c1: 1 })
  table.cells[0].text = '<script>alert("x")</script>\n&'
  const values = new Map(),
    data = {
      getData: (type) => values.get(type) ?? '',
      setData: (type, value) => values.set(type, value),
    }
  writeTable(data, table)
  values.delete(tableMime)
  assert.deepEqual(readTable(data), table)
  assert.ok(values.get('text/html').includes('&lt;script&gt;'))
  assert.ok(!values.get('text/html').includes('<script>'))
  values.set('text/html', '<table><tr><td>Other</td></tr></table>')
  assert.equal(readTable(data), undefined)
})
