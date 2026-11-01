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
    source = source.replace(`'${name}'`, JSON.stringify(url))
  return `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
}
const { insertDocumentPage } = await import(
  await moduleURL('insertDocumentPage', {
    './objectLayout': await moduleURL('objectLayout'),
    './inkCutHistory': await moduleURL('inkCutHistory'),
    './documentPages': await moduleURL('documentPages'),
  })
)
const copyDocumentPage = (...args) => insertDocumentPage(...args, 'copy')
const contents = { text: '续文', images: [], tables: [], erasures: [] }
const document = {
  settings: {
    paperHeight: 240,
    marginTop: 0,
    marginBottom: 0,
    lineSpacing: 100,
    pageLayouts: [{ lineCount: 4, continuation: true }],
  },
  pages: [{ text: contents.text, lineHeight: 60 }],
}

test('copying a page remaps erasure targets to copied objects without touching the original', () => {
  const t = { id: 'grid', page: 0, x: 10, y: 20, columnWidths: [30], rowHeights: [10], cells: [] }
  const e = { id: 'erase', page: 0, x: 10, y: 20, width: 30, height: 10, objectIds: ['grid'] }
  const next = copyDocumentPage({ ...contents, tables: [t], erasures: [e] }, document, 0)
  assert.deepEqual(next.erasures[0], e)
  assert.deepEqual(next.erasures[1].objectIds, [next.tables[1].id])
  assert.notEqual(next.tables[1].id, 'grid')
  assert.deepEqual(e.objectIds, ['grid'])
})

test('copying a remaining continuation page preserves its unindented first line and source data', () => {
  const next = copyDocumentPage(contents, document, 0)
  assert.equal(next.text, '续文\f续文')
  assert.deepEqual(next.pageLayouts, [
    document.settings.pageLayouts[0],
    document.settings.pageLayouts[0],
  ])
  assert.equal(contents.text, '续文')
  assert.equal(copyDocumentPage({ ...contents, text: '尚未排版' }, document, 0), null)
})

test('copy rejects object and text limits before changing the original document', () => {
  const tables = Array.from({ length: 26 }, (_, i) => ({ id: `t${i}`, page: 0, cells: [] }))
  const source = { ...contents, tables }
  assert.throws(() => copyDocumentPage(source, document, 0), /50/)
  assert.equal(tables.length, 26)
  const text = '字'.repeat(10001)
  assert.throws(
    () =>
      copyDocumentPage(
        { ...contents, text },
        { ...document, pages: [{ text, lineHeight: 60 }] },
        0,
      ),
    /20,000/,
  )
})
