import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'
export const rail = (page) => page.locator('.page-rail button')
export const payload = (page) => page.evaluate(() => window.__tables.payload)
export const doc = (page) => page.evaluate(() => window.__tables.document)
export async function seed(
  page,
  text = '第一页\n'.repeat(4) + '第二页\n'.repeat(4) + '第三页',
  layout = {},
  objects = false,
) {
  const errors = await open(page)
  await page.evaluate(
    ({ text, layout, objects }) => {
      const table = (id, page, y, kind) => ({
        id,
        page,
        kind,
        groupId: 'original-group',
        x: 100,
        y,
        columnWidths: [40],
        rowHeights: [12],
        cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '表格' }],
        fontSize: 5,
        padding: 1,
        align: 'left',
        verticalAlign: 'middle',
        style: kind ? 'none' : 'grid',
      })
      const canvas = document.createElement('canvas')
      canvas.width = 30
      canvas.height = 30
      const ctx = canvas.getContext('2d')
      ctx.fillStyle = 'white'
      ctx.fillRect(0, 0, 30, 30)
      ctx.strokeRect(2, 2, 26, 26)
      window.__tables.saved = {
        ...window.__tables.preferences,
        layout: { ...window.__tables.preferences.layout, lineCount: 4, ...layout },
        text,
        images: objects
          ? [
              {
                id: 'source-image',
                page: 1,
                x: 100,
                y: 110,
                width: 25,
                height: 25,
                rotation: 0,
                data: canvas.toDataURL(),
                mode: 'line',
                textRegions: [],
              },
            ]
          : [],
        tables: objects
          ? [table('grid', 1, 30), table('caption', 1, 55, 'textbox'), table('later', 2, 30)]
          : [],
        erasures: objects ? [{ id: 'erase', page: 1, x: 110, y: 120, width: 2, height: 2 }] : [],
      }
      window.__tables.menu['menu:action']('open')
    },
    { text, layout, objects },
  )
  await expect.poll(async () => (await doc(page))?.settings.lineCount).toBe(4)
  await page.getByLabel('页面缩放').selectOption('50')
  return errors
}
export { test }
export const ink = (page) =>
  page.strokes.map((s) => s.map((p) => [Math.round(p.x * 1e7), Math.round(p.y * 1e7)]))
