import { expect } from '@playwright/test'
import { test } from './tableHarness.mjs'
import { setup, generate } from './calculationHarness.mjs'

async function seed(page, text, layout = {}, objects = {}) {
  const errors = await setup(page, { closed: true })
  await page.evaluate(
    ({ text, layout, objects }) => {
      window.__tables.saved = {
        ...window.__tables.preferences,
        ...objects,
        text,
        layout: { ...window.__tables.preferences.layout, ...layout },
      }
      window.__tables.menu['menu:action']('open')
      // One short calculation keeps the test focused on the insertion point.
      window.__calc.plan.sections = [window.__calc.plan.sections[0]]
      window.__calc.plan.sections[0].steps = [window.__calc.plan.sections[0].steps[0]]
    },
    { text, layout, objects },
  )
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages.map((p) => p.text).join('')))
    .toBe(text)
  await page.getByLabel('页面缩放').selectOption('50')
  return errors
}

async function clickCaret(page, pageIndex, stop) {
  const paper = page.locator(`[data-paper="${pageIndex}"]`)
  await paper.scrollIntoViewIfNeeded()
  const point = await paper.evaluate((el, stop) => {
    const box = el.getBoundingClientRect()
    const doc = window.__tables.document
    const scale = box.width / doc.settings.paperWidth
    return {
      x: box.left + stop.x * scale,
      y: box.top + (stop.y + doc.settings.fontSize / 2) * scale,
    }
  }, stop)
  await page.mouse.click(point.x, point.y)
  await expect(page.getByLabel('书写内容')).toBeFocused()
  await expect
    .poll(() => page.getByLabel('书写内容').evaluate((el) => el.selectionStart))
    .toBe(stop.index)
}

async function insert(page) {
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '数据处理', exact: true }).click()
  await generate(page)
  const generated = await page.evaluate(() => window.__calc.preview.text)
  await page.getByRole('button', { name: '插入正文', exact: true }).click()
  await expect(page.getByRole('dialog', { name: '数据处理', exact: true })).not.toBeVisible()
  return generated
}

test('calculation inserts at a clicked body caret on a later page and can be undone', async ({
  page,
}) => {
  const original = '第一页正文\f第二页正文\f第三页前文\n光标位置\n后文'
  const errors = await seed(page, original)
  const index = original.indexOf('光标位置')
  const stop = await page.evaluate(
    (index) => window.__tables.document.pages[2].carets.find((c) => c.index === index),
    index,
  )
  await clickCaret(page, 2, stop)
  const generated = await insert(page)
  const expected = original.slice(0, index) + generated + '\n' + original.slice(index)
  await expect(page.getByLabel('书写内容')).toHaveValue(expected)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[2]?.formulas?.length ?? 0))
    .toBe(1)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue(original)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue(expected)
  expect(errors).toEqual([])
})

test('calculation keeps the clicked blank row instead of moving to the page start', async ({
  page,
}) => {
  const original = '第一页正文\f第二页正文\f'
  const errors = await seed(page, original)
  const stop = { index: original.length, x: 0, y: 5 * 15 }
  await clickCaret(page, 2, stop)
  const generated = await insert(page)
  const expected = original + '\n'.repeat(5) + generated
  await expect(page.getByLabel('书写内容')).toHaveValue(expected)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[2]?.cells?.[0]?.y))
    .toBe(stop.y)
  await page.getByRole('button', { name: '另存为', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.text)).toBe(expected)
  await page.getByRole('button', { name: '打开项目', exact: true }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages.map((p) => p.text).join('')))
    .toBe(expected)
  expect(errors).toEqual([])
})

test('calculation uses a later empty sheet when all source text indices are zero', async ({
  page,
}) => {
  const errors = await seed(
    page,
    '',
    {
      pageLayouts: [{ lineCount: 16 }, { lineCount: 16 }, { lineCount: 16 }],
    },
    {
      tables: [
        {
          id: 'last-sheet',
          page: 2,
          x: 100,
          y: 180,
          columnWidths: [40],
          rowHeights: [20],
          cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '已有表格' }],
          fontSize: 5,
          padding: 1,
          style: 'grid',
          align: 'left',
          verticalAlign: 'middle',
        },
      ],
    },
  )
  await expect(page.locator('[data-paper]')).toHaveCount(3)
  await clickCaret(page, 2, { index: 0, x: 0, y: 30 })
  const generated = await insert(page)
  await expect(page.getByLabel('书写内容')).toHaveValue('\n'.repeat(34) + generated)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[2]?.cells?.[0]?.y))
    .toBe(30)
  expect(errors).toEqual([])
})

test('scrolling away from a placed caret does not change the calculation insertion point', async ({
  page,
}) => {
  const original = '第一页正文\f第二页正文\f第三页前文\n目标\n后文'
  const errors = await seed(page, original)
  const index = original.indexOf('目标')
  const stop = await page.evaluate(
    (index) => window.__tables.document.pages[2].carets.find((c) => c.index === index),
    index,
  )
  await clickCaret(page, 2, stop)
  await page.locator('.document-viewport').evaluate((el) => {
    el.scrollTop = 0
  })
  await expect.poll(() => page.locator('.document-viewport').evaluate((el) => el.scrollTop)).toBe(0)
  const generated = await insert(page)
  await expect(page.getByLabel('书写内容')).toHaveValue(
    original.slice(0, index) + generated + '\n' + original.slice(index),
  )
  expect(errors).toEqual([])
})

test('selected text is replaced at its range even when the menu temporarily restores focus', async ({
  page,
}) => {
  const original = '第一页正文\f第二页前文\n替换范围\n第二页后文'
  const errors = await seed(page, original)
  const from = original.indexOf('替换范围'),
    to = from + 4
  await page.getByLabel('书写内容').evaluate(
    (input, { from, to }) => {
      input.focus({ preventScroll: true })
      input.setSelectionRange(from, to)
      input.dispatchEvent(new Event('select'))
    },
    { from, to },
  )
  const generated = await insert(page)
  await expect(page.getByLabel('书写内容')).toHaveValue(
    original.slice(0, from) + generated + original.slice(to),
  )
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue(original)
  await expect
    .poll(() => page.getByLabel('书写内容').evaluate((el) => [el.selectionStart, el.selectionEnd]))
    .toEqual([from, to])
  expect(errors).toEqual([])
})
