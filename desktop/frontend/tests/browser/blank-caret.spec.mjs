import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

const input = (page) => page.getByLabel('书写内容', { exact: true })
const document = (page) => page.evaluate(() => window.__tables.document)

async function seed(page, layout = {}, text = '') {
  const errors = await open(page)
  await page.evaluate(
    ({ layout, text }) => {
      window.__tables.saved = {
        ...window.__tables.preferences,
        text,
        layout: { ...window.__tables.preferences.layout, ...layout },
      }
      window.__tables.menu['menu:action']('open')
    },
    { layout, text },
  )
  await expect
    .poll(async () => (await document(page))?.pages.map((p) => p.text).join(''))
    .toBe(text)
  await page.getByLabel('页面缩放').selectOption('50')
  return errors
}

async function clickRow(page, row, x = 80, pageIndex = 0) {
  const doc = await document(page)
  const y =
    doc.settings.marginTop + row * (doc.pages[pageIndex].lineHeight + doc.settings.paragraphGap)
  const paper = page.locator(`[data-paper="${pageIndex}"]`)
  await paper.scrollIntoViewIfNeeded()
  const point = await paper.evaluate(
    (el, p) => {
      const rect = el.getBoundingClientRect()
      const scale = rect.width / p.width
      return { x: rect.left + p.x * scale, y: rect.top + (p.y + p.height / 2) * scale }
    },
    {
      x,
      y,
      width: doc.settings.paperWidth,
      height: Math.min(doc.settings.fontSize, doc.pages[pageIndex].lineHeight),
    },
  )
  await page.mouse.click(point.x, point.y)
  await expect(input(page)).toBeFocused()
  return y
}

async function expectCaret(page, x, y, pageIndex = 0) {
  // Measure the actual rendered caret against the paper, including zoom.
  await expect
    .poll(() =>
      page.locator('.caret-anchor').evaluate(
        (el, expected) => {
          const paper = document
            .querySelector(`[data-paper="${expected.pageIndex}"]`)
            .getBoundingClientRect()
          const caret = el.getBoundingClientRect()
          const scale = paper.width / window.__tables.document.settings.paperWidth
          return Math.max(
            Math.abs((caret.left - paper.left) / scale - expected.x),
            Math.abs((caret.top - paper.top) / scale - expected.y),
          )
        },
        { x, y, pageIndex },
      ),
    )
    .toBeLessThan(0.02)
}

test('every blank row is clickable at its first column without editing the project', async ({
  page,
}) => {
  const errors = await seed(page)
  const before = await page.evaluate(() => window.__tables.payload)
  for (const row of [...Array(16).keys(), 2]) {
    const y = await clickRow(page, row, row % 2 ? 145 : 12)
    await expectCaret(page, 0, y)
    await expect(input(page)).toHaveValue('')
  }
  await page.keyboard.press('ArrowDown')
  await expectCaret(page, 0, 45)
  await page.keyboard.press('ArrowUp')
  await expectCaret(page, 0, 30)
  await page.keyboard.press('Home')
  await expectCaret(page, 0, 30)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.text)).toBe('')
  expect(await page.evaluate(() => window.__tables.payload)).toEqual(before)
  expect((await document(page)).characterCount).toBe(0)
  await page.screenshot({ path: test.info().outputPath('blank-row-caret.png') })
  expect(errors).toEqual([])
})

test('unused rows below existing text accept clicks and typing while body navigation stays intact', async ({
  page,
}) => {
  const errors = await seed(page, {}, '已有正文')
  const original = await document(page)
  for (const row of [1, 7, 15]) {
    const y = await clickRow(page, row)
    await expectCaret(page, 0, y)
    await expect(input(page)).toHaveValue('已有正文')
    expect((await document(page)).id).toBe(original.id)
  }
  await page.keyboard.press('Delete')
  await expect(input(page)).toHaveValue('已有正文')
  await page.keyboard.press('Backspace')
  await expectCaret(page, 0, 210)
  await page.keyboard.press('ArrowDown')
  await expectCaret(page, 0, 225)
  await page.keyboard.insertText('末行')
  const text = '已有正文' + '\n'.repeat(15) + '末行'
  await expect(input(page)).toHaveValue(text)
  await expect.poll(async () => (await document(page)).pages[0].cells.at(-1)?.y).toBe(225)
  await page.keyboard.press('Meta+z')
  await expect(input(page)).toHaveValue('已有正文')
  await page.keyboard.press('Meta+Shift+z')
  await expect(input(page)).toHaveValue(text)
  // Existing writing rows must still participate in mouse and keyboard navigation.
  await clickRow(page, 0, 0)
  await expectCaret(page, 0, 0)
  await page.keyboard.press('End')
  await expect.poll(() => input(page).evaluate((el) => el.selectionStart)).toBe(4)
  expect(errors).toEqual([])
})

test('blank rows before an explicit page boundary preserve later pages and Chinese composition', async ({
  page,
}) => {
  const originalText = '第一页\f已有正文\n\f后页正文'
  const errors = await seed(page, {}, originalText)
  const original = await document(page)
  await clickRow(page, 1, 80, 1)
  await expectCaret(page, 0, 15, 1)
  await page.keyboard.insertText('首行')
  await expect(input(page)).toHaveValue('第一页\f已有正文\n首行\f后页正文')
  await expect.poll(async () => (await document(page)).pages[1].cells.at(-1)?.y).toBe(15)
  await page.keyboard.press('Meta+z')
  await expect(input(page)).toHaveValue(originalText)
  await expect.poll(async () => (await document(page)).id).toBe(original.id)
  await clickRow(page, 15, 80, 1)
  await expectCaret(page, 0, 225, 1)
  await input(page).dispatchEvent('compositionstart', { data: '' })
  await input(page).dispatchEvent('compositionend', { data: '' })
  await expectCaret(page, 0, 225, 1)
  await input(page).evaluate((el) => {
    el.dispatchEvent(new CompositionEvent('compositionstart', { bubbles: true }))
    el.setRangeText('补充', el.selectionStart, el.selectionEnd, 'end')
    el.dispatchEvent(new InputEvent('input', { data: '补充', isComposing: true, bubbles: true }))
    el.dispatchEvent(new CompositionEvent('compositionend', { data: '补充', bubbles: true }))
    el.dispatchEvent(new InputEvent('input', { data: '补充', bubbles: true }))
  })
  const text = '第一页\f已有正文' + '\n'.repeat(15) + '补充\f后页正文'
  await expect(input(page)).toHaveValue(text)
  await expect.poll(async () => (await document(page)).pages[1].cells.at(-1)?.y).toBe(225)
  const edited = await document(page)
  expect(edited.pages).toHaveLength(3)
  expect(edited.pages[0].strokes).toEqual(original.pages[0].strokes)
  expect(edited.pages[2].strokes).toEqual(original.pages[2].strokes)
  await page.screenshot({ path: test.info().outputPath('partial-page-blank-caret.png') })
  await page.keyboard.press('Meta+z')
  await expect(input(page)).toHaveValue(originalText)
  await page.keyboard.press('Meta+Shift+z')
  await expect(input(page)).toHaveValue(text)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.text)).toBe(text)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect.poll(async () => (await document(page)).id).toBe(edited.id)
  await page.locator('.page-rail button').nth(1).click()
  await page.getByRole('button', { name: '导出', exact: true }).click()
  await page.getByRole('button', { name: /导出当前页/ }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect((await page.evaluate(() => window.__tables.exported.document)).pages[1].strokes).toEqual(
    edited.pages[1].strokes,
  )
  expect(errors).toEqual([])
})

test('selection extending into unused rows replaces only the selected text without blank padding', async ({
  page,
}) => {
  const errors = await seed(page, {}, '正文\f后页')
  await clickRow(page, 0, 0)
  await page.keyboard.down('Shift')
  await clickRow(page, 7)
  await page.keyboard.up('Shift')
  await expect
    .poll(() => input(page).evaluate((el) => [el.selectionStart, el.selectionEnd]))
    .toEqual([0, 2])
  await page.keyboard.insertText('替换')
  await expect(input(page)).toHaveValue('替换\f后页')
  await page.keyboard.press('Meta+z')
  await expect(input(page)).toHaveValue('正文\f后页')
  expect(errors).toEqual([])
})

test('A4 with tight line spacing selects the final row even when the font exceeds the row pitch', async ({
  page,
}) => {
  const errors = await seed(page, {
    paperWidth: 210,
    paperHeight: 297,
    marginTop: 25.4,
    marginBottom: 25.4,
    marginLeft: 25.4,
    marginRight: 25.4,
    lineCount: 30,
    lineSpacing: 85,
    letterSpacing: 76.5,
    fontSize: 8.206666666666667,
  })
  await expect.poll(async () => (await document(page)).settings.paperWidth).toBe(210)
  const rows = (await document(page)).pages[0].lineCount
  const y = await clickRow(page, rows - 1)
  await expectCaret(page, 25.4, y)
  await page.keyboard.insertText('末行')
  await expect(input(page)).toHaveValue('\n'.repeat(rows - 1) + '末行')
  await expect.poll(async () => (await document(page)).pages[0].cells?.[0]?.y).toBeCloseTo(y, 8)
  expect((await document(page)).pages).toHaveLength(1)
  expect(errors).toEqual([])
})

test('typing at the last blank row survives undo, redo, save, reopen and export', async ({
  page,
}) => {
  const errors = await seed(page)
  const y = await clickRow(page, 15)
  await page.keyboard.insertText('末行')
  const text = '\n'.repeat(15) + '末行'
  await expect(input(page)).toHaveValue(text)
  await expect.poll(async () => (await document(page)).pages[0].cells?.[0]?.y).toBe(y)
  const preview = await document(page)
  expect(preview.pages).toHaveLength(1)
  expect(preview.pages[0].cells[0].x).toBe(0)
  await page.keyboard.press('Meta+z')
  await expect(input(page)).toHaveValue('')
  await page.keyboard.press('Meta+Shift+z')
  await expect(input(page)).toHaveValue(text)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.text)).toBe(text)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect.poll(async () => (await document(page)).id).toBe(preview.id)
  await page.getByRole('button', { name: '导出', exact: true }).click()
  const exportPage = page.getByRole('button', { name: /导出当前页/ })
  await expect(exportPage).toBeEnabled()
  await exportPage.click()
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect(await page.evaluate(() => window.__tables.exported.document.pages)).toEqual(preview.pages)
  expect(errors).toEqual([])
})

test('Chinese composition and cancellation preserve the selected blank row', async ({ page }) => {
  const errors = await seed(page)
  const y = await clickRow(page, 7)
  await input(page).dispatchEvent('compositionstart', { data: '' })
  await input(page).dispatchEvent('compositionend', { data: '' })
  await expectCaret(page, 0, y)
  await input(page).evaluate((el) => {
    el.dispatchEvent(new CompositionEvent('compositionstart', { bubbles: true }))
    el.dispatchEvent(
      new KeyboardEvent('keydown', { key: 'Process', isComposing: true, bubbles: true }),
    )
    el.value = '测试'
    el.setSelectionRange(2, 2)
    el.dispatchEvent(new InputEvent('input', { data: '测试', isComposing: true, bubbles: true }))
    el.dispatchEvent(new CompositionEvent('compositionend', { data: '测试', bubbles: true }))
    // Some WebViews send one final input event after compositionend.
    el.dispatchEvent(new InputEvent('input', { data: '测试', bubbles: true }))
  })
  await expect(input(page)).toHaveValue('\n'.repeat(7) + '测试')
  await expect.poll(async () => (await document(page)).pages[0].cells?.[0]?.y).toBe(y)
  await page.keyboard.press('Meta+z')
  await expect(input(page)).toHaveValue('')
  expect(errors).toEqual([])
})

test('native paste inserts text at the clicked row and deleting all text restores blank-row selection', async ({
  page,
}) => {
  const errors = await seed(page)
  await page.evaluate(() => {
    const source = document.createElement('textarea')
    source.value = '粘贴文字'
    document.body.append(source)
    source.select()
    document.execCommand('copy')
    source.remove()
  })
  const y = await clickRow(page, 5)
  await page.keyboard.press('Meta+v')
  await expect(input(page)).toHaveValue('\n'.repeat(5) + '粘贴文字')
  await expect.poll(async () => (await document(page)).pages[0].cells?.[0]?.y).toBe(y)
  await page.keyboard.press('Meta+a')
  await page.keyboard.press('Backspace')
  await expect(input(page)).toHaveValue('')
  await expect.poll(async () => (await document(page)).characterCount).toBe(0)
  await clickRow(page, 3)
  await page.keyboard.type('AB')
  await expect(input(page)).toHaveValue('\n\n\nAB')
  await expect.poll(async () => (await document(page)).pages[0].cells?.[0]?.y).toBe(45)
  expect(errors).toEqual([])
})

test('blank-row coordinates follow margins, indentation, random rows and paragraph spacing', async ({
  page,
}) => {
  const errors = await seed(page, {
    lineMode: 'random',
    lineCount: 22,
    lineSpacing: 85,
    marginLeft: 15,
    marginRight: 15,
    marginTop: 20,
    marginBottom: 20,
    indent: 2,
    letterSpacing: 80,
    paragraphGap: 2,
  })
  await expect.poll(async () => (await document(page)).settings.marginTop).toBe(20)
  await page.getByLabel('页面缩放').selectOption('100')
  const y = await clickRow(page, 5, 110)
  await expectCaret(page, 24.6, y)
  await page.keyboard.insertText('定位')
  await expect.poll(async () => (await document(page)).pages[0].cells?.[0]?.y).toBeCloseTo(y, 8)
  expect((await document(page)).pages[0].cells[0].x).toBeCloseTo(24.6, 8)
  expect(errors).toEqual([])
})
