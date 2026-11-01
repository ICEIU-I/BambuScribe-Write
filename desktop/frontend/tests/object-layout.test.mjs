import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'
const source = ts.transpileModule(
  await readFile(new URL('../src/state/objectLayout.ts', import.meta.url), 'utf8'),
  { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
).outputText
const { placedObjects, keepObjectAnchor } = await import(
  `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
)
const image = { id: 'image', page: 0, x: 10, y: 20, width: 80, height: 40, rotation: 20 }
const table = { id: 'table', page: 0, x: 10, y: 20, rowHeights: [10, 10], columnWidths: [40, 40] }
const doc = {
  objectLayout: [image, table].map((a, i) => ({
    id: a.id,
    sourcePage: a.page,
    source: { x: a.x, y: a.y, width: 80, height: i ? 20 : 40 },
    rotation: a.rotation ?? 0,
    page: 1,
    x: 10,
    y: 60,
  })),
}
test('resolved positions follow pagination without applying a stale preview to pending moves, sizes or rotations', () => {
  for (const asset of [image, table]) {
    const [shown] = placedObjects([asset], doc)
    assert.equal(shown.page, 1)
    assert.equal(shown.y, 60)
    assert.equal(asset.page, 0)
    const moved = { ...asset, y: 80 }
    assert.equal(placedObjects([moved], doc)[0], moved)
  }
  for (const pending of [
    { ...image, rotation: 30 },
    { ...image, width: 100 },
    { ...table, rowHeights: [12, 12] },
  ])
    assert.equal(placedObjects([pending], doc)[0], pending)
})
test('editing a shifted object preserves its requested anchor, while dragging commits the new position', () => {
  const shown = placedObjects([table], doc)
  const edited = keepObjectAnchor({ ...shown[0], cells: [{ text: '编辑文字' }] }, [table], shown)
  assert.equal(edited.page, 0)
  assert.equal(edited.y, 20)
  assert.equal(edited.cells[0].text, '编辑文字')
  const dragged = { ...shown[0], y: 90 }
  assert.equal(keepObjectAnchor(dragged, [table], shown), dragged)
})
