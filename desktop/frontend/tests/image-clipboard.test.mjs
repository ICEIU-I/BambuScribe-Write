import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'

const source = ts.transpileModule(
  await readFile(new URL('../src/state/imageClipboard.ts', import.meta.url), 'utf8'),
  { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
).outputText
const { copyImage, writeImage, readImage, pasteImage } = await import(
  `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
)
const original = {
  id: 'diagram',
  page: 2,
  data: 'data:image/png;base64,AAAA',
  x: 20,
  y: 10,
  width: 70,
  height: 40,
  rotation: 15,
  mode: 'line',
  textRegions: [
    {
      id: 'erase',
      mode: 'erase',
      source: 'manual',
      text: '',
      x: 0.1,
      y: 0.1,
      width: 0.2,
      height: 0.2,
      erasePath: [{ x: 0.1, y: 0.1 }],
      eraseRadius: 0.02,
    },
    {
      id: 'label',
      mode: 'handwriting',
      source: 'manual',
      text: '光源',
      x: 0.5,
      y: 0.5,
      width: 0.2,
      height: 0.1,
    },
  ],
}
test('image copies preserve data and corrections without sharing objects; paste fits the destination page', () => {
  const copied = copyImage(original)
  const pasted = pasteImage(copied, { paperWidth: 50, paperHeight: 60 }, { page: 1, x: 100, y: -1 })
  assert.equal(pasted.page, 1)
  assert.notEqual(pasted.id, original.id)
  assert.equal(pasted.width, 50)
  assert.equal(pasted.height, (40 * 50) / 70)
  assert.equal(pasted.x, 0)
  assert.equal(pasted.y, 0)
  assert.equal(pasted.rotation, original.rotation)
  assert.equal(pasted.data, original.data)
  assert.deepEqual(pasted.textRegions, original.textRegions)
  pasted.textRegions[0].erasePath[0].x = 0.9
  pasted.textRegions[1].text = '副本'
  assert.deepEqual(copied.image.textRegions, original.textRegions)
  assert.notEqual(
    pasteImage(copied, { paperWidth: 163, paperHeight: 240 }, { page: 1 }).id,
    pasted.id,
  )
})
test('custom image clipboard survives serialization and WebView fallback requires the exact copy token', () => {
  const values = new Map(),
    copied = copyImage(original)
  const data = {
    setData: (key, value) => values.set(key, value),
    getData: (key) => values.get(key) ?? '',
  }
  writeImage(data, copied)
  assert.deepEqual(readImage(data), copied)
  assert.notEqual(readImage(data), copied)
  values.delete('application/x-bambuscribe-image')
  assert.equal(readImage(data, copied), copied)
  assert.equal(readImage(data), undefined)
  values.set('text/plain', '其他内容')
  assert.equal(readImage(data, copied), undefined)
  values.set('text/plain', copyImage(original).token)
  assert.equal(readImage(data, copied), undefined)
  values.set(
    'application/x-bambuscribe-image',
    JSON.stringify({ ...copied, image: { ...original, width: -1 } }),
  )
  assert.equal(readImage(data), undefined)
})
