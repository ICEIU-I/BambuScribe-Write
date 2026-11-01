import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

test('legacy overflowing caption scales both ways with its frame and survives cancel, undo and export', async ({
  page,
}) => {
  const errors = await open(page)
  await page.evaluate(() => {
    window.__tables.saved = {
      ...window.__tables.preferences,
      tables: [
        {
          id: 'caption',
          kind: 'textbox',
          page: 0,
          x: 20,
          y: 40,
          columnWidths: [70],
          rowHeights: [13.31],
          fontSize: 8.206,
          padding: 1,
          align: 'left',
          verticalAlign: 'middle',
          style: 'none',
          cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '图一 霍尔效应原理图' }],
        },
      ],
    }
    window.__tables.menu['menu:action']('open')
  })
  const caption = page.locator('[data-free-text-box="caption"]')
  await expect(caption).toHaveCount(1)
  const rendered = () => page.evaluate(() => window.__tables.document.pages[0].tables[0])
  const source = () => page.evaluate(() => window.__tables.payload.tables[0])
  await expect.poll(async () => (await rendered()).issues).toEqual([])
  const first = await rendered(),
    original = await source()
  expect(first.fontSize).toBeLessThan(original.fontSize)
  await caption.click()
  await expect(page.getByLabel('文本框字号')).toHaveValue(first.fontSize.toFixed(2))
  await expect(page.getByLabel('文本框字号')).toHaveAttribute('readonly', '')
  async function resize(factor, cancel = false) {
    await caption.click()
    const before = await source()
    const handle = await page.getByRole('button', { name: '调整文本框大小' }).boundingBox()
    const k = await page
      .locator('[data-paper="0"]')
      .evaluate((el) => el.getBoundingClientRect().width / 163)
    const oldPath = await caption.locator('path').getAttribute('d')
    await page.mouse.move(handle.x + handle.width / 2, handle.y + handle.height / 2)
    await page.mouse.down()
    await page.mouse.move(
      handle.x + handle.width / 2 + before.columnWidths[0] * (factor - 1) * k,
      handle.y + handle.height / 2 + before.rowHeights[0] * (factor - 1) * k,
      { steps: 6 },
    )
    expect(await caption.locator('path').getAttribute('d')).not.toBe(oldPath)
    expect(await source()).toEqual(before)
    if (cancel) await page.keyboard.press('Escape')
    await page.mouse.up()
  }
  await resize(0.5, true)
  expect(await source()).toEqual(original)
  await resize(0.5)
  await expect.poll(async () => (await rendered()).fontSize).toBeLessThan(first.fontSize * 0.6)
  const small = await source()
  await resize(2)
  await expect
    .poll(async () => Math.abs((await rendered()).fontSize - first.fontSize))
    .toBeLessThan(0.1)
  const large = await source()
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(source).toEqual(small)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect.poll(source).toEqual(large)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.tables[0])).toEqual(large)
  const reopened = page.waitForResponse('**/__table-preview')
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await reopened
  await expect.poll(source).toEqual(large)
  // A response arriving does not mean the editor has finished applying it.
  // Use the enabled export action so the normal layout guard has cleared.
  await page.getByRole('button', { name: '导出', exact: true }).click()
  await page.getByRole('button', { name: /导出当前页/ }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  const final = await rendered()
  for (const stroke of final.strokes)
    for (const p of stroke) {
      expect(p.x).toBeGreaterThanOrEqual(final.padding - 1e-7)
      expect(p.x).toBeLessThanOrEqual(final.width - final.padding + 1e-7)
      expect(p.y).toBeGreaterThanOrEqual(final.padding - 1e-7)
      expect(p.y).toBeLessThanOrEqual(final.height - final.padding + 1e-7)
    }
  expect(errors).toEqual([])
})
