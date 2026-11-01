import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

for (const fontId of ['handwriting', 'dc'])
  test(`文本框跟随字距、固定毫米间隔缩放及保存重开：${fontId}`, async ({ page }, info) => {
    const errors = await open(page)
    await page.evaluate((fontId) => {
      window.__tables.saved = {
        ...window.__tables.preferences,
        text: '图一 霍尔效应原理图',
        layout: { ...window.__tables.preferences.layout, fontId },
        tables: [
          {
            id: 'spacing-caption',
            kind: 'textbox',
            page: 0,
            x: 15,
            y: 40,
            columnWidths: [100],
            rowHeights: [28],
            fontSize: 6,
            padding: 1,
            align: 'center',
            verticalAlign: 'middle',
            style: 'none',
            cells: [
              { row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '图一 霍尔效应原理图\nU̅_{H}=____' },
            ],
          },
        ],
      }
      window.__tables.menu['menu:action']('open')
    }, fontId)
    const caption = page.locator('[data-free-text-box="spacing-caption"]')
    await expect(caption.locator('path')).toHaveCount(1)
    const rendered = () => page.evaluate(() => window.__tables.document.pages[0].tables[0])
    const source = () => page.evaluate(() => window.__tables.payload.tables[0])
    const before = await rendered()
    await page.getByRole('button', { name: '排版设置', exact: true }).click()
    await page.getByLabel('字距', { exact: true }).fill('76.5')
    await page.getByLabel('字距', { exact: true }).press('Tab')
    await expect
      .poll(() => page.evaluate(() => window.__tables.document.settings.letterSpacing))
      .toBe(76.5)
    expect((await rendered()).strokes).not.toEqual(before.strokes)
    await page.getByLabel('额外字距', { exact: true }).fill('1.2')
    await page.getByLabel('额外字距', { exact: true }).press('Tab')
    await expect
      .poll(() => page.evaluate(() => window.__tables.document.settings.letterGap))
      .toBe(1.2)
    await page.getByRole('button', { name: '排版设置', exact: true }).click()

    const coordinates = async () =>
      (await caption.locator('path').getAttribute('d'))
        .match(/-?\d+(?:\.\d+)?(?:e[-+]?\d+)?/gi)
        .map(Number)
    for (const align of ['left', 'center', 'right']) {
      await caption.click()
      const previousId = await page.evaluate(() => window.__tables.document.id)
      await page.getByLabel('文本框对齐').selectOption(align)
      await expect.poll(() => page.evaluate(() => window.__tables.document.id)).not.toBe(previousId)
      await expect
        .poll(async () => (await rendered()).cells[0].align || (await source()).align)
        .toBe(align)
      await expect.poll(async () => (await rendered()).cells[0].textSpacing?.lines.length).toBe(2)
      const old = await source(),
        handle = await page.getByRole('button', { name: '调整文本框大小' }).boundingBox()
      const k = await page
        .locator('[data-paper="0"]')
        .evaluate((el) => el.getBoundingClientRect().width / 163)
      await page.mouse.move(handle.x + handle.width / 2, handle.y + handle.height / 2)
      await page.mouse.down()
      await page.mouse.move(
        handle.x + handle.width / 2 - 6 * k,
        handle.y + handle.height / 2 + 2 * k,
        { steps: 5 },
      )
      const live = await coordinates()
      expect(await source()).toEqual(old)
      await page.mouse.up()
      await expect.poll(async () => (await rendered()).width).toBeLessThan(old.columnWidths[0] - 5)
      const settled = await coordinates()
      expect(settled.length).toBe(live.length)
      expect(Math.max(...settled.map((v, i) => Math.abs(v - live[i])))).toBeLessThan(0.03)
    }
    const final = await rendered()
    expect(final.issues).toEqual([])
    await page.screenshot({
      path: `/tmp/bambuscribe-textbox-spacing-${fontId}-${info.project.name}.png`,
    })
    await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
    await expect.poll(() => page.evaluate(() => window.__tables.saved?.layout.letterGap)).toBe(1.2)
    const reopened = page.waitForResponse('**/__table-preview')
    await page.evaluate(() => window.__tables.menu['menu:action']('open'))
    await reopened
    await expect.poll(async () => (await rendered()).strokes).toEqual(final.strokes)
    await expect(page.getByRole('button', { name: '导出', exact: true })).toBeEnabled()
    await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
    await expect
      .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
      .toContain('A1 handwriting')
    expect(errors).toEqual([])
  })
