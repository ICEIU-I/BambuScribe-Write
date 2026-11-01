import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { test, open } from './tableHarness.mjs'

test('导入的霍尔公式矢量笔画实际更新，公式间距微调且页面位置不变', async ({ page }, info) => {
  const errors = await open(page)
  const table = JSON.parse(
    readFileSync(
      new URL('../../../internal/engine/testdata/hall-imported-equations.json', import.meta.url),
      'utf8',
    ),
  )
  await page.evaluate((table) => {
    window.__tables.saved = {
      ...window.__tables.preferences,
      layout: {
        ...window.__tables.preferences.layout,
        paperWidth: 210,
        paperHeight: 297,
        handwritingSeed: 2321509729,
      },
      tables: [table],
    }
    window.__tables.menu['menu:action']('open')
  }, table)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.tables?.length))
    .toBe(1)
  const vector = page.locator(`[data-table="${table.id}"] .table-ink path`)
  const object = page.locator(`[data-table="${table.id}"]`)
  await expect(vector).toHaveAttribute('d', /^M/)
  const before = await vector.getAttribute('d')
  const position = await object.boundingBox()
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  await page.getByRole('checkbox', { name: '手写变化', exact: true }).check()
  await page.getByRole('slider', { name: '手写变化强度' }).fill('36')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.settings.handwritingVariation))
    .toBe(36)
  await expect.poll(() => vector.getAttribute('d')).not.toBe(before)
  const path = await vector.getAttribute('d')
  const coordinates = path.match(/-?\d+(?:\.\d+)?(?:e[+-]?\d+)?/gi).map(Number)
  const expected = await page.evaluate(() =>
    window.__tables.document.pages[0].tables[0].strokes.flatMap((s) =>
      s.flatMap((p) => [p.x, p.y]),
    ),
  )
  expect(coordinates).toHaveLength(expected.length)
  coordinates.forEach((n, i) => expect(n).toBeCloseTo(expected[i], 7))
  expect((await object.boundingBox()).width).toBeCloseTo(position.width, 6)
  expect((await object.boundingBox()).height).toBeCloseTo(position.height, 6)
  await page.screenshot({ path: `/tmp/bambuscribe-imported-variation-${info.project.name}.png` })
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect(
    await page.evaluate(() =>
      window.__tables.exported.document.pages[0].tables[0].strokes.flatMap((s) =>
        s.flatMap((p) => [p.x, p.y]),
      ),
    ),
  ).toEqual(expected)
  expect(errors).toEqual([])
})

test('手写变化同步公式及文本框，保存重开和导出保持相同笔画', async ({ page }, info) => {
  const errors = await open(page)
  await page.evaluate(() => {
    window.__tables.saved = {
      ...window.__tables.preferences,
      text: '霍尔效应 888888 242242\n霍尔效应 888888 242242\n霍尔效应 888888 242242\n\\(U_{H}=\\frac{25.3+25.5+25.4+25.4}{4}=25.4mV\\)',
      tables: [
        {
          id: 'formula-textbox',
          kind: 'textbox',
          page: 0,
          x: 10,
          y: 90,
          columnWidths: [140],
          rowHeights: [20],
          fontSize: 5,
          padding: 1,
          align: 'left',
          verticalAlign: 'middle',
          style: 'none',
          cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: 'U̅_{H}=88.8mV 8888' }],
        },
      ],
    }
    window.__tables.menu['menu:action']('open')
  })
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.tables?.length))
    .toBe(1)
  const original = await page.evaluate(() => window.__tables.document.pages[0])
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  const toggle = page.getByRole('checkbox', { name: '手写变化', exact: true })
  await expect(toggle).not.toBeChecked()
  await toggle.check()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.settings.handwritingVariation))
    .toBe(20)
  const initial = await page.evaluate(() => window.__tables.document)
  expect(initial.pages[0].strokes).not.toEqual(original.strokes)
  const lines = [
    ...new Set(initial.pages[0].cells.filter((c) => c.end - c.start === 1).map((c) => c.y)),
  ]
  expect(lines.length).toBeGreaterThanOrEqual(3)
  expect(Math.abs(lines[1] - lines[0] - (lines[2] - lines[1]))).toBeGreaterThan(0.000001)
  expect(initial.pages[0].tables[0].strokes).not.toEqual(original.tables[0].strokes)
  expect(initial.pages[0].formulas.map((f) => f.source)).toEqual(
    original.formulas.map((f) => f.source),
  )
  await page.getByRole('slider', { name: '手写变化强度' }).fill('70')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.settings.handwritingVariation))
    .toBe(70)
  await page.getByRole('button', { name: '换一组', exact: true }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.settings.handwritingSeed))
    .not.toBe(initial.settings.handwritingSeed)
  const changed = await page.evaluate(() => window.__tables.document)
  expect(changed.pages[0].strokes).not.toEqual(initial.pages[0].strokes)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved.layout.handwritingSeed))
    .toBe(changed.settings.handwritingSeed)
  const reopened = page.waitForResponse('**/__table-preview')
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await reopened
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0].strokes))
    .toEqual(changed.pages[0].strokes)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect(await page.evaluate(() => window.__tables.exported.document.pages[0].strokes)).toEqual(
    changed.pages[0].strokes,
  )
  await toggle.scrollIntoViewIfNeeded()
  await page.screenshot({ path: `/tmp/bambuscribe-variation-ui-${info.project.name}.png` })
  await toggle.uncheck()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.settings.handwritingVariation ?? 0))
    .toBe(0)
  expect(await page.evaluate(() => window.__tables.document.pages[0].strokes)).toEqual(
    original.strokes,
  )
  await toggle.check()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0].strokes))
    .toEqual(changed.pages[0].strokes)
  expect(errors).toEqual([])
})
