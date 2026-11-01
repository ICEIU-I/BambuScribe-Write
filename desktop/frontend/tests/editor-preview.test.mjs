import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'

async function moduleURL(name, imports = {}) {
  let source = ts.transpileModule(
    await readFile(new URL(`../src/services/${name}.ts`, import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
  ).outputText
  for (const [from, to] of Object.entries(imports))
    source = source.replace(`'${from}'`, JSON.stringify(to))
  return `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
}
const { unpackEditorPreview, previewEditor } = await import(
  await moduleURL('editorPreview', { './bridge': await moduleURL('bridge') })
)

test('editor restores full precision and shares repeated body, object and erasure vectors', () => {
  const coordinates = [1.2345678901234567, 2, 3, 4.987654321098765]
  const doc = unpackEditorPreview({
    strokes: [coordinates],
    document: {
      pages: [
        {
          strokes: [0],
          textStrokes: [0],
          ink: [{ strokes: [0], part: '0' }],
          images: [
            {
              sourceStrokes: [0],
              graphicStrokes: [0],
              textRegions: [{ strokes: [0], sourceStrokes: [0] }],
            },
          ],
          tables: [{ strokes: [0], cells: [{ strokes: [0] }] }],
          issues: ['行距过小'],
          warnings: ['识别提示'],
        },
        { strokes: [], textStrokes: null, images: [] },
      ],
      erasures: [{ cuts: [{ stroke: 0, ranges: [[0, 1]] }] }],
    },
  })
  const p = doc.pages[0]
  const stroke = p.strokes[0]
  assert.deepEqual(stroke, [
    { x: coordinates[0], y: coordinates[1] },
    { x: coordinates[2], y: coordinates[3] },
  ])
  for (const list of [
    p.textStrokes,
    p.ink[0].strokes,
    p.images[0].sourceStrokes,
    p.images[0].graphicStrokes,
    p.images[0].textRegions[0].strokes,
    p.images[0].textRegions[0].sourceStrokes,
    p.tables[0].strokes,
    p.tables[0].cells[0].strokes,
  ])
    assert.equal(list[0], stroke)
  assert.deepEqual(doc.pages[1], { strokes: [], textStrokes: null, images: [] })
  assert.deepEqual(doc.erasures, [{ cuts: [{ stroke: 0, ranges: [[0, 1]] }] }])
  assert.deepEqual(p.issues, ['行距过小'])
  assert.deepEqual(p.warnings, ['识别提示'])
})

test('compact preview uses one native call and older bridges retain their existing path', async () => {
  const previous = globalThis.window
  try {
    const calls = []
    globalThis.window = {
      go: {
        application: {
          App: {
            PreviewEditor: async (...args) => {
              calls.push(args)
              return { document: { pages: [{ strokes: [0] }] }, strokes: [[1, 2, 3, 4]] }
            },
          },
        },
      },
    }
    assert.deepEqual((await previewEditor('正文', {}, [], [], [])).pages[0].strokes, [
      [
        { x: 1, y: 2 },
        { x: 3, y: 4 },
      ],
    ])
    assert.deepEqual(calls, [['正文', {}, [], [], []]])
    const api = window.go.application.App
    delete api.PreviewEditor
    for (const name of ['PreviewWithImages', 'PreviewWithObjects', 'PreviewWithErasures'])
      api[name] = async (...args) => ({ name, args })
    assert.equal((await previewEditor('正文', {}, [], [], [])).name, 'PreviewWithImages')
    assert.equal((await previewEditor('正文', {}, [], [{}], [])).name, 'PreviewWithObjects')
    assert.equal((await previewEditor('正文', {}, [], [], [{}])).name, 'PreviewWithErasures')
  } finally {
    globalThis.window = previous
  }
})
