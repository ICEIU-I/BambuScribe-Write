import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'
async function url(name, replacements = {}) {
  let s = ts.transpileModule(
    await readFile(new URL(`../src/state/${name}.ts`, import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
  ).outputText
  for (const [a, b] of Object.entries(replacements)) s = s.replace(`'${a}'`, JSON.stringify(b))
  return `data:text/javascript;base64,${Buffer.from(s).toString('base64')}`
}
const { captureInk, cutInk, validInkCuts } = await import(await url('inkCuts'))
const { rebaseBodyCuts, remapPageCuts, resetChangedObjectCuts } = await import(
  await url('inkCutHistory')
)
const { insertDocumentPage } = await import(
  await url('insertDocumentPage', {
    './inkCutHistory': await url('inkCutHistory'),
    './objectLayout': await url('objectLayout'),
    './documentPages': await url('documentPages'),
  })
)
const layer = {
  objectId: 'box',
  part: 'cell:0:0',
  source: 'same-text',
  strokes: [
    [
      { x: 0, y: 5 },
      { x: 20, y: 5 },
    ],
  ],
}
const e = { id: 'e', page: 0, x: 5, y: 0, width: 5, height: 10 }
test('rectangle and free gestures delete the same vertex fractions after move and resize, not the old paper area', () => {
  for (const brush of [
    e,
    {
      ...e,
      path: [
        { x: 7.5, y: 0 },
        { x: 7.5, y: 10 },
      ],
      radius: 2.5,
    },
  ]) {
    const frozen = { ...brush, cuts: captureInk(brush, [layer]) }
    const moved = [
      [
        { x: 100, y: 100 },
        { x: 140, y: 100 },
      ],
    ]
    assert.deepEqual(cutInk(moved, layer, [frozen]), [
      [
        { x: 100, y: 100 },
        { x: 110, y: 100 },
      ],
      [
        { x: 120, y: 100 },
        { x: 140, y: 100 },
      ],
    ])
    assert.deepEqual(cutInk(moved, { ...layer, source: 'new-text' }, [frozen]), moved)
    assert.deepEqual(cutInk(moved, { ...layer, objectId: 'later-box' }, [frozen]), moved)
    assert.ok(validInkCuts(frozen.cuts))
    assert.equal(validInkCuts([{ ...frozen.cuts[0], ranges: [[0, NaN]] }]), false)
  }
})
test('new正文 and repeated edits never inherit old cuts; untouched UTF-16 source offsets shift', () => {
  const cut = { objectId: '', part: '2', source: '字', stroke: 0, ranges: [[0, 1]] }
  const old = [{ ...e, cuts: [cut] }]
  const shifted = rebaseBodyCuts(old, '甲乙丙', '新增甲乙丙')
  assert.equal(shifted[0].cuts[0].part, '4')
  assert.equal(old[0].cuts[0].part, '2')
  assert.deepEqual(rebaseBodyCuts(shifted, '新增甲乙丙', '新文字')[0].cuts, [])
  assert.equal(rebaseBodyCuts(old, '甲乙丙', '𠮷甲乙丙')[0].cuts[0].part, '4')
})
test('page copy remaps erased object/region identities and body offsets; page reorder retains original characters', () => {
  const image = {
    id: 'im',
    page: 0,
    x: 0,
    y: 0,
    width: 10,
    height: 10,
    textRegions: [{ id: 'r', text: '字' }],
  }
  const object = { objectId: 'im', part: 'region:r', source: 's', stroke: 0, ranges: [[0, 1]] }
  const body = { ...object, objectId: '', part: '0' }
  const contents = {
    text: '甲\f乙',
    images: [image],
    tables: [],
    erasures: [{ ...e, cuts: [object, body] }],
  }
  const doc = {
    settings: { paperHeight: 100, marginTop: 0, marginBottom: 0, lineSpacing: 100 },
    pages: [
      { text: '甲\f', lineHeight: 10, images: [image] },
      { text: '乙', lineHeight: 10 },
    ],
  }
  const next = insertDocumentPage(contents, doc, 0, 'copy')
  assert.equal(next.text, '甲\f甲\f乙')
  assert.equal(next.erasures[1].cuts[0].objectId, next.images[1].id)
  assert.equal(next.erasures[1].cuts[0].part, 'region:' + next.images[1].textRegions[0].id)
  assert.equal(next.erasures[1].cuts[1].part, '2')
  assert.equal(remapPageCuts(contents.erasures, doc, [1, 0], ['乙', '甲'])[0].cuts[1].part, '2')
})
test('rewriting a cell consumes its old cuts, so retyping the former text is also complete; movement retains cuts', () => {
  const old = {
    id: 'box',
    style: 'none',
    columnWidths: [30],
    rowHeights: [10],
    cells: [{ row: 0, col: 0, text: '旧字' }],
  }
  const erased = [{ ...e, cuts: captureInk(e, [layer]) }]
  assert.equal(resetChangedObjectCuts(erased, [old], [{ ...old, x: 50 }], [], []), erased)
  assert.deepEqual(
    resetChangedObjectCuts(
      erased,
      [old],
      [{ ...old, cells: [{ ...old.cells[0], text: '新字' }] }],
      [],
      [],
    )[0].cuts,
    [],
  )
})
