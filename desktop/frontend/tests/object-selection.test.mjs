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
const { selectionAt, onBodyText, imageBounds } = await import(await moduleURL('objectSelection'))
const { copyObjects, writeObjects, readObjects, pasteObjects, objectGroupMime } = await import(
  await moduleURL('objectGroupClipboard')
)
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
  settings: { paperWidth: 163, paperHeight: 240, marginTop: 5, marginBottom: 10, fontSize: 6 },
  pages: [
    {
      lineHeight: 15,
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
const copy = () => copyObjects(doc, images, tables, select())
const data = () => {
  const values = new Map()
  return { values, setData: (k, v) => values.set(k, v), getData: (k) => values.get(k) ?? '' }
}
test('marquee includes complete images, merged tables and captions; body and other pages stay out', () => {
  assert.deepEqual(
    select().objects.map((o) => o.id),
    ['diagram', 'table', 'caption'],
  )
  const partial = selectionAt(doc, images, tables, 0, { x: 104, y: 59 }, { x: 106, y: 61 })
  assert.deepEqual(
    partial.objects.map((o) => o.id),
    ['table'],
  ) // actual grown height
  assert.equal(
    selectionAt(doc, images, tables, 1, { x: 0, y: 0 }, { x: 150, y: 100 }).objects.length,
    0,
  )
  const reverse = selectionAt(doc, images, tables, 0, { x: 150, y: 100 }, { x: 0, y: 0 })
  assert.deepEqual(reverse, select())
  assert.equal(onBodyText(doc, '字 ', 0, { x: 2, y: 2 }), true)
  assert.equal(onBodyText(doc, '字 ', 0, { x: 8, y: 2 }), true)
  assert.equal(onBodyText(doc, '字 ', 0, { x: 13, y: 2 }), false)
})
test('body hit area follows each row, includes spaces and stops one character past its visible end', () => {
  const text = '甲 乙。    \n丙\n   ',
    cells = [...text]
      .map((char, start) => ({
        start,
        end: start + 1,
        x: start < 9 ? 10 + start * 6 : start === 9 ? 10 : 10 + (start - 11) * 6,
        y: start < 9 ? 15 : start === 9 ? 30 : 60,
        width: char === '。' ? 2 : 6,
        height: 6,
      }))
      .filter((c) => text[c.start] !== '\n'),
    d = { ...doc, pages: [{ ...doc.pages[0], cells }] },
    hit = (x, y) => onBodyText(d, text, 0, { x, y })
  assert.equal(hit(19, 18), true) // internal space
  assert.equal(hit(35, 18), true) // after compact final punctuation
  assert.equal(hit(37, 18), false) // remainder of the half-empty row
  assert.equal(hit(35, 11), true) // upper half of the interline gap
  assert.equal(hit(35, 25), true) // lower half, still nearest to this row
  assert.equal(hit(35, 27), false) // next row is shorter
  assert.equal(hit(21, 33), true)
  assert.equal(hit(23, 33), false)
  assert.equal(hit(12, 48), false) // empty/object-only row
  assert.equal(hit(12, 63), false) // whitespace-only row
  assert.equal(hit(9, 18), false) // left paper blank
  assert.equal(onBodyText(d, text, 1, { x: 12, y: 18 }), false)
})
test('body hit padding follows font size and respects tightly spaced rows and paper edges', () => {
  for (const fontSize of [3, 9]) {
    const height = fontSize * 0.85,
      d = {
        ...doc,
        settings: { ...doc.settings, fontSize },
        pages: [
          {
            ...doc.pages[0],
            lineHeight: height,
            cells: [{ start: 0, end: 1, x: 155, y: 240 - height, width: 2, height }],
          },
        ],
      }
    assert.equal(onBodyText(d, '。', 0, { x: Math.min(163, 157 + fontSize), y: 239 }), true)
    assert.equal(onBodyText(d, '。', 0, { x: 157 + fontSize + 0.1, y: 239 }), false)
    assert.equal(onBodyText(d, '。', 0, { x: 156, y: 240 - height - 0.1 }), false)
    assert.equal(onBodyText(d, '。', 0, { x: 156, y: 240 }), false)
  }
})
test('copy preserves visible resolved positions, OCR refresh, manual erasure and editable merged cells', () => {
  const resolved = {
    ...doc,
    pages: [
      doc.pages[0],
      {
        ...doc.pages[0],
        images: [
          {
            id: image.id,
            textRegions: [
              {
                id: 'ocr',
                source: 'ocr',
                ocrVersion: 'current',
                text: '新识别',
                x: 0.1,
                y: 0.1,
                width: 0.2,
                height: 0.2,
              },
              image.textRegions[0],
            ],
          },
        ],
      },
    ],
    objectLayout: [
      {
        id: image.id,
        sourcePage: 0,
        source: { x: 20, y: 30, width: 40, height: 20 },
        rotation: 15,
        page: 1,
        x: 30,
        y: 40,
      },
    ],
  }
  const selection = selectionAt(resolved, images, tables, 1, { x: 0, y: 0 }, { x: 150, y: 100 })
  const copied = copyObjects(resolved, images, tables, selection)
  assert.equal(copied.images[0].page, 1)
  assert.equal(copied.images[0].x, 30)
  assert.equal(copied.images[0].textRegions[0].text, '新识别')
  copied.images[0].textRegions[1].erasePath[0].x = 0.9
  assert.equal(image.textRegions[0].erasePath[0].x, 0.2)
  const all = copy()
  assert.equal(all.tables[0].cells[0].colSpan, 2)
  all.tables[0].cells[0].text = '副本'
  assert.equal(table.cells[0].text, '合并格')
  assert.equal(all.tables[1].kind, 'textbox')
  assert.equal(Object.hasOwn(all, 'text'), false)
})
test('custom and HTML clipboard round trip; native fallback requires the exact token', () => {
  const copied = copy(),
    transfer = data()
  writeObjects(transfer, copied)
  assert.deepEqual(readObjects(transfer), copied)
  transfer.values.delete(objectGroupMime)
  assert.deepEqual(readObjects(transfer), copied)
  transfer.values.delete('text/html')
  assert.equal(readObjects(transfer, copied), copied)
  assert.equal(readObjects(transfer), undefined)
  transfer.setData('text/plain', '普通文字')
  assert.equal(readObjects(transfer, copied), undefined)
  transfer.setData('text/plain', copy().token)
  assert.equal(readObjects(transfer, copied), undefined)
})
test('invalid group geometry, duplicate IDs, mixed pages and partial table data are rejected', () => {
  const copied = copy()
  for (const invalid of [
    { ...copied, bounds: { ...copied.bounds, width: 1 } },
    { ...copied, tables: [table, { ...box, id: table.id }] },
    { ...copied, tables: [table, { ...box, page: 1 }] },
    { ...copied, tables: [{ ...table, cells: [] }] },
    { ...copied, images: [{ ...image, width: -1 }] },
  ]) {
    const transfer = data()
    transfer.setData(objectGroupMime, JSON.stringify(invalid))
    assert.equal(readObjects(transfer, copied), undefined)
  }
})
test('paste allows overlap at the chosen location without changing relative positions', () => {
  const copied = copy()
  const existing = { ...box, id: 'block', x: 0, y: 0, columnWidths: [163], rowHeights: [180] }
  const next = pasteObjects(copied, doc, [], [existing], { page: 0, x: 20, y: 10 })
  assert.equal(next.page, 0)
  assert.equal(next.y, 10)
  assert.equal(next.tables[0].x - next.images[0].x, table.x - image.x)
  assert.equal(next.tables[1].y - next.images[0].y, box.y - image.y)
  assert.equal(new Set([...next.images, ...next.tables].map((o) => o.id)).size, 3)
  assert.notEqual(next.images[0].id, image.id)
  next.tables[0].cells[0].text = '副本修改'
  assert.equal(copied.tables[0].cells[0].text, table.cells[0].text)
})
test('clicked X, page-only paste and repeated paste retain the original image, table and caption columns', () => {
  const copied = copy()
  for (const position of [{ page: 1 }, { page: 1, x: 0, y: 100 }, { page: 1, x: 155, y: 100 }]) {
    const next = pasteObjects(copied, doc, [], [], position)
    assert.equal(next.images[0].x, image.x)
    assert.deepEqual(
      next.tables.map((t) => t.x),
      tables.map((t) => t.x),
    )
    const repeated = pasteObjects(copied, doc, next.images, next.tables, {
      page: next.page,
      x: next.x + 2,
      y: next.y + 2,
    })
    assert.equal(repeated.images[0].x, image.x)
    assert.deepEqual(
      repeated.tables.map((t) => t.x),
      tables.map((t) => t.x),
    )
  }
})
test('narrower destination paper translates the whole group only as far as needed to fit', () => {
  const copied = copy(),
    narrow = { ...doc, settings: { ...doc.settings, paperWidth: 95 } }
  const next = pasteObjects(copied, narrow, [], [], { page: 1, x: 0, y: 100 })
  const dx = 95 - copied.bounds.width - copied.bounds.x
  assert.equal(next.images[0].x, image.x + dx)
  assert.deepEqual(
    next.tables.map((t) => t.x),
    tables.map((t) => t.x + dx),
  )
  assert.ok(Math.abs(next.tables[0].x + 40 - 95) < 1e-8)
})
test('edge paste fits both a rotated image and its unrotated frame so the group remains together', () => {
  const rotated = { ...image, rotation: 90, width: 70, height: 10 }
  const selection = selectionAt(doc, [rotated], [], 0, { x: 0, y: 0 }, { x: 160, y: 200 })
  const copied = copyObjects(doc, [rotated], [], selection)
  const next = pasteObjects(copied, doc, [], [], { page: 0, x: 160, y: 220 })
  const im = next.images[0],
    ink = imageBounds(im)
  assert.equal(next.page, 1)
  for (const r of [im, ink]) {
    assert.ok(r.x >= -1e-8 && r.x + r.width <= 163 + 1e-8)
    assert.ok(r.y >= -1e-8 && r.y + r.height <= 230 + 1e-8)
  }
})
test('oversized groups and object limits fail without mutating the destination', () => {
  const copied = copy(),
    before = JSON.stringify({ images, tables })
  assert.throws(
    () => pasteObjects(copied, doc, images, Array(50).fill(box), { page: 0 }),
    /最多 50/,
  )
  assert.throws(
    () =>
      pasteObjects(
        copied,
        { ...doc, settings: { ...doc.settings, paperWidth: 10 } },
        images,
        tables,
        { page: 0 },
      ),
    /超过纸张/,
  )
  assert.equal(JSON.stringify({ images, tables }), before)
})
