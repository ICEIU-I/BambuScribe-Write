import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

test('new project saves an active cell, clears every object and starts a fresh undo history', async ({
  page,
}, info) => {
  const errors = await open(page)
  await page.evaluate(() => {
    const canvas = document.createElement('canvas')
    canvas.width = 100
    canvas.height = 60
    const context = canvas.getContext('2d')
    context.fillStyle = 'white'
    context.fillRect(0, 0, 100, 60)
    context.strokeRect(5, 5, 90, 50)
    const table = {
      id: 'data',
      page: 0,
      x: 20,
      y: 40,
      columnWidths: [60],
      rowHeights: [14],
      cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '原始表格' }],
      style: 'grid',
      fontSize: 5,
      padding: 1,
      align: 'left',
      verticalAlign: 'middle',
    }
    window.__tables.saved = {
      ...window.__tables.preferences,
      text: '这是一份已有项目。\n'.repeat(24),
      images: [
        {
          id: 'image',
          page: 0,
          x: 90,
          y: 40,
          width: 40,
          height: 24,
          mode: 'line',
          data: canvas.toDataURL(),
          textRegions: [],
        },
      ],
      tables: [table, { ...table, id: 'caption', kind: 'textbox', y: 90, style: 'none' }],
      erasures: [{ id: 'erase', page: 0, x: 130, y: 180, width: 2, height: 2 }],
    }
    window.__tables.menu['menu:action']('open')
  })
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.objectLayout?.length))
    .toBe(3)
  await expect.poll(() => page.locator('.page-rail button').count()).toBeGreaterThan(1)
  await page.getByLabel('页面缩放').selectOption('75')
  await page.locator('[data-table="data"] [data-table-cell="0:0"]').dblclick()
  const input = page.locator('textarea.table-cell-input')
  await input.fill('尚未失焦的单元格')
  await page.evaluate(() => {
    window.go.application.App.ConfirmDiscardProject = async () => 'save'
    window.__tables.menu['menu:action']('new')
  })
  await expect(page.locator('textarea.paper-input')).toHaveValue('')
  await expect.poll(() => page.evaluate(() => window.__tables.payload?.text)).toBe('')
  const payload = await page.evaluate(() => window.__tables.payload)
  expect(payload.images).toEqual([])
  expect(payload.tables).toEqual([])
  expect(payload.erasures).toEqual([])
  const previous = await page.evaluate(() => window.__tables.saved)
  expect(previous.images).toHaveLength(1)
  expect(previous.tables).toHaveLength(2)
  expect(previous.erasures).toHaveLength(1)
  expect(previous.tables[0].cells[0].text).toBe('尚未失焦的单元格')
  await expect(page.locator('.page-rail button')).toHaveCount(1)
  await expect(page.locator('[data-table], .image-overlay')).toHaveCount(0)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('textarea.paper-input')).toHaveValue('')
  await page.locator('textarea.paper-input').fill('新项目内容')
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('textarea.paper-input')).toHaveValue('')
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-table]')).toHaveCount(0)
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  await expect(page.getByLabel('模板名称', { exact: true })).toHaveCount(0)
  await page.screenshot({
    path: `/tmp/bambuscribe-new-project-${info.project.name}.png`,
    animations: 'disabled',
  })
  expect(errors).toEqual([])
})
