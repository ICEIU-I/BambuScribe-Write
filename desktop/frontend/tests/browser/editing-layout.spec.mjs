import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

test('圆圈序号与下一行的字框相交但笔画分离时不报重叠', async ({ page }) => {
  const errors = await open(page, {}, true)
  await page.evaluate(() => {
    window.__tables.saved = {
      ...window.__tables.preferences,
      text: '③\n看清',
      layout: {
        ...window.__tables.preferences.layout,
        fontId: 'handwriting',
        paperWidth: 210,
        paperHeight: 297,
        marginTop: 25.4,
        marginBottom: 25.4,
        marginLeft: 17,
        marginRight: 17,
        lineCount: 30,
        fontSize: (297 - 2 * 25.4) / 30,
        letterSpacing: 72.5,
        lineSpacing: 85,
        handwritingVariation: 9,
        handwritingSeed: 2418718602,
      },
    }
    window.__tables.menu['menu:action']('open')
  })
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toBe('③\n看清')
  await expect(page.locator('.document-alert[role="alert"]')).toHaveCount(0)
  await expect(page.locator('[data-layout-issue]')).toHaveCount(0)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  const layout = await page.evaluate(() => window.__tables.document.settings)
  expect(layout.letterSpacing).toBe(72.5)
  expect(layout.lineSpacing).toBe(85)
  expect(errors).toEqual([])
})

async function expectVisibleWaves(page) {
  const locations = await page.locator('[data-paper="0"]').evaluate((paper) => {
    const doc = window.__tables.document
    const rect = paper.getBoundingClientRect()
    const scale = rect.width / doc.settings.paperWidth
    return doc.pages[0].diagnostics.map((d) => ({
      x: Math.round(rect.left + d.x * scale),
      y: Math.round(rect.top + (d.y + d.height + 0.5) * scale),
      width: Math.max(1, Math.floor(d.width * scale)),
    }))
  })
  const screenshot = await page.screenshot({ scale: 'css' })
  const visible = await page.evaluate(
    async ({ png, locations }) => {
      const image = new Image()
      image.src = `data:image/png;base64,${png}`
      await image.decode()
      const canvas = document.createElement('canvas')
      canvas.width = image.width
      canvas.height = image.height
      const ctx = canvas.getContext('2d')
      ctx.drawImage(image, 0, 0)
      return locations.map(({ x, y, width }) => {
        const pixels = ctx.getImageData(x, y - 3, width, 7).data
        let red = 0
        for (let i = 0; i < pixels.length; i += 4)
          if (pixels[i] > 170 && pixels[i + 1] < 120 && pixels[i + 2] < 120) red++
        return red > 2
      })
    },
    { png: screenshot.toString('base64'), locations },
  )
  expect(visible.length).toBeGreaterThanOrEqual(2)
  expect(visible.every(Boolean)).toBe(true)
}

test('回车遇到行间重叠仍更新正文、换行及光标，修正行距后恢复导出', async ({ page }, info) => {
  const errors = await open(page, {}, true)
  await page.evaluate(() => {
    window.__tables.saved = {
      ...window.__tables.preferences,
      text: '你四、实验',
      layout: { ...window.__tables.preferences.layout, fontSize: 16.5, lineSpacing: 50 },
    }
    window.__tables.menu['menu:action']('open')
  })
  const input = page.getByRole('textbox', { name: '书写内容', exact: true })
  await expect(input).toHaveValue('你四、实验')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toBe('你四、实验')
  await expect(page.locator('[data-layout-issue]')).toHaveCount(0)
  await input.focus()
  await input.evaluate((el) => {
    el.setSelectionRange(1, 1)
    el.dispatchEvent(new Event('select'))
  })
  await page.keyboard.press('Enter')
  await expect(input).toHaveValue('你\n四、实验')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toBe('你\n四、实验')
  expect(await input.evaluate((el) => el.selectionStart)).toBe(2)
  const cells = await page.evaluate(() => window.__tables.document.pages[0].cells)
  expect(cells[1].start).toBe(2)
  expect(cells[1].y).toBeGreaterThan(cells[0].y)
  const paper = await page.locator('[data-paper="0"]').boundingBox()
  const caret = await page.locator('.paper-caret').boundingBox()
  const scale = paper.width / 163
  expect(caret.x - paper.x).toBeCloseTo(cells[1].x * scale, 0)
  expect(caret.y - paper.y).toBeCloseTo(cells[1].y * scale, 0)
  await expect(page.locator('.document-alert[role="alert"]')).toContainText('行距过小')
  await expect(page.locator('[data-layout-issue="line-overlap"]')).toHaveCount(2)
  await expectVisibleWaves(page)
  await page.screenshot({ path: `/tmp/bambuscribe-enter-overlap-${info.project.name}.png` })
  await page.getByRole('button', { name: '导出', exact: true }).click()
  await expect(page.getByRole('button', { name: '导出当前页' })).toBeDisabled()
  await expect(page.getByRole('button', { name: '导出全部页面' })).toBeDisabled()
  await page.keyboard.press('Escape')
  await page.getByLabel('页面缩放').selectOption('50')
  await expectVisibleWaves(page)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(input).toHaveValue('你四、实验')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toBe('你四、实验')
  await expect(page.locator('[data-layout-issue]')).toHaveCount(0)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(input).toHaveValue('你\n四、实验')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toBe('你\n四、实验')
  await expect(page.locator('[data-layout-issue="line-overlap"]')).toHaveCount(2)
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  await page.getByLabel('行距', { exact: true }).fill('100')
  await page.getByLabel('行距', { exact: true }).press('Tab')
  await expect(page.locator('.document-alert[role="alert"]')).toHaveCount(0)
  await expect(page.locator('[data-layout-issue]')).toHaveCount(0)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect(errors).toEqual([])
})
