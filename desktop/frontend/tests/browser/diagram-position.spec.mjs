import { expect } from '@playwright/test'
import { test } from './tableHarness.mjs'
import { setup, menu, start, settled, png } from './diagramHarness.mjs'

async function seed(page, text, layout = {}) {
  const errors = await setup(page)
  await page.evaluate(
    ({ text, layout }) => {
      window.__tables.saved = {
        ...window.__tables.preferences,
        text,
        layout: { ...window.__tables.preferences.layout, ...layout },
      }
      window.__tables.menu['menu:action']('open')
      window.__diagram.scene = {
        width: 100,
        height: 20,
        strokes: [
          {
            id: 'line',
            points: [
              { x: 5, y: 5 },
              { x: 95, y: 5 },
            ],
          },
        ],
        circles: [],
        labels: [],
        uncertainties: [],
      }
    },
    { text, layout },
  )
  await expect(page.getByLabel('书写内容')).toHaveValue(text)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages.map((p) => p.text).join('')))
    .toBe(text)
  if (text) await settled(page)
  await page.getByLabel('页面缩放').selectOption('50')
  return errors
}

async function clickCaret(page, pageIndex, stop) {
  const paper = page.locator(`[data-paper="${pageIndex}"]`)
  await paper.scrollIntoViewIfNeeded()
  const point = await paper.evaluate((el, stop) => {
    const box = el.getBoundingClientRect(),
      scale = box.width / window.__tables.document.settings.paperWidth
    return { x: box.left + stop.x * scale, y: box.top + (stop.y + 3) * scale }
  }, stop)
  await page.mouse.click(point.x, point.y)
  await expect(page.getByLabel('书写内容')).toBeFocused()
}

test('without an active caret, two figures follow the body instead of the initial page-zero caret', async ({
  page,
}) => {
  const text = '第一页正文\f第二页正文\f最后一段正文\n最后一行\n\f'
  const errors = await seed(page, text)
  const before = await page.evaluate(() => window.__tables.document)
  await expect(page.getByLabel('书写内容')).not.toBeFocused()
  await menu(page)
  await page
    .getByLabel('图像文件', { exact: true })
    .setInputFiles([1, 2].map((i) => ({ name: `${i}.png`, mimeType: 'image/png', buffer: png })))
  await page.getByRole('button', { name: '识别并插入', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(2)
  await settled(page)
  const after = await page.evaluate(() => window.__tables.document)
  expect(after.pages[0].images ?? []).toHaveLength(0)
  const [first, second] = after.pages[2].images
  expect(first.y).toBe(30)
  expect(second.y).toBe(first.y + first.height + 4)
  expect(after.pages.flatMap((p) => p.cells)).toEqual(before.pages.flatMap((p) => p.cells))
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(2)
  await settled(page)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved?.images.map((i) => i.page)))
    .toEqual([2, 2])
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await settled(page)
  expect(errors).toEqual([])
})

test('a clicked blank row on a later page survives menu focus and scrolling', async ({ page }) => {
  const errors = await seed(page, '第一页正文\f第二页正文\f')
  await clickCaret(page, 2, { x: 0, y: 75 })
  await page.locator('.document-viewport').evaluate((el) => {
    el.scrollTop = 0
  })
  await start(page)
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await settled(page)
  const images = await page.evaluate(() => window.__tables.payload.images)
  expect(images[0]).toMatchObject({ page: 2, x: 0, y: 75 })
  expect(errors).toEqual([])
})

test('a caret at the last text character inserts below the final line without moving it', async ({
  page,
}) => {
  const errors = await seed(page, '第一页正文\f末尾正文')
  const before = await page.evaluate(() => window.__tables.document)
  await clickCaret(page, 1, before.pages[1].carets.at(-1))
  await start(page)
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await settled(page)
  const after = await page.evaluate(() => window.__tables.document)
  expect(after.pages[1].images[0].y).toBe(15)
  expect(after.pages.flatMap((p) => p.cells)).toEqual(before.pages.flatMap((p) => p.cells))
  expect(errors).toEqual([])
})

test('compact body spacing keeps the final line above the imported figure', async ({ page }) => {
  const errors = await seed(page, '前文\f' + '实验记录与末行文字\n'.repeat(9), {
    fontSize: 8.3,
    lineCount: 30,
    lineSpacing: 85,
    handwritingVariation: 15,
    handwritingSeed: 1234,
  })
  const before = await page.evaluate(() => window.__tables.document)
  await start(page)
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await settled(page)
  const after = await page.evaluate(() => window.__tables.document)
  expect(after.pages.map((p) => p.cells)).toEqual(before.pages.map((p) => p.cells))
  const last = before.pages[1].cells.at(-1)
  expect(after.pages[1].images[0].y).toBeGreaterThanOrEqual(last.y + last.height + 1)
  expect(errors).toEqual([])
})

test('an empty document uses the viewed page instead of page zero', async ({ page }) => {
  const errors = await seed(page, '\f\f', {
    pageLayouts: [{ lineCount: 16 }, { lineCount: 16 }, { lineCount: 16 }],
  })
  await page.locator('.page-rail button').nth(2).click()
  // Page navigation scrolls smoothly; the viewport updates the active page
  // while crossing earlier sheets. Wait until the requested sheet is reached.
  const destination = await page.locator('.document-viewport').evaluate((el) => {
    const paper = el.querySelector('[data-paper="2"]')
    return Math.min(
      el.scrollHeight - el.clientHeight,
      el.scrollTop + paper.getBoundingClientRect().top - el.getBoundingClientRect().top,
    )
  })
  await expect
    .poll(() =>
      page
        .locator('.document-viewport')
        .evaluate((el, y) => Math.abs(el.scrollTop - y), destination),
    )
    .toBeLessThan(2)
  await expect(page.locator('.page-rail button').nth(2)).toHaveAttribute('aria-current', 'page')
  await start(page)
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await settled(page)
  expect(await page.evaluate(() => window.__tables.payload.images[0].page)).toBe(2)
  expect(errors).toEqual([])
})
