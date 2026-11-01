import { expect } from '@playwright/test'
import { tmpdir } from 'node:os'
import { test, open, insert, table, clipboard } from './tableHarness.mjs'

async function point(page, x, y, index = 0) {
  const paper = page.locator(`[data-paper="${index}"]`)
  await paper.scrollIntoViewIfNeeded()
  return paper.evaluate(
    (el, { x, y }) => {
      const r = el.getBoundingClientRect(),
        k = r.width / window.__tables.document.settings.paperWidth
      return { x: r.left + x * k, y: r.top + y * k }
    },
    { x, y },
  )
}
async function boxes(page) {
  return page.evaluate(() => window.__tables.payload.tables.filter((t) => t.kind === 'textbox'))
}
async function add(page, text = '表1 I-U关系', x = 20, y = 25) {
  await page.getByRole('button', { name: '文本框', exact: true }).click()
  const p = await point(page, x, y)
  await page.mouse.click(p.x, p.y)
  const input = page.getByLabel('文本框文字', { exact: true })
  await expect(input).toBeFocused()
  await input.fill(text)
  await input.press('Enter')
  await expect.poll(async () => (await boxes(page)).at(-1)?.cells[0].text).toBe(text)
}

test('inserts anywhere on a blank page, immediately types and exports real handwriting without a border', async ({
  page,
}) => {
  const errors = await open(page)
  await add(page)
  expect(await page.evaluate(() => window.__tables.payload.images)).toEqual([])
  const [box] = await boxes(page)
  expect(box.fontSize).toBe(await page.evaluate(() => window.__tables.payload.settings.fontSize))
  // Native WebKit mouse coordinates round to CSS pixels (about 0.15 mm here).
  expect(Math.abs(box.x - 20)).toBeLessThan(0.3)
  expect(Math.abs(box.y - 25)).toBeLessThan(0.3)
  await expect
    .poll(() =>
      page.evaluate(() => {
        const b = window.__tables.document?.pages[0]?.tables[0]
        if (!b || !b.strokes.length) return false
        const path = b.strokes
          .map((s) => s.map((p, i) => `${i ? 'L' : 'M'}${p.x},${p.y}`).join(' '))
          .join(' ')
        return (
          b.kind === 'textbox' &&
          b.strokes.length === b.cells[0].strokes.length &&
          document.querySelector('.free-text-ink path').getAttribute('d') === path
        )
      }),
    )
    .toBe(true)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  await page.screenshot({ path: `${tmpdir()}/bambuscribe-free-text-box.png` })
  expect(errors).toEqual([])
})

test('adds independent captions near a table and image without changing their content or masks', async ({
  page,
}) => {
  const errors = await open(page)
  await insert(page, { rows: 3, cols: 3, width: 100 })
  const grid = await table(page)
  await add(page, '表格注释', 20, 10)
  expect(await table(page)).toEqual(grid)
  await page.evaluate(() => {
    const canvas = document.createElement('canvas')
    canvas.width = 120
    canvas.height = 60
    const ctx = canvas.getContext('2d')
    ctx.fillStyle = 'white'
    ctx.fillRect(0, 0, 120, 60)
    ctx.strokeStyle = 'black'
    ctx.lineWidth = 2
    ctx.beginPath()
    ctx.moveTo(5, 30)
    ctx.lineTo(115, 30)
    ctx.stroke()
    window.__tables.saved = {
      version: 2,
      text: '',
      layout: window.__tables.payload.settings,
      writing: window.__tables.preferences.writing,
      images: [
        {
          id: 'diagram',
          page: 0,
          data: canvas.toDataURL(),
          x: 80,
          y: 60,
          width: 40,
          height: 20,
          rotation: 0,
          mode: 'line',
        },
      ],
      tables: [],
    }
  })
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-image-id="diagram"]')).toHaveCount(1)
  await add(page, '图片注释', 90, 65)
  expect((await boxes(page)).length).toBe(1)
  const image = await page.evaluate(() => window.__tables.payload.images[0])
  expect(image.textRegions?.filter((r) => r.source === 'manual') ?? []).toEqual([])
  expect(image.x).toBe(80)
  expect(image.y).toBe(60)
  expect(errors).toEqual([])
})

test('moves, resizes text automatically, aligns, deletes, and restores through shared undo', async ({
  page,
}) => {
  const errors = await open(page)
  await add(page)
  await page.getByRole('button', { name: '选择', exact: true }).click()
  const original = (await boxes(page))[0]
  const originalSize = await page.evaluate(
    () => window.__tables.document.pages[0].tables[0].fontSize,
  )
  const r = await page.getByRole('button', { name: '移动文本框' }).boundingBox()
  await page.mouse.move(r.x + 8, r.y + 8)
  await page.mouse.down()
  await page.mouse.move(r.x + 38, r.y + 38, { steps: 5 })
  await page.mouse.up()
  await expect.poll(async () => (await boxes(page))[0].x).toBeGreaterThan(original.x)
  const size = await page.getByRole('button', { name: '调整文本框大小' }).boundingBox()
  await page.mouse.move(size.x + 5, size.y + 5)
  await page.mouse.down()
  await page.mouse.move(size.x + 45, size.y + 25, { steps: 5 })
  await page.mouse.up()
  await expect
    .poll(async () => (await boxes(page))[0].columnWidths[0])
    .toBeGreaterThan(original.columnWidths[0])
  expect((await boxes(page))[0].fontSize).toBeGreaterThan(originalSize)
  await expect(page.getByLabel('文本框字号')).toHaveAttribute('readonly', '')
  await page.getByLabel('文本框对齐').selectOption('center')
  await expect.poll(async () => (await boxes(page))[0].align).toBe('center')
  const changed = (await boxes(page))[0]
  await page.getByRole('button', { name: '删除文本框', exact: true }).click()
  await expect(page.locator('[data-free-text-box]')).toHaveCount(0)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-free-text-box]')).toHaveCount(1)
  await expect.poll(async () => (await boxes(page))[0]).toEqual(changed)
  await page.locator('[data-free-text-box]').click()
  await page.keyboard.press('Delete')
  await expect(page.locator('[data-free-text-box]')).toHaveCount(0)
  expect(errors).toEqual([])
})

test('clipboard and project save retain captions, including an edit that has not lost focus', async ({
  page,
}) => {
  const errors = await open(page)
  await add(page)
  await page.locator('[data-free-text-box]').dblclick()
  await page.getByLabel('文本框文字').fill('尚未失焦')
  await page.keyboard.press('Meta+s')
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved?.tables[0].cells[0].text))
    .toBe('尚未失焦')
  await expect.poll(async () => (await boxes(page))[0].cells[0].text).toBe('尚未失焦')
  const original = (await boxes(page))[0]
  await page.locator('[data-free-text-box]').click()
  const copied = await page.evaluate(() => {
    const data = new DataTransfer()
    document
      .querySelector('[data-free-text-box]')
      .dispatchEvent(
        new ClipboardEvent('copy', { bubbles: true, cancelable: true, clipboardData: data }),
      )
    return { text: data.getData('text/plain'), html: data.getData('text/html') }
  })
  expect(copied.text).toBe('尚未失焦')
  await page.evaluate(({ text, html }) => {
    const data = new DataTransfer()
    data.setData('text/plain', text)
    data.setData('text/html', html)
    document
      .querySelector('.document-viewport')
      .dispatchEvent(
        new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: data }),
      )
  }, copied)
  await expect(page.locator('[data-free-text-box]')).toHaveCount(2)
  await expect.poll(async () => (await boxes(page)).length).toBe(2)
  expect((await boxes(page))[1].cells).toEqual(original.cells)
  expect((await boxes(page))[1].id).not.toBe(original.id)
  const autosaved = await boxes(page)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-free-text-box]')).toHaveCount(2)
  await expect.poll(async () => (await boxes(page)).length).toBe(2)
  expect(await boxes(page)).toEqual(autosaved)
  expect(errors).toEqual([])
})

test('only Shift+Enter makes new lines, and missing or overflowing text blocks output', async ({
  page,
}) => {
  const errors = await open(page)
  await add(page, '实验'.repeat(100))
  await expect(page.locator('.document-alert[role="alert"]')).toContainText('宽度不足')
  expect((await boxes(page))[0].rowHeights.length).toBe(1)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  expect(await page.evaluate(() => window.__tables.exported)).toBeNull()
  await page.locator('[data-free-text-box]').dblclick()
  const input = page.getByLabel('文本框文字')
  await expect(input).toHaveAttribute('wrap', 'off')
  await input.fill('实验')
  await input.evaluate((el) =>
    el.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Enter', bubbles: true, isComposing: true }),
    ),
  )
  await expect(input).toBeFocused()
  await input.press('Shift+Enter')
  await page.keyboard.insertText('数据')
  await input.press('Enter')
  await expect.poll(async () => (await boxes(page))[0].cells[0].text).toBe('实验\n数据')
  await expect(page.locator('.document-alert[role="alert"]')).toHaveCount(0)
  await page.locator('[data-free-text-box]').dblclick()
  await input.fill('🙂')
  await input.press('Enter')
  await expect(page.locator('.document-alert[role="alert"]')).toContainText('缺字')
  expect(errors).toEqual([])
})

test('captions drag across pages and cancel without moving images or tables', async ({ page }) => {
  const errors = await open(page)
  await insert(page, { rows: 2, cols: 2, width: 60 })
  await page.locator('[data-table-cell="0:0"]').click()
  await page.getByRole('button', { name: '表格设置', exact: true }).click()
  await page.getByLabel('表格页码').fill('2')
  await page.getByLabel('表格页码').press('Tab')
  await expect.poll(async () => (await table(page)).page).toBe(1)
  await page.getByRole('button', { name: '关闭表格设置' }).click()
  await add(page, '标题', 25, 200, 0)
  await page.getByRole('button', { name: '选择', exact: true }).click()
  await page.getByLabel('页面缩放').selectOption('50')
  const original = (await boxes(page))[0]
  const target = await point(page, 35, 35, 1)
  const handle = await page.getByRole('button', { name: '移动文本框' }).boundingBox()
  await page.mouse.move(handle.x + 8, handle.y + 8)
  await page.mouse.down()
  await page.mouse.move(target.x, target.y, { steps: 5 })
  await page.mouse.up()
  await expect.poll(async () => (await boxes(page))[0].page).toBe(1)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(async () => (await boxes(page))[0].page).toBe(0)
  expect((await boxes(page))[0]).toEqual(original)
  const cancelledTarget = await point(page, 35, 35, 1)
  const cancelledHandle = await page.getByRole('button', { name: '移动文本框' }).boundingBox()
  await page.mouse.move(cancelledHandle.x + 8, cancelledHandle.y + 8)
  await page.mouse.down()
  await page.mouse.move(cancelledTarget.x, cancelledTarget.y, { steps: 5 })
  await page.keyboard.press('Escape')
  await page.mouse.up()
  expect((await boxes(page))[0]).toEqual(original)
  expect((await table(page)).page).toBe(1)
  expect(errors).toEqual([])
})
