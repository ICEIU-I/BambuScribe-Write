import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'

const ink = ts.transpileModule(
  await readFile(new URL('../src/state/inkCuts.ts', import.meta.url), 'utf8'),
  { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
).outputText
const inkURL = `data:text/javascript;base64,${Buffer.from(ink).toString('base64')}`
const source = ts
  .transpileModule(
    await readFile(new URL('../src/state/erasureTargets.ts', import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
  )
  .outputText.replace("'./inkCuts'", JSON.stringify(inkURL))
const { erasureTouches, targetErasure } = await import(
  `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
)

test('rectangle targets crossing strokes and single dots, but no outside or empty layers', () => {
  const e = { id: 'cut', page: 0, x: 10, y: 10, width: 10, height: 10 }
  assert.ok(
    erasureTouches(e, [
      [
        { x: 0, y: 15 },
        { x: 30, y: 15 },
      ],
    ]),
  )
  assert.ok(erasureTouches(e, [[{ x: 15, y: 15 }]]))
  assert.equal(
    erasureTouches(e, [
      [
        { x: 11, y: 1 },
        { x: 21, y: 9 },
      ],
    ]),
    false,
  )
  const page = {
    images: [
      {
        id: 'image',
        strokes: [
          [
            { x: 0, y: 15 },
            { x: 30, y: 15 },
          ],
        ],
      },
    ],
    tables: [
      {
        id: 'caption',
        x: 100,
        y: 100,
        strokes: [
          [
            { x: 0, y: 0 },
            { x: 5, y: 5 },
          ],
        ],
      },
    ],
  }
  assert.deepEqual(targetErasure(e, page).objectIds, ['image'])
  assert.deepEqual(targetErasure({ ...e, objectIds: [] }, page).objectIds, [])
})
test('free erasure uses capsules and does not target objects inside the path bounding box', () => {
  const e = {
    id: 'brush',
    page: 0,
    x: 10,
    y: 10,
    width: 20,
    height: 20,
    path: [
      { x: 10, y: 10 },
      { x: 10, y: 30 },
      { x: 30, y: 30 },
    ],
    radius: 2,
  }
  assert.equal(
    erasureTouches(e, [
      [
        { x: 20, y: 20 },
        { x: 25, y: 20 },
      ],
    ]),
    false,
  )
  assert.ok(
    erasureTouches(e, [
      [
        { x: 0, y: 20 },
        { x: 15, y: 20 },
      ],
    ]),
  )
  assert.ok(erasureTouches(e, [[{ x: 31, y: 31 }]]))
  assert.equal(erasureTouches(e, [[{ x: 33, y: 33 }]]), false)
})
