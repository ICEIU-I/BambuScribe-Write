import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'

const modules = new Map()
async function moduleURL(name) {
  if (modules.has(name)) return modules.get(name)
  let source = ts.transpileModule(
    await readFile(new URL(`../src/state/${name}.ts`, import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
  ).outputText
  for (const [, dependency] of source.matchAll(/from ['"]\.\/([^'"]+)['"]/g))
    source = source.replace(`'./${dependency}'`, `'${await moduleURL(dependency)}'`)
  const url = `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
  modules.set(name, url)
  return url
}

const { moveImportedTable } = await import(await moduleURL('importedTableGroup'))
const moveImportedGroup = (next, tables, doc) => moveImportedTable(next, tables, doc, true)
const { fitTableFont } = await import(await moduleURL('tableFontFit'))
const { pasteObjects } = await import(await moduleURL('objectGroupClipboard'))
const { readTable, tableMime } = await import(await moduleURL('tableClipboard'))
const make = (id, y) => ({
  id,
  groupId: 'original-group',
  page: 0,
  x: 10,
  y,
  columnWidths: [60],
  rowHeights: [10],
  cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: 'A' }],
  style: 'grid',
  fontSize: 5,
  padding: 1,
  align: 'center',
  verticalAlign: 'middle',
})
const tables = [make('title', 10), make('table', 22)]
const doc = {
  settings: { paperWidth: 163, paperHeight: 240, marginBottom: 0, marginTop: 0 },
  pages: [{ tables: tables.map((t) => ({ ...t, width: 60, height: 10 })), cells: [], images: [] }],
  objectLayout: [],
}
test('font fit uses measured spans, explicit lines, padding and a 1 mm minimum', () => {
  const t = {
    ...make('fit', 0),
    columnWidths: [30, 30],
    rowHeights: [10, 10],
    cells: [{ row: 0, col: 0, rowSpan: 2, colSpan: 2, text: 'U_{H}\n数据' }],
  }
  const preview = { fontSize: 5, cells: [{ ...t.cells[0], textWidth: 20, textHeight: 12 }] }
  assert.equal(fitTableFont(t, preview).fontSize, 7.5)
  const short = { ...t, rowHeights: [3, 3] }
  assert.ok(Math.abs(fitTableFont(short, preview).fontSize - (5 * 4) / 12) < 1e-8)
  const narrow = { ...t, columnWidths: [3, 3] }
  assert.equal(fitTableFont(narrow, preview).fontSize, 1)
  const tooSmall = { ...short, padding: 2.5 }
  const minimum = fitTableFont(tooSmall, preview)
  assert.equal(minimum.fontSize, 1)
  assert.equal(
    minimum.rowHeights.reduce((a, b) => a + b),
    7.4,
  )
  assert.deepEqual(t.rowHeights, [10, 10])
})
test('imported group moves captions, allows overlap and leaves text edits independent', () => {
  const moved = moveImportedGroup({ ...tables[1], x: 80, y: 50 }, tables, doc)
  assert.equal(moved[0].x, 80)
  assert.equal(moved[0].y, 38)
  assert.equal(moved[1].y, 50)
  assert.equal(tables[0].x, 10)
  const atTop = moveImportedGroup({ ...tables[1], y: 0 }, tables, doc)
  assert.equal(atTop[0].y, 0)
  assert.equal(atTop[1].y, 12)
  const occupied = structuredClone(doc)
  occupied.pages[0].tables.push({ ...make('other', 40), x: 80, width: 60, height: 20 })
  assert.deepEqual(moveImportedGroup({ ...tables[1], x: 80, y: 50 }, tables, occupied), moved)
  assert.equal(
    moveImportedGroup({ ...tables[1], cells: [{ ...tables[1].cells[0], text: 'B' }] }, tables, doc),
    undefined,
  )
})
test('individual movement adjusts relative placement and keeps the import association', () => {
  const moved = moveImportedTable({ ...tables[0], x: 20, y: 1 }, tables, doc)
  assert.equal(moved[0].x, 20)
  assert.equal(moved[0].y, 1)
  assert.equal(moved[0].groupId, tables[0].groupId)
  assert.equal(moved[1], tables[1])
  assert.equal(moveImportedTable({ ...tables[0], y: 22 }, tables, doc)[0].y, 22)
  const across = moveImportedTable({ ...tables[0], page: 1 }, tables, doc)
  assert.equal(across[0].page, 1)
  assert.equal(across[1], tables[1])
  const splitDoc = { ...doc, pages: [doc.pages[0], { tables: [{ ...across[0], height: 10 }] }] }
  assert.equal(moveImportedTable({ ...across[0], y: 12 }, across, splitDoc)[0].y, 12)
  assert.throws(
    () => moveImportedGroup({ ...across[0], y: 12 }, across, splitDoc),
    /分别移到同一页/,
  )
})
test('cross-page group move uses measured heights and the whole paper, with wider captions', () => {
  const source = [make('title', 5), make('table', 25)]
  source[0].columnWidths = [80]
  source.forEach((t) => (t.page = 2))
  const document = {
    ...doc,
    settings: { ...doc.settings, paperWidth: 210, paperHeight: 297, marginBottom: 25.4 },
    pages: [
      {},
      { tables: [{ id: 'existing', x: 10, y: 30, width: 180, height: 118 }] },
      { tables: source.map((t) => ({ ...t, height: t.id === 'table' ? 122.48 : 10 })) },
    ],
  }
  const moved = moveImportedGroup({ ...source[1], page: 1, x: 190, y: 290 }, source, document)
  assert.equal(moved[0].x, 130)
  assert.equal(moved[1].x, 130)
  assert.equal(moved[0].y, 297 - 122.48 - 20)
  assert.equal(moved[1].y, 297 - 122.48)
  assert.ok(moved.every((t) => t.page === 1))
  assert.deepEqual(moved[1].rowHeights, source[1].rowHeights)
  assert.equal(moved[1].fontSize, source[1].fontSize)
  assert.equal(moveImportedGroup({ ...source[1], page: 1, y: 130 }, source, document)[1].y, 130)
  document.pages[2].tables[1].height = 300
  assert.throws(() => moveImportedGroup({ ...source[1], page: 1 }, source, document), /超过整页/)
})
test('copied imports do not keep links to original captions', () => {
  const copied = {
    version: 1,
    token: 'BambuScribe:objects:test',
    bounds: { x: 10, y: 10, width: 60, height: 22 },
    images: [],
    tables,
  }
  const pasted = pasteObjects(copied, doc, [], tables, { page: 0, y: 100 })
  assert.equal(pasted.tables[0].groupId, pasted.tables[1].groupId)
  assert.notEqual(pasted.tables[0].groupId, tables[0].groupId)
  assert.equal(
    readTable({ getData: (type) => (type === tableMime ? JSON.stringify(tables[0]) : '') }).groupId,
    undefined,
  )
})
