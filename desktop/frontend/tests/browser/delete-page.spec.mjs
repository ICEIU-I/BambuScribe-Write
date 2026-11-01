import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

const rail = (page) => page.locator('.page-rail button')
const payload = (page) => page.evaluate(() => window.__tables.payload)
const doc = (page) => page.evaluate(() => window.__tables.document)
async function seed(page, options = {}) {
  const errors = await open(page)
  await page.evaluate((options) => {
    const table = (id, page, y = 45) => ({
      id,
      page,
      x: 90,
      y,
      columnWidths: [30],
      rowHeights: [12],
      cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '表格' }],
      fontSize: 5,
      padding: 1,
      align: 'left',
      verticalAlign: 'middle',
      style: 'grid',
    })
    const canvas = document.createElement('canvas')
    canvas.width = 30
    canvas.height = 30
    const context = canvas.getContext('2d')
    context.fillStyle = 'white'
    context.fillRect(0, 0, 30, 30)
    context.strokeStyle = 'black'
    context.strokeRect(2, 2, 26, 26)
    const images = [1, 2].map((page) => ({
      id: `image-${page}`,
      page,
      x: 125,
      y: 90,
      width: 25,
      height: 25,
      data: canvas.toDataURL(),
      mode: 'line',
      rotation: 0,
      textRegions: [],
    }))
    window.__tables.saved = {
      ...window.__tables.preferences,
      layout: { ...window.__tables.preferences.layout, lineCount: 4 },
      text: Array.from({ length: 12 }, (_, i) => `第${i + 1}行`).join('\n'),
      images,
      tables: [
        table('middle', 1),
        { ...table('caption', 1, 70), kind: 'textbox', style: 'none' },
        table('later', 2),
      ],
      erasures: [1, 2].map((page) => ({
        id: `erase-${page}`,
        page,
        x: 155,
        y: 200,
        width: 2,
        height: 2,
      })),
      ...options,
    }
    window.__tables.menu['menu:action']('open')
  }, options)
  await expect(rail(page)).toHaveCount(3)
  await page.getByLabel('页面缩放').selectOption('50')
  return errors
}
async function remove(page, index) {
  await rail(page).nth(index).click({ button: 'right' })
  const item = page.getByRole('menuitem', { name: '删除整页', exact: true })
  await expect(item).toBeEnabled()
  await item.click()
  await expect(page.getByRole('menu')).toHaveCount(0)
}

test('right click deletes that page, shifts later content and restores everything in one undo', async ({
  page,
}) => {
  const errors = await seed(page)
  const before = await payload(page)
  const preview = await doc(page)
  await expect(rail(page).first()).toHaveAttribute('aria-current', 'page')
  await remove(page, 1)
  await expect(rail(page)).toHaveCount(2)
  const after = await payload(page)
  expect(after.text).toBe(preview.pages[0].text + preview.pages[2].text)
  expect(after.images).toEqual([{ ...before.images[1], page: 1 }])
  expect(after.tables).toEqual([{ ...before.tables[2], page: 1 }])
  expect(after.erasures).toEqual([{ ...before.erasures[1], page: 1 }])
  await page.keyboard.press('Meta+z')
  await expect(rail(page)).toHaveCount(3)
  expect(await payload(page)).toEqual(before)
  await page.keyboard.press('Meta+Shift+z')
  await expect(rail(page)).toHaveCount(2)
  expect(await payload(page)).toEqual(after)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.text)).toBe(after.text)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-paper]')).toHaveCount(2)
  await page.getByRole('button', { name: '导出', exact: true }).click()
  const exportPage = page.getByRole('button', { name: /导出当前页/ })
  await expect(exportPage).toBeEnabled()
  await exportPage.click()
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect(await payload(page)).toEqual(after)
  expect(errors).toEqual([])
})

test('deleting blank and nonempty final pages removes their separator; the last sheet stays editable', async ({
  page,
}) => {
  const errors = await seed(page, {
    text: '正文\n'.repeat(8),
    images: [],
    tables: [],
    erasures: [],
  })
  await remove(page, 2)
  await expect(rail(page)).toHaveCount(2)
  expect((await payload(page)).text).toBe('正文\n'.repeat(7) + '正文')
  await remove(page, 1)
  await expect(rail(page)).toHaveCount(1)
  const previous = (await payload(page)).text
  expect(previous).toBe('正文\n'.repeat(3) + '正文')
  await remove(page, 0)
  await expect(page.getByLabel('书写内容')).toHaveValue('')
  await expect.poll(async () => (await doc(page)).characterCount).toBe(0)
  await expect(rail(page)).toHaveCount(1)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue(previous)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await page.getByLabel('书写内容').fill('新内容')
  await expect.poll(async () => (await payload(page)).text).toBe('新内容')
  expect(errors).toEqual([])
})

test('object-only pages shift and deleting the first page keeps the remaining objects', async ({
  page,
}) => {
  const errors = await seed(page, { text: '' })
  const before = await payload(page)
  await remove(page, 0)
  await expect(rail(page)).toHaveCount(2)
  const after = await payload(page)
  expect(after.images.map((i) => i.page)).toEqual([0, 1])
  expect(after.tables.map((t) => t.page)).toEqual([0, 0, 1])
  await remove(page, 0)
  await expect(rail(page)).toHaveCount(1)
  expect((await payload(page)).images).toEqual([{ ...before.images[1], page: 0 }])
  await page.keyboard.press('Meta+z')
  await expect(rail(page)).toHaveCount(2)
  expect(await payload(page)).toEqual(after)
  expect(errors).toEqual([])
})

test('page menu supports Escape, outside click and keyboard opening in both themes', async ({
  page,
}) => {
  const errors = await seed(page)
  const before = await payload(page)
  for (const theme of ['light', 'dark']) {
    if (theme === 'dark') await page.getByRole('button', { name: '深色主题', exact: true }).click()
    await rail(page).nth(1).click({ button: 'right' })
    await expect(page.getByRole('menu', { name: '第 2 页操作' })).toBeVisible()
    await expect(page.getByRole('menuitem', { name: '复制整页' })).toBeFocused()
    await page.keyboard.press('ArrowDown')
    await expect(page.getByRole('menuitem', { name: '插入空白页' })).toBeFocused()
    await page.keyboard.press('ArrowDown')
    await expect(page.getByRole('menuitem', { name: '删除整页' })).toBeFocused()
    await page.screenshot({ path: test.info().outputPath(`delete-page-${theme}.png`) })
    await page.keyboard.press('Escape')
    await expect(page.getByRole('menu')).toHaveCount(0)
    await expect(rail(page).nth(1)).toBeFocused()
    await page.keyboard.press('Shift+F10')
    await expect(page.getByRole('menu')).toBeVisible()
    await page.mouse.click(500, 100)
    await expect(page.getByRole('menu')).toHaveCount(0)
    expect(await payload(page)).toEqual(before)
  }
  expect(errors).toEqual([])
})
