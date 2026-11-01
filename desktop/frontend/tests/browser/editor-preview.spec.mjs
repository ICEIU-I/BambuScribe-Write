import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { test, open } from './tableHarness.mjs'

const data = `data:image/png;base64,${readFileSync(
  new URL('../../../internal/engine/testdata/hall-switch.png', import.meta.url),
).toString('base64')}`

test('精简预览保留正文、图表和导出笔画，关闭的识别提示不随重排重新出现', async ({ page }, info) => {
  const errors = await open(page, {}, true)
  await page.evaluate((data) => {
    window.__tables.saved = {
      ...window.__tables.preferences,
      text: '正文\n四、实验',
      images: [
        {
          id: 'diagram',
          page: 0,
          x: 10,
          y: 45,
          width: 40,
          height: 40,
          rotation: 0,
          data,
          textRegions: [],
          diagram: {
            width: 100,
            height: 100,
            strokes: [
              {
                id: 'line',
                role: 'ray',
                points: [
                  { x: 5, y: 5 },
                  { x: 90, y: 90 },
                ],
              },
            ],
            circles: [],
            labels: [{ text: 'A', x: 20, y: 5, width: 10, height: 15 }],
            uncertainties: ['中心交点为估算位置'],
          },
        },
      ],
      tables: [
        {
          id: 'table',
          page: 0,
          x: 10,
          y: 100,
          fontSize: 6,
          padding: 1,
          columnWidths: [80],
          rowHeights: [15],
          align: 'left',
          verticalAlign: 'middle',
          style: 'grid',
          cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '实验记录' }],
        },
      ],
    }
    window.__tables.menu['menu:action']('open')
  }, data)
  await expect(page.locator('[data-image-id="diagram"] .image-ink path').first()).toHaveAttribute(
    'd',
    /M.+L/,
  )
  await expect(page.locator('[data-table="table"] .table-ink path')).toHaveAttribute('d', /M.+L/)
  await expect(page.locator('.document-alert[role="status"]')).toContainText('中心交点为估算位置')
  const shown = await page.evaluate(() => window.__tables.document)
  expect(await page.evaluate(() => window.__tables.payload.compact)).toBe(true)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect(await page.evaluate(() => window.__tables.exported.document.pages[0].strokes)).toEqual(
    shown.pages[0].strokes,
  )
  await page.getByRole('button', { name: '关闭页面提示', exact: true }).click()
  await expect(page.locator('.document-alert[role="status"]')).toHaveCount(0)
  const input = page.getByRole('textbox', { name: '书写内容', exact: true })
  await input.focus()
  await input.evaluate((el) => {
    el.setSelectionRange(3, 3)
    el.dispatchEvent(new Event('select'))
  })
  await page.keyboard.press('Enter')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].text))
    .toBe('正文\n\n四、实验')
  await expect(page.locator('.document-alert[role="status"]')).toHaveCount(0)
  await page.screenshot({ path: `/tmp/bambuscribe-compact-preview-${info.project.name}.png` })
  expect(errors).toEqual([])
})

test('连续回车合并等待中的重排，始终只运行一次原生预览并显示最新正文', async ({ page }) => {
  const errors = await open(page, {}, true)
  await expect.poll(() => page.evaluate(() => !!window.__tables.document)).toBe(true)
  await expect(page.locator('.loading-paper')).toHaveCount(0)
  await page.evaluate(() => {
    const api = window.go.application.App
    const preview = api.PreviewEditor
    window.__queue = { calls: 0, active: 0, maximum: 0 }
    api.PreviewEditor = async (...args) => {
      const queue = window.__queue
      queue.calls++
      queue.active++
      queue.maximum = Math.max(queue.maximum, queue.active)
      try {
        if (queue.calls === 1)
          await new Promise((resolve) => {
            queue.release = resolve
          })
        return await preview(...args)
      } finally {
        queue.active--
      }
    }
  })
  const input = page.getByRole('textbox', { name: '书写内容', exact: true })
  await input.fill('正文\n四、实验')
  await expect.poll(() => page.evaluate(() => window.__queue.calls)).toBe(1)
  for (let i = 0; i < 4; i++) await page.keyboard.press('Enter')
  await expect(input).toHaveValue('正文\n四、实验\n\n\n\n')
  expect(await page.evaluate(() => window.__queue.calls)).toBe(1)
  await page.evaluate(() => window.__queue.release())
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages.map((p) => p.text).join('')))
    .toBe('正文\n四、实验\n\n\n\n')
  expect(await page.evaluate(() => window.__queue.maximum)).toBe(1)
  expect(await page.evaluate(() => window.__queue.calls)).toBe(2)
  expect(errors).toEqual([])
})
