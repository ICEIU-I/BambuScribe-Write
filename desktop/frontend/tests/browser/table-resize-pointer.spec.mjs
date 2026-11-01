import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

async function load(page) {
  await open(page)
  await page.evaluate(() => {
    window.__tables.saved = {
      version: 7,
      text: '',
      images: [],
      writing: window.__tables.preferences.writing,
      layout: { ...window.__tables.payload.settings, paperWidth: 210, paperHeight: 297 },
      tables: [
        {
          id: 'pointer-resize',
          page: 0,
          x: 15,
          y: 35,
          columnWidths: [25, 25, 25],
          rowHeights: [12, 12, 12],
          fontSize: 6,
          padding: 1,
          align: 'center',
          verticalAlign: 'middle',
          style: 'grid',
          cells: Array.from({ length: 9 }, (_, i) => ({
            row: Math.floor(i / 3),
            col: i % 3,
            rowSpan: 1,
            colSpan: 1,
            text: i < 3 ? '测量' : String(i),
          })),
        },
      ],
    }
    window.__tables.menu['menu:action']('open')
  })
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.tables?.[0]?.id))
    .toBe('pointer-resize')
}

const grid = (page) => page.locator('[data-table="pointer-resize"]')
const saved = (page) => page.evaluate(() => window.__tables.payload.tables)
async function corner(page) {
  const b = await grid(page).boundingBox()
  return { x: b.x + b.width, y: b.y + b.height }
}
async function expectCorner(page, target) {
  await expect.poll(async () => Math.abs((await corner(page)).x - target.x)).toBeLessThan(1)
  await expect.poll(async () => Math.abs((await corner(page)).y - target.y)).toBeLessThan(1)
}

for (const zoom of [50, 100, 150]) {
  test(`table corner follows the pointer at ${zoom}% and retains its size after relayout`, async ({
    page,
  }) => {
    await load(page)
    await page.getByLabel('页面缩放').selectOption(String(zoom))
    await grid(page).locator('[data-table-cell]').first().click()
    const original = await saved(page)
    const start = await corner(page)
    const target = { x: start.x + 65, y: start.y + 40 }
    await page.mouse.move(start.x, start.y)
    await page.mouse.down()
    await page.mouse.move(target.x, target.y, { steps: 8 })
    await expectCorner(page, target)
    expect(await saved(page)).toEqual(original)
    await page.mouse.up()
    await expect
      .poll(async () => (await saved(page))[0].columnWidths)
      .not.toEqual(original[0].columnWidths)
    await expect
      .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].width))
      .toBeCloseTo(
        await grid(page).evaluate(
          (el) =>
            parseFloat(el.style.width) / (parseFloat(el.closest('[data-paper]').style.width) / 210),
        ),
        2,
      )
    await expectCorner(page, target)
    const resized = await saved(page)
    const resizedBox = await grid(page).boundingBox()
    expect(resized[0].cells).toEqual(original[0].cells)
    await page.getByRole('button', { name: '撤销', exact: true }).click()
    await expect.poll(() => saved(page)).toEqual(original)
    await page.getByRole('button', { name: '重做', exact: true }).click()
    await expect.poll(() => saved(page)).toEqual(resized)
    await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
    await expect.poll(() => page.evaluate(() => window.__tables.saved.tables)).toEqual(resized)
    await page.evaluate(() => window.__tables.menu['menu:action']('open'))
    await expect.poll(() => saved(page)).toEqual(resized)
    await expect
      .poll(async () => (await grid(page).boundingBox()).width)
      .toBeCloseTo(resizedBox.width, 2)
    await expect
      .poll(async () => (await grid(page).boundingBox()).height)
      .toBeCloseTo(resizedBox.height, 2)
    await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
    await expect
      .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
      .toContain('A1 handwriting')
    expect(
      await page.evaluate(() => window.__tables.exported.document.pages[0].tables[0].strokes),
    ).toEqual(await page.evaluate(() => window.__tables.document.pages[0].tables[0].strokes))
  })
}

test('scrolling during resize keeps the corner under the pointer and Escape restores the table', async ({
  page,
}) => {
  await load(page)
  await grid(page).locator('[data-table-cell]').first().click()
  const original = await saved(page)
  const start = await corner(page)
  const target = { x: start.x + 45, y: start.y + 30 }
  await page.mouse.move(start.x, start.y)
  await page.mouse.down()
  await page.mouse.move(target.x, target.y, { steps: 5 })
  await page.locator('.document-viewport').evaluate((el) => {
    el.scrollTop += 60
  })
  await expectCorner(page, target)
  await page.keyboard.press('Escape')
  await page.mouse.up()
  expect(await saved(page)).toEqual(original)
})

test('manual dimensions change independently and the resize handle stops at the paper edge', async ({
  page,
}) => {
  await load(page)
  await page.getByLabel('页面缩放').selectOption('50')
  await grid(page).locator('[data-table-cell]').first().click()
  await page.getByRole('button', { name: '表格设置', exact: true }).click()
  const width = page.getByLabel('设置表格宽度', { exact: true })
  await width.fill('100')
  await width.press('Tab')
  await expect
    .poll(async () => (await saved(page))[0].columnWidths.reduce((a, b) => a + b, 0))
    .toBeCloseTo(100, 6)
  await expect.poll(async () => (await saved(page))[0].rowHeights).toEqual([12, 12, 12])
  const height = page.getByLabel('设置表格高度', { exact: true })
  await height.fill('60')
  await height.press('Tab')
  await expect.poll(async () => (await saved(page))[0].rowHeights).toEqual([20, 20, 20])
  await expect
    .poll(async () => (await saved(page))[0].columnWidths.reduce((a, b) => a + b, 0))
    .toBeCloseTo(100, 6)
  await page.getByLabel('关闭表格设置', { exact: true }).click()
  const start = await corner(page)
  const edge = await page.locator('[data-paper="0"]').evaluate((el) => {
    const r = el.getBoundingClientRect()
    return { x: r.right, y: r.bottom }
  })
  await page.mouse.move(start.x, start.y)
  await page.mouse.down()
  await page.mouse.move(edge.x + 25, edge.y + 25, { steps: 8 })
  await expectCorner(page, edge)
  await page.mouse.up()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].issues))
    .toEqual([])
  await expectCorner(page, edge)
})
