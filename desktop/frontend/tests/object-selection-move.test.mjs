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
const { selectionAt, imageBounds } = await import(await moduleURL('objectSelection'))
const { resetChangedObjectCuts } = await import(await moduleURL('inkCutHistory'))
const image = {
  id: 'diagram',
  page: 0,
  data: 'data:image/png;base64,AAAA',
  mode: 'line',
  x: 20,
  y: 30,
  width: 40,
  height: 20,
  rotation: 15,
  textRegions: [
    {
      id: 'erase',
      source: 'manual',
      mode: 'erase',
      text: '',
      x: 0.2,
      y: 0.2,
      width: 0.1,
      height: 0.1,
      erasePath: [{ x: 0.2, y: 0.2 }],
      eraseRadius: 0.03,
    },
  ],
}
const table = {
  id: 'table',
  page: 0,
  x: 65,
  y: 30,
  columnWidths: [20, 20],
  rowHeights: [10],
  cells: [
    {
      row: 0,
      col: 0,
      rowSpan: 1,
      colSpan: 2,
      text: '合并格',
      fontSize: 4,
      borders: { bottom: false },
    },
  ],
  style: 'grid',
  fontSize: 6,
  padding: 1,
  align: 'center',
  verticalAlign: 'middle',
}
const box = {
  ...table,
  id: 'caption',
  kind: 'textbox',
  x: 20,
  y: 80,
  columnWidths: [50],
  style: 'none',
  cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '图表注释' }],
}
const images = [image],
  tables = [table, box]
const doc = {
  settings: { paperWidth: 163, paperHeight: 240, marginTop: 5, marginBottom: 10 },
  pages: [
    {
      cells: [
        { start: 0, end: 1, x: 0, y: 0, width: 6, height: 6 },
        { start: 1, end: 2, x: 6, y: 0, width: 3, height: 6 },
      ],
      tables: [
        { id: 'table', height: 30 },
        { id: 'caption', height: 10 },
      ],
      images: [],
    },
  ],
}
const select = () => selectionAt(doc, images, tables, 0, { x: 0, y: 0 }, { x: 150, y: 100 })

const { selectedGeometry, clampSelectionMove, translateObjects } = await import(
  await moduleURL('objectSelectionMove')
)
test('moving a selection preserves measured spacing, source sizes, IDs, groups and erasures', () => {
  const grouped = tables.map((t) => ({ ...t, groupId: 'import' }))
  const measured = {
    ...doc,
    objectLayout: [
      {
        id: 'table',
        sourcePage: 0,
        source: { x: 65, y: 30, width: 40, height: 10 },
        page: 0,
        x: 70,
        y: 45,
      },
    ],
  }
  const source = JSON.stringify({ images, grouped, measured })
  const geometry = selectedGeometry(measured, images, grouped, select())
  assert.equal(geometry.tables[0].y, 45)
  const delta = { x: 8, y: 12 },
    movedImages = translateObjects(geometry.images, delta, 1),
    movedTables = translateObjects(geometry.tables, delta, 1)
  assert.deepEqual(movedImages[0], { ...image, x: 28, y: 42, page: 1 })
  assert.deepEqual(movedTables[0], { ...grouped[0], x: 78, y: 57, page: 1 })
  const cuts = [
    {
      page: 0,
      cuts: [
        { objectId: 'diagram', part: 'graphic', ranges: [[0, 0.1, 0.5]] },
        { objectId: 'table', part: 'cell:0:0', ranges: [[0, 0, 1]] },
      ],
    },
  ]
  assert.deepEqual(resetChangedObjectCuts(cuts, grouped, movedTables, images, movedImages), cuts)
  assert.equal(movedTables[0].rowHeights[0], 10) // measured 30 mm is not stored
  assert.equal(JSON.stringify({ images, grouped, measured }), source)
  const partial = { ...select(), objects: select().objects.filter((o) => o.id === 'table') }
  assert.deepEqual(
    selectedGeometry(measured, images, grouped, partial).tables.map((t) => t.id),
    ['table'],
  )
  assert.equal(selectedGeometry(measured, [], grouped, select()), null)
})
test('group edge limiting includes rotated ink, image frame and grown table height', () => {
  const rotated = { ...image, rotation: 90, width: 70, height: 10, x: 40, y: 40 }
  const selection = selectionAt(doc, [rotated], tables, 0, { x: 0, y: 0 }, { x: 160, y: 200 })
  const geometry = selectedGeometry(doc, [rotated], tables, selection)
  for (const sign of [-1, 1]) {
    const delta = clampSelectionMove(
      geometry.bounds,
      { x: sign * 1000, y: sign * 1000 },
      doc.settings,
    )
    const moved = translateObjects(geometry.images, delta, 0)[0]
    for (const r of [
      moved,
      imageBounds(moved),
      ...geometry.selection.objects.map((o) => ({ ...o, x: o.x + delta.x, y: o.y + delta.y })),
    ]) {
      assert.ok(r.x >= -1e-8 && r.x + r.width <= 163 + 1e-8)
      assert.ok(r.y >= -1e-8 && r.y + r.height <= 240 + 1e-8)
    }
  }
  assert.deepEqual(
    clampSelectionMove({ x: 0, y: 20, width: 200, height: 30 }, { x: 10, y: 5 }, doc.settings),
    { x: 0, y: 5 },
  )
})
