import { expect } from '@playwright/test'
import { tmpdir } from 'node:os'
import { test, open, insert, table, waitText, cell, clipboard } from './tableHarness.mjs'

test('inserts presets with real font, edits cells and shares preview paths with export', async ({
  page,
}) => {
  const errors = await open(page)
  await insert(page, { style: 'three-line' })
  await (await cell(page)).dblclick()
  await expect(page.getByLabel('单元格文字')).toBeFocused()
  await page.getByLabel('单元格文字').fill('实验')
  await page.keyboard.press('Tab')
  await expect(page.getByLabel('单元格文字')).toBeFocused()
  await page.getByLabel('单元格文字').fill('10⁻¹⁹')
  await page.keyboard.press('Enter')
  await waitText(page, '实验')
  await waitText(page, '10⁻¹⁹', 0, 1)
  await expect
    .poll(() =>
      page.evaluate(() => window.__tables.document?.pages[0]?.tables[0]?.cells[0]?.strokes.length),
    )
    .toBeGreaterThan(0)
  await expect
    .poll(() =>
      page.evaluate(() => {
        const table = window.__tables.document.pages[0].tables[0]
        const path = table.strokes
          .map((s) => s.map((p, i) => `${i ? 'L' : 'M'}${p.x},${p.y}`).join(' '))
          .join(' ')
        return document.querySelector('.table-ink path').getAttribute('d') === path
      }),
    )
    .toBe(true)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  const result = await page.evaluate(() => window.__tables.exported)
  expect(result.document.pages[0].strokes.length).toBeGreaterThan(3)
  expect(errors).toEqual([])
  await page.screenshot({ path: `${tmpdir()}/bambuscribe-table-first.png` })
})
test('merges and splits rectangular selections, changes row/column counts, and undoes every operation', async ({
  page,
}) => {
  const errors = await open(page)
  await insert(page)
  await (await cell(page)).click()
  await (await cell(page, 0, 1)).click({ modifiers: ['Shift'] })
  await page.getByRole('button', { name: '合并', exact: true }).click()
  await expect.poll(async () => (await table(page)).cells.length).toBe(8)
  expect((await table(page)).cells.find((c) => c.row === 0 && c.col === 0).colSpan).toBe(2)
  await page.getByRole('button', { name: '拆分', exact: true }).click()
  await expect.poll(async () => (await table(page)).cells.length).toBe(9)
  await page.getByRole('button', { name: '表格设置', exact: true }).click()
  await page.getByRole('button', { name: '下方插入行' }).click()
  await expect.poll(async () => (await table(page)).rowHeights.length).toBe(4)
  await page.getByRole('button', { name: '右侧插入列' }).click()
  await expect.poll(async () => (await table(page)).columnWidths.length).toBe(4)
  await page.getByRole('button', { name: '关闭表格设置' }).click()
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(async () => (await table(page)).columnWidths.length).toBe(3)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(async () => (await table(page)).rowHeights.length).toBe(3)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect.poll(async () => (await table(page)).rowHeights.length).toBe(4)
  expect(errors).toEqual([])
})
test('Excel TSV preserves quoted multiline cells and blank columns; cell paste is independent of prose cleanup', async ({
  page,
}) => {
  await open(page)
  await clipboard(page, 'paste', '名称\t数值\t\n"光源\n波长"\t546.1\tA')
  await expect(page.locator('[data-table]')).toHaveCount(1)
  await waitText(page, '光源\n波长', 1, 0)
  const asset = await table(page)
  expect(asset.columnWidths.length).toBe(3)
  expect(asset.cells[2].text).toBe('')
  await (await cell(page, 0, 0)).click()
  await clipboard(page, 'paste', '参数  单位\t10\n电流\t2', '.table-overlay')
  await waitText(page, '参数  单位')
  await waitText(page, '电流', 1, 0)
  expect(await page.evaluate(() => window.__tables.payload.text)).toBe('')
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await waitText(page, '名称')
})
test('column and whole-table resizing maximize font size and cancel without history changes', async ({
  page,
}) => {
  await open(page)
  await insert(page)
  await (await cell(page)).click()
  const original = await table(page),
    handle = page.getByRole('button', { name: '调整第 1 列宽度' }),
    rect = await handle.boundingBox()
  await page.mouse.move(rect.x + 5, rect.y + 5)
  await page.mouse.down()
  await page.mouse.move(rect.x + 35, rect.y + 5, { steps: 5 })
  await page.mouse.up()
  await expect
    .poll(async () => (await table(page)).columnWidths[0])
    .toBeGreaterThan(original.columnWidths[0])
  const changed = await table(page)
  expect(changed.columnWidths.reduce((a, b) => a + b, 0)).toBeCloseTo(105, 5)
  expect(changed.fontSize).toBeGreaterThanOrEqual(original.fontSize)
  const size = await page.getByRole('button', { name: '调整表格大小' }).boundingBox()
  await page.mouse.move(size.x + 5, size.y + 5)
  await page.mouse.down()
  await page.mouse.move(size.x + 40, size.y + 25, { steps: 5 })
  await page.mouse.up()
  await expect
    .poll(async () => (await table(page)).columnWidths.reduce((a, b) => a + b, 0))
    .toBeGreaterThan(105)
  expect((await table(page)).fontSize).toBeLessThanOrEqual(6)
  const before = await table(page),
    move = await page.getByRole('button', { name: '移动表格' }).boundingBox()
  await page.mouse.move(move.x + 10, move.y + 10)
  await page.mouse.down()
  await page.mouse.move(move.x + 40, move.y + 60, { steps: 5 })
  await page.keyboard.press('Escape')
  await page.mouse.up()
  expect(await table(page)).toEqual(before)
})
test('moves to another page, copies complete tables, and keeps tables when body text is deleted', async ({
  page,
}) => {
  await open(page)
  await insert(page)
  await (await cell(page)).click()
  await page.getByRole('button', { name: '表格设置', exact: true }).click()
  await page.getByLabel('表格页码').fill('2')
  await page.getByLabel('表格页码').press('Tab')
  await expect.poll(async () => (await table(page)).page).toBe(1)
  await page.getByRole('button', { name: '关闭表格设置' }).click()
  await (await cell(page)).click()
  await page.getByRole('button', { name: '移动表格' }).click()
  const copied = await page.evaluate(() => {
    const data = new DataTransfer()
    document
      .querySelector('.table-overlay')
      .dispatchEvent(
        new ClipboardEvent('copy', { bubbles: true, cancelable: true, clipboardData: data }),
      )
    return {
      text: data.getData('text/plain'),
      mime: data.getData('application/x-bambuscribe-table-v1'),
    }
  })
  expect(copied.mime).toBeTruthy()
  await clipboard(page, 'paste', copied.text, '.document-viewport', copied.mime)
  await expect(page.locator('[data-table]')).toHaveCount(2)
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(2)
  expect((await table(page, 1)).id).not.toBe((await table(page)).id)
  await page.evaluate(() => {
    const input = document.querySelector('.paper-input')
    input.value = '正文'
    input.dispatchEvent(new Event('input', { bubbles: true }))
  })
  await expect.poll(() => page.evaluate(() => window.__tables.payload.text)).toBe('正文')
  await page.evaluate(() => {
    const input = document.querySelector('.paper-input')
    input.value = ''
    input.dispatchEvent(new Event('input', { bubbles: true }))
  })
  await expect.poll(() => page.evaluate(() => window.__tables.payload.text)).toBe('')
  await expect(page.locator('[data-table]')).toHaveCount(2)
  await expect.poll(() => page.evaluate(() => window.__tables.document.pages.length)).toBe(2)
})
test('project saving/opening retains editable topology, border changes, and the workspace language', async ({
  page,
}) => {
  await open(page)
  await insert(page)
  await (await cell(page)).dblclick()
  await page.getByLabel('单元格文字').fill('数据')
  await page.keyboard.press('Enter')
  await waitText(page, '数据')
  await page.getByRole('button', { name: '表格设置', exact: true }).click()
  await page.getByLabel('单元格边框').selectOption('none')
  await expect.poll(async () => (await table(page)).cells[0].borders?.top).toBe(false)
  await page.getByRole('button', { name: '关闭表格设置' }).click()
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved?.version))
    .toBeGreaterThanOrEqual(5)
  const saved = await table(page)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-table]')).toHaveCount(1)
  expect(await table(page)).toEqual(saved)
  await page.locator('[data-table]').click()
  await page.getByRole('button', { name: '删除表格' }).click()
  await expect(page.locator('[data-table]')).toHaveCount(0)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-table]')).toHaveCount(0)
  await expect.poll(() => page.evaluate(() => window.__tables.saved.tables.length)).toBe(0)
})
test('multiline rows grow, page overflow moves the table intact and missing glyphs block export', async ({
  page,
}) => {
  const errors = await open(page)
  await insert(page, { width: 80, cols: 2 })
  await (await cell(page)).dblclick()
  await page.getByLabel('单元格文字').fill('实验数据\n保持字大小')
  await page.keyboard.press('Enter')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].rowHeights[0]))
    .toBeGreaterThan(10)
  await page.getByRole('button', { name: '表格设置', exact: true }).click()
  await page.getByLabel('表格位置 Y').fill('230')
  await page.getByLabel('表格位置 Y').press('Tab')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[1]?.tables?.length))
    .toBe(1)
  await expect(page.locator('[data-table]')).toHaveCount(1)
  await expect(page.locator('[data-paper="1"] [data-table]')).toHaveCount(1)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  await page.getByLabel('表格位置 Y').fill('20')
  await page.getByLabel('表格位置 Y').press('Tab')
  await page.getByRole('button', { name: '关闭表格设置' }).click()
  await (await cell(page)).dblclick()
  await page.getByLabel('单元格文字').fill('🙂')
  await page.keyboard.press('Enter')
  await expect(page.locator('.document-alert[role="alert"]')).toContainText('缺字')
  expect(errors).toEqual([])
})

test('cell input and handwriting only break lines at Shift+Enter', async ({ page }) => {
  const errors = await open(page)
  await insert(page, { cols: 2, width: 40 })
  const original = await table(page)
  await (await cell(page)).dblclick()
  const input = page.getByLabel('单元格文字')
  await expect(input).toHaveAttribute('wrap', 'off')
  await input.fill('实验数据需要多行显示')
  await page.keyboard.press('Enter')
  await waitText(page, '实验数据需要多行显示')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].rowHeights))
    .toEqual(original.rowHeights)
  await expect(page.locator('.document-alert[role="alert"]')).toHaveCount(0)
  const fittedSize = await page.evaluate(() => window.__tables.document.pages[0].tables[0].fontSize)
  expect(fittedSize).toBeLessThan(original.fontSize)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  await (await cell(page)).dblclick()
  await input.fill('实验')
  await page.keyboard.press('Shift+Enter')
  await page.keyboard.insertText('数据')
  await page.keyboard.press('Enter')
  await waitText(page, '实验\n数据')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].rowHeights))
    .toEqual(original.rowHeights)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].fontSize))
    .toBeGreaterThan(fittedSize)
  await expect(page.locator('.document-alert[role="alert"]')).toHaveCount(0)
  expect(errors).toEqual([])
})

test('drag crosses a page boundary with immediate vector preview and one undo step', async ({
  page,
}) => {
  const errors = await open(page)
  await insert(page, { rows: 2, cols: 2, width: 75 })
  await (await cell(page)).click()
  await page.getByRole('button', { name: '表格设置', exact: true }).click()
  await page.getByLabel('表格页码').fill('2')
  await page.getByLabel('表格页码').press('Tab')
  await expect.poll(async () => (await table(page)).page).toBe(1)
  await page.getByRole('button', { name: '关闭表格设置' }).click()
  await page.getByLabel('页面缩放').selectOption('50')
  await page.locator('.document-viewport').evaluate((view) => {
    const paper = view.querySelector('[data-paper="1"]')
    view.scrollTop +=
      paper.getBoundingClientRect().top - view.getBoundingClientRect().top - view.clientHeight * 0.6
  })
  await (await cell(page)).click()
  const before = await table(page),
    handle = await page.getByRole('button', { name: '移动表格' }).boundingBox(),
    target = await page.locator('[data-paper="0"]').evaluate((el) => {
      const r = el.getBoundingClientRect(),
        k = r.width / 163
      return { x: r.left + 30 * k, y: r.top + 210 * k }
    })
  await page.mouse.move(handle.x + 10, handle.y + 10)
  await page.mouse.down()
  await page.mouse.move(target.x, target.y, { steps: 5 })
  expect((await table(page)).page).toBe(1)
  await expect(page.locator('.table-ink path')).toHaveAttribute('d', /M.+L/)
  await page.mouse.up()
  await expect.poll(async () => (await table(page)).page).toBe(0)
  expect((await table(page)).fontSize).toBe(before.fontSize)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(async () => (await table(page)).page).toBe(1)
  expect(await table(page)).toEqual(before)
  expect(errors).toEqual([])
})
test('HTML clipboard fallback preserves table shape when custom system types are stripped', async ({
  page,
}) => {
  await open(page)
  await insert(page)
  await (await cell(page)).click()
  await (await cell(page, 0, 1)).click({ modifiers: ['Shift'] })
  await page.getByRole('button', { name: '合并', exact: true }).click()
  await expect.poll(async () => (await table(page)).cells.length).toBe(8)
  await page.getByRole('button', { name: '移动表格' }).click()
  const content = await page.evaluate(() => {
    const data = new DataTransfer()
    document
      .querySelector('.table-overlay')
      .dispatchEvent(
        new ClipboardEvent('copy', { bubbles: true, cancelable: true, clipboardData: data }),
      )
    return { text: data.getData('text/plain'), html: data.getData('text/html') }
  })
  await page.evaluate((content) => {
    const data = new DataTransfer()
    data.setData('text/plain', content.text)
    data.setData('text/html', content.html)
    document
      .querySelector('.document-viewport')
      .dispatchEvent(
        new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: data }),
      )
  }, content)
  await expect(page.locator('[data-table]')).toHaveCount(2)
  await expect
    .poll(() =>
      page.evaluate(
        () =>
          window.__tables.payload.tables[1]?.cells.find((c) => c.row === 0 && c.col === 0)?.colSpan,
      ),
    )
    .toBe(2)
})

test('saving from a cell shortcut or native menu commits the active edit before serializing', async ({
  page,
}) => {
  await open(page)
  await insert(page)
  await (await cell(page)).dblclick()
  await page.getByLabel('单元格文字').fill('尚未失焦的文字')
  await page.keyboard.press('Meta+s')
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved?.tables?.[0].cells[0].text))
    .toBe('尚未失焦的文字')
  await (await cell(page)).dblclick()
  await page.getByLabel('单元格文字').fill('菜单保存的文字')
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved?.tables?.[0].cells[0].text))
    .toBe('菜单保存的文字')
})

test('trailing spaces do not grow merged cells or shift centered handwriting', async ({ page }) => {
  const errors = await open(page)
  await insert(page, { rows: 4, cols: 3, width: 60 })
  const original = await table(page)
  await (await cell(page)).click()
  await (await cell(page, 1, 0)).click({ modifiers: ['Shift'] })
  await page.getByRole('button', { name: '合并', exact: true }).click()
  await expect.poll(async () => (await table(page)).cells[0].rowSpan).toBe(2)
  await (await cell(page)).dblclick()
  await page.getByLabel('单元格文字').fill('435.8nm       \n光阑2mm')
  await page.keyboard.press('Enter')
  await waitText(page, '435.8nm       \n光阑2mm')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].rowHeights))
    .toEqual(original.rowHeights)
  const before = await page.locator('.table-ink path').getAttribute('d')
  await (await cell(page)).dblclick()
  expect(await page.getByLabel('单元格文字').inputValue()).toBe('435.8nm       \n光阑2mm')
  await page.getByLabel('单元格文字').fill('435.8nm\n光阑2mm')
  await page.keyboard.press('Enter')
  await waitText(page, '435.8nm\n光阑2mm')
  await expect(page.locator('.table-ink path')).toHaveAttribute('d', before)
  expect(errors).toEqual([])
})

test('moving, copying, changing columns and settings keep automatic row growth temporary', async ({
  page,
}) => {
  const errors = await open(page)
  await insert(page, { cols: 2, width: 40 })
  const original = await table(page)
  await (await cell(page)).dblclick()
  await page.getByLabel('单元格文字').fill(Array(20).fill('实验').join('\n'))
  await page.keyboard.press('Enter')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].rowHeights[0]))
    .toBeGreaterThan(original.rowHeights[0])
  const handle = await page.getByRole('button', { name: '移动表格' }).boundingBox()
  await page.mouse.move(handle.x + 10, handle.y + 10)
  await page.mouse.down()
  await page.mouse.move(handle.x + 30, handle.y + 30, { steps: 5 })
  await page.mouse.up()
  await expect.poll(async () => (await table(page)).x).toBeGreaterThan(original.x)
  expect((await table(page)).rowHeights).toEqual(original.rowHeights)
  const copied = await page.evaluate(() => {
    const data = new DataTransfer()
    document
      .querySelector('.table-overlay')
      .dispatchEvent(
        new ClipboardEvent('copy', { bubbles: true, cancelable: true, clipboardData: data }),
      )
    return JSON.parse(data.getData('application/x-bambuscribe-table-v1'))
  })
  expect(copied.rowHeights).toEqual(original.rowHeights)
  const column = await page.getByRole('button', { name: '调整第 1 列宽度' }).boundingBox()
  await page.mouse.move(column.x + 5, column.y + 5)
  await page.mouse.down()
  await page.mouse.move(column.x + 15, column.y + 5, { steps: 5 })
  await page.mouse.up()
  await expect.poll(async () => (await table(page)).columnWidths[0]).toBeGreaterThan(20)
  expect((await table(page)).rowHeights).toEqual(original.rowHeights)
  await page.getByRole('button', { name: '表格设置', exact: true }).click()
  await page.getByLabel('表格位置 X').fill('15')
  await page.getByLabel('表格位置 X').press('Tab')
  await expect.poll(async () => (await table(page)).x).toBe(15)
  expect((await table(page)).rowHeights).toEqual(original.rowHeights)
  await page.getByRole('button', { name: '关闭表格设置' }).click()
  await (await cell(page)).dblclick()
  await page.getByLabel('单元格文字').fill('实验')
  await page.keyboard.press('Enter')
  await waitText(page, '实验')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].rowHeights))
    .toEqual(expect.arrayContaining([expect.any(Number)]))
  expect(
    (await page.evaluate(() => window.__tables.document.pages[0].tables[0].rowHeights)).every(
      (h, i) => h <= original.rowHeights[i] + 1e-7,
    ),
  ).toBe(true)
  expect(errors).toEqual([])
})
