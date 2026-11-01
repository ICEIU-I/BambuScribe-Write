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
const { deleteDocumentPage } = await import(
  await moduleURL('deleteDocumentPage', {
    './objectLayout': await moduleURL('objectLayout'),
    './inkCutHistory': await moduleURL('inkCutHistory'),
  })
)
const contents = (text) => ({ text, images: [], tables: [], erasures: [] })
const doc = (...texts) => ({ pages: texts.map((text) => ({ text, carets: [{}] })) })

test('delete the selected source range, preserving duplicate text, Unicode and CRLF outside it', () => {
  const before = contents('𠮷甲\r\n重复\r\n重复\r\n尾字')
  const next = deleteDocumentPage(before, doc('𠮷甲\n重复\n', '重复\n', '尾字'), 1)
  assert.equal(next.text, '𠮷甲\r\n重复\r\n尾字')
  assert.equal(next.start, '𠮷甲\r\n重复\r\n'.length)
  assert.equal(before.text, '𠮷甲\r\n重复\r\n重复\r\n尾字')
})
test('remove the final page separator while preserving a separator before surviving later text', () => {
  assert.equal(deleteDocumentPage(contents('前\n后'), doc('前\n', '后'), 1).text, '前')
  assert.equal(deleteDocumentPage(contents('前\n'), doc('前\n', ''), 1).text, '前')
  assert.equal(
    deleteDocumentPage(contents('前\n中\n后'), doc('前\n', '中\n', '后'), 1).text,
    '前\n后',
  )
  assert.equal(deleteDocumentPage(contents('全文'), doc('全文'), 0).text, '')
})
test('delete objects by their visible page and move later objects and erasures back one page', () => {
  const image = { id: 'moved', page: 0, x: 0, y: 220, width: 20, height: 20, rotation: 0 }
  const later = { ...image, id: 'later', page: 3, y: 20 }
  const before = {
    ...contents(''),
    images: [image, later],
    tables: [{ id: 'table', page: 1, x: 5, y: 5, columnWidths: [10], rowHeights: [10] }],
    erasures: [
      { id: 'deleted', page: 1 },
      { id: 'kept', page: 3 },
    ],
  }
  const document = {
    ...doc('', '', '', ''),
    objectLayout: [
      {
        id: 'moved',
        page: 1,
        x: 0,
        y: 10,
        sourcePage: 0,
        source: { x: 0, y: 220, width: 20, height: 20 },
        rotation: 0,
      },
    ],
  }
  const next = deleteDocumentPage(before, document, 1)
  assert.deepEqual(next.images, [{ ...later, page: 2 }])
  assert.deepEqual(next.tables, [])
  assert.deepEqual(next.erasures, [{ id: 'kept', page: 2 }])
  assert.equal(before.images[1].page, 3)
})
test('an object-only trailing page does not consume body text; invalid or stale pages do nothing', () => {
  const document = doc('正文\n', '')
  document.pages[1].carets = []
  assert.equal(deleteDocumentPage(contents('正文\n'), document, 1).text, '正文\n')
  assert.equal(deleteDocumentPage(contents('新正文'), doc('旧正文'), 0), null)
  assert.equal(deleteDocumentPage(contents('正文'), doc('正文'), -1), null)
  assert.equal(deleteDocumentPage(contents('正文'), doc('正文'), 1), null)
})
