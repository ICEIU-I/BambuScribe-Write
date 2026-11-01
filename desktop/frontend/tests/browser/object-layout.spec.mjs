import { expect } from '@playwright/test'
import { tmpdir } from 'node:os'
import { test, open } from './tableHarness.mjs'

async function seed(page) {
  const errors = await open(page)
  await page.evaluate(() => {
    const c = document.createElement('canvas')
    c.width = 200
    c.height = 100
    const ctx = c.getContext('2d')
    ctx.fillStyle = 'white'
    ctx.fillRect(0, 0, 200, 100)
    ctx.strokeStyle = 'black'
    ctx.lineWidth = 2
    ctx.strokeRect(0, 0, 200, 100)
    const table = (id, y, height, kind) => ({
      id,
      page: 0,
      x: 20,
      y,
      kind,
      columnWidths: [80],
      rowHeights: [height],
      cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: kind ? '实验图表' : '实验数据' }],
      style: kind ? 'none' : 'grid',
      fontSize: 6,
      padding: 1,
      align: 'center',
      verticalAlign: 'middle',
    })
    window.__tables.saved = {
      ...window.__tables.preferences,
      text: '正文文字保持完整。\n'.repeat(24),
      images: [
        {
          id: 'a-image',
          page: 0,
          x: 20,
          y: 15,
          width: 80,
          height: 40,
          rotation: 15,
          mode: 'line',
          data: c.toDataURL(),
          textRegions: [],
        },
      ],
      tables: [table('b-table', 80, 40), table('c-caption', 15, 12, 'textbox')],
    }
    window.__tables.menu['menu:action']('open')
  })
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.objectLayout?.length))
    .toBe(3)
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  return errors
}

async function assertLayout(page) {
  await expect
    .poll(() =>
      page.evaluate(() => {
        const doc = window.__tables.document
        if (!doc?.objectLayout?.length) return false
        const overlap = (a, b) =>
          a.x < b.x + b.width && b.x < a.x + a.width && a.y < b.y + b.height && b.y < a.y + a.height
        for (const p of doc.pages) {
          const bodyObjects = [
            ...(p.tables ?? []),
            ...(p.images ?? []).flatMap((im) => {
              const points = im.strokes.flat()
              if (!points.length) return []
              const xs = points.map((q) => q.x),
                ys = points.map((q) => q.y)
              const x = Math.min(...xs),
                y = Math.min(...ys)
              return [{ x, y, width: Math.max(...xs) - x, height: Math.max(...ys) - y }]
            }),
          ]
          for (const object of bodyObjects) {
            if (p.cells.some((c) => overlap(c, object))) return false
          }
        }
        return doc.pages.map((p) => p.text).join('') === window.__tables.payload.text
      }),
    )
    .toBe(true)
  await expect
    .poll(() =>
      page.evaluate(() => {
        const doc = window.__tables.document
        for (const p of doc.objectLayout) {
          const el = document.querySelector(
            `[data-table="${p.id}"], [data-free-text-box="${p.id}"], [data-image-id="${p.id}"]`,
          )
          const paper = el?.closest('[data-paper]')
          if (!paper || Number(paper.dataset.paper) !== p.page) return p.id
          const k = paper.getBoundingClientRect().width / doc.settings.paperWidth
          if (
            Math.abs(parseFloat(el.style.left) / k - p.x) > 0.001 ||
            Math.abs(parseFloat(el.style.top) / k - p.y) > 0.001
          )
            return p.id
        }
        return ''
      }),
    )
    .toBe('')
}

test('body avoids overlapping objects; moving and undo restore matching export geometry', async ({
  page,
}) => {
  const errors = await seed(page)
  await assertLayout(page)
  // Body rows remain continuous below each object's ink envelope.
  const interrupted = await page.evaluate(() => {
    const p = window.__tables.document.pages[0]
    const points = p.images[0].strokes.flat()
    const top = Math.min(...points.map((q) => q.y))
    const bottom = Math.max(...points.map((q) => q.y))
    return p.cells.some((c) => c.y < bottom + 1 && c.y + c.height > top - 1)
  })
  expect(interrupted).toBe(false)
  await page.locator('[data-table="b-table"] [data-table-cell="0:0"]').click()
  await page.getByRole('button', { name: '表格设置', exact: true }).click()
  await page.getByLabel('表格位置 Y').fill('230')
  await page.getByLabel('表格位置 Y').press('Tab')
  await expect(page.locator('[data-paper="1"] [data-table="b-table"]')).toHaveCount(1)
  await assertLayout(page)
  await page.getByRole('button', { name: '关闭表格设置' }).click()
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-paper="0"] [data-table="b-table"]')).toHaveCount(1)
  await assertLayout(page)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(page.locator('[data-paper="1"] [data-table="b-table"]')).toHaveCount(1)
  await assertLayout(page)
  await page.screenshot({
    path: `${tmpdir()}/bambuscribe-exclusive-layout-${test.info().project.name}.png`,
  })
  expect(errors).toEqual([])
})

test('deleting objects reclaims body space; save and reopen preserve all text and requested anchors', async ({
  page,
}) => {
  const errors = await seed(page)
  const before = await page.evaluate(() => window.__tables.document)
  await page.locator('[data-table="b-table"] [data-table-cell="0:0"]').click()
  await page.getByRole('button', { name: '删除表格', exact: true }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.objectLayout?.length))
    .toBe(2)
  await assertLayout(page)
  const after = await page.evaluate(() => window.__tables.document)
  const cell = after.pages[0].cells.find((c) => {
    const earlier = before.pages[0].cells.find((old) => old.start === c.start)
    return earlier && c.y < earlier.y
  })
  expect(cell).toBeDefined()
  expect(after.pages.map((p) => p.text).join('')).toBe(before.pages.map((p) => p.text).join(''))
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.objectLayout?.length))
    .toBe(3)
  await assertLayout(page)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.tables?.length)).toBe(2)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await assertLayout(page)
  expect(await page.evaluate(() => window.__tables.document.objectLayout)).toEqual(
    before.objectLayout,
  )
  expect(errors).toEqual([])
})
