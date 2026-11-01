import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'

const source = ts.transpileModule(
  await readFile(new URL('../src/state/diagramPlacement.ts', import.meta.url), 'utf8'),
  { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
).outputText
const { placeDiagram, nextDiagramAnchor } = await import(
  `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
)
const settings = {
  paperWidth: 210,
  paperHeight: 297,
  marginTop: 25.4,
  marginBottom: 25.4,
  marginLeft: 17,
  marginRight: 17,
}
test('smart import retains the chosen row and size, including an overflowing insertion', () => {
  const scene = { width: 632, height: 240 }
  const image = placeDiagram('source', scene, settings, { page: 1, x: 17, y: 192.816 })
  assert.equal(image.page, 1)
  assert.equal(image.y, 192.816)
  assert.equal(image.width, 150)
  const bottom = placeDiagram('source', scene, settings, { page: 1, x: 17, y: 270 })
  assert.equal(bottom.y, 270)
  assert.equal(bottom.height, image.height)
  const next = nextDiagramAnchor(bottom, settings)
  assert.equal(next.page, 2)
  assert.equal(next.y, settings.marginTop + image.height + 4)
  assert.equal(bottom.page, 1)
})
test('batches continue after each resolved frame and can use the bottom margin', () => {
  const scene = { width: 150, height: 60 }
  const first = placeDiagram('source', scene, settings, { page: 0, x: 17, y: 220 })
  const anchor = nextDiagramAnchor(first, settings)
  assert.equal(anchor.page, 0)
  assert.equal(anchor.y, 284)
  const second = placeDiagram('second', scene, settings, anchor)
  assert.equal(second.y, 284)
  assert.equal(nextDiagramAnchor(second, settings).page, 1)
})
