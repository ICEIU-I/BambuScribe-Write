import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { test, open } from './tableHarness.mjs'
const fringe = JSON.parse(
  readFileSync(
    new URL('../../../internal/engine/testdata/fringe-table.json', import.meta.url),
    'utf8',
  ),
)

test('existing fringe table retains its saved size, caps letters and exports the displayed grid', async ({
  page,
}) => {
  const errors = await open(page)
  await page.evaluate((table) => {
    window.__tables.saved = {
      version: 4,
      text: '正文',
      images: [],
      tables: [table],
      writing: window.__tables.preferences.writing,
      layout: {
        ...window.__tables.payload.settings,
        fontSize: 8.206666666666667,
        paperWidth: 210,
        paperHeight: 297,
      },
    }
    window.__tables.menu['menu:action']('open')
  }, fringe)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.tables?.[0]?.id))
    .toBe('fringe')
  const read = () =>
    page.evaluate(() => {
      const t = window.__tables.document.pages[0].tables[0]
      const expected = t.strokes.flat().flatMap((p) => [p.x, p.y])
      const shown = (
        document
          .querySelector('[data-table="fringe"] .table-ink path')
          .getAttribute('d')
          .match(/-?\d+(?:\.\d+)?(?:e[+-]?\d+)?/g) ?? []
      ).map(Number)
      return {
        body: window.__tables.document.settings.fontSize,
        width: t.width,
        height: t.height,
        padding: t.padding,
        cells: t.cells.filter((c) => c.text).map((c) => c.fontSize),
        error:
          expected.length === shown.length
            ? Math.max(...expected.map((v, i) => Math.abs(v - shown[i])))
            : Infinity,
      }
    })
  await expect.poll(async () => (await read()).error).toBeLessThan(1e-7)
  const initial = await read()
  expect(initial.width).toBeCloseTo(
    fringe.columnWidths.reduce((a, b) => a + b, 0),
    7,
  )
  expect(initial.height).toBeCloseTo(
    fringe.rowHeights.reduce((a, b) => a + b, 0),
    7,
  )
  for (const size of initial.cells) {
    expect(size).toBeLessThanOrEqual(initial.body + 1e-7)
    expect(size).toBeGreaterThanOrEqual(1)
  }
  const grid = page.locator('[data-table="fringe"]')
  const width = await grid.evaluate((el) => el.getBoundingClientRect().width)
  const paperWidth = await page
    .locator('[data-paper="0"]')
    .evaluate((el) => el.getBoundingClientRect().width)
  expect((width / paperWidth) * 210).toBeCloseTo(initial.width, 2)
  await page.locator('[data-table-cell="3:0"]').dblclick()
  await expect(page.getByLabel('单元格文字', { exact: true })).toHaveValue('条纹位置x_{i+n}/mm')
  await page.getByLabel('单元格文字', { exact: true }).press('Enter')
  await expect.poll(async () => (await read()).error).toBeLessThan(1e-7)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  expect(
    await page.evaluate(() => window.__tables.exported.document.pages[0].tables[0].strokes),
  ).toEqual(await page.evaluate(() => window.__tables.document.pages[0].tables[0].strokes))
  await page.screenshot({
    path: `/tmp/bambuscribe-body-capped-table-${test.info().project.name}.png`,
  })
  expect(errors).toEqual([])
})
