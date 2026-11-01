import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

async function load(page, tables) {
  await page.evaluate((tables) => {
    window.__tables.saved = {
      version: 4,
      text: '',
      images: [],
      tables,
      writing: window.__tables.preferences.writing,
      layout: {
        ...window.__tables.payload.settings,
        paperWidth: 210,
        paperHeight: 297,
        marginTop: 25.4,
        marginBottom: 25.4,
        marginLeft: 17,
        marginRight: 17,
      },
    }
    window.__tables.menu['menu:action']('open')
  }, tables)
  await expect
    .poll(() => page.evaluate(() => window.__tables.payload.tables.length))
    .toBe(tables.length)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.settings.paperWidth))
    .toBe(210)
}
const make = (id, page, y, rows, width = 176) => ({
  id,
  page,
  x: 22.6,
  y,
  columnWidths: [width],
  rowHeights: Array(rows).fill(3),
  fontSize: 8.2,
  padding: 1,
  align: 'center',
  verticalAlign: 'middle',
  style: 'grid',
  cells: Array.from({ length: rows }, (_, row) => ({
    row,
    col: 0,
    rowSpan: 1,
    colSpan: 1,
    text: String(row),
  })),
})
const assets = (page) => page.evaluate(() => window.__tables.payload.tables)

test('short numbers fill the cell ink area independently of long headings', async ({ page }) => {
  const errors = await open(page)
  const table = make('legible', 0, 40, 3, 22)
  table.rowHeights = [6.78, 6.78, 6.78]
  table.cells[0].text = '表头很长 U_{H}/mV'
  table.cells[1].text = '1000'
  table.cells[2].text = '0.00'
  await load(page, [table])
  const readInk = () =>
    page.evaluate(() => {
      const rendered = window.__tables.document.pages[0].tables[0]
      const expected = rendered.strokes.flat().flatMap((p) => [p.x, p.y])
      const shown = (
        document
          .querySelector('.table-ink path')
          .getAttribute('d')
          .match(/-?\d+(?:\.\d+)?(?:e[+-]?\d+)?/g) ?? []
      ).map(Number)
      return {
        error:
          expected.length === shown.length
            ? Math.max(...expected.map((v, i) => Math.abs(v - shown[i])))
            : Infinity,
        cells: rendered.cells.map((c) => {
          const ys = c.strokes.flat().map((p) => p.y)
          return { size: c.fontSize, height: Math.max(...ys) - Math.min(...ys) }
        }),
      }
    })
  await expect.poll(async () => (await readInk()).error).toBeLessThan(1e-7)
  const initial = await readInk()
  expect(initial.cells[1].size).toBeLessThanOrEqual(6)
  expect(initial.cells[2].size).toBeLessThanOrEqual(6)
  expect(initial.cells[1].size).toBeGreaterThan(initial.cells[0].size)
  await page.locator('[data-table-cell="0:0"]').dblclick()
  await page.getByLabel('单元格文字', { exact: true }).fill('短')
  await page.getByLabel('单元格文字', { exact: true }).press('Enter')
  await expect
    .poll(async () => (await readInk()).cells[0].size)
    .toBeGreaterThan(initial.cells[0].size)
  await expect.poll(async () => (await readInk()).error).toBeLessThan(1e-7)
  for (const [i, c] of (await readInk()).cells.slice(1).entries()) {
    expect(c.size).toBeLessThanOrEqual(6)
    expect(c.height / c.size).toBeCloseTo(
      initial.cells[i + 1].height / initial.cells[i + 1].size,
      7,
    )
  }
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  expect(
    await page.evaluate(() => window.__tables.exported.document.pages[0].tables[0].strokes),
  ).toEqual(await page.evaluate(() => window.__tables.document.pages[0].tables[0].strokes))
  await page.screenshot({ path: `/tmp/bambuscribe-cell-ink-${test.info().project.name}.png` })
  expect(errors).toEqual([])
})

test('third-page group uses the second-page bottom margin, previews captions and undoes atomically', async ({
  page,
}) => {
  const errors = await open(page)
  const title = {
      ...make('title', 2, 5, 1),
      groupId: 'group',
      kind: 'textbox',
      padding: 0.5,
      style: 'none',
    },
    note = {
      ...make('note', 2, 16.2, 1),
      groupId: 'group',
      kind: 'textbox',
      fontSize: 5,
      padding: 0.5,
      style: 'none',
    },
    table = { ...make('moving', 2, 24.2, 12, 175.6), groupId: 'group' }
  await load(page, [make('existing', 1, 36.6, 11, 180), title, note, table])
  await page.getByLabel('页面缩放').selectOption('50')
  const selector = (id) =>
    page.locator(id === 'moving' ? '[data-table="moving"]' : `[data-free-text-box="${id}"]`)
  await selector('moving').locator('[data-table-cell]').first().click()
  async function drag() {
    await page.locator('.document-viewport').evaluate((view) => {
      const paper = view.querySelector('[data-paper="2"]')
      view.scrollTop +=
        paper.getBoundingClientRect().top -
        view.getBoundingClientRect().top -
        view.clientHeight * 0.6
    })
    const before = await Promise.all(
        ['moving', 'title', 'note'].map((id) => selector(id).boundingBox()),
      ),
      handle = await page.getByRole('button', { name: '整组移动', exact: true }).boundingBox(),
      target = await page.locator('[data-paper="1"]').evaluate((el) => {
        const r = el.getBoundingClientRect(),
          k = r.width / 210
        return { x: r.left + 205 * k, y: r.top + 294 * k }
      })
    await page.mouse.move(handle.x + 10, handle.y + 10)
    await page.mouse.down()
    await page.mouse.move(target.x, target.y, { steps: 6 })
    const during = await Promise.all(
      ['moving', 'title', 'note'].map((id) => selector(id).boundingBox()),
    )
    for (let i = 1; i < 3; i++) {
      expect(during[i].x - during[0].x).toBeCloseTo(before[i].x - before[0].x, 1)
      expect(during[i].y - during[0].y).toBeCloseTo(before[i].y - before[0].y, 1)
    }
    expect((await assets(page))[3].page).toBe(2)
  }
  const before = await assets(page)
  await drag()
  await page.keyboard.press('Escape')
  await page.mouse.up()
  expect(await assets(page)).toEqual(before)
  await drag()
  await page.screenshot({ path: `/tmp/bambuscribe-group-drag-${test.info().project.name}.png` })
  await page.mouse.up()
  await expect.poll(async () => (await assets(page))[3].page).toBe(1)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[1]?.tables.length))
    .toBe(4)
  const moved = await assets(page)
  expect(moved[0]).toEqual(before[0])
  for (const t of moved.slice(1)) {
    expect(t.page).toBe(1)
    expect(t.x).toBeCloseTo(34, 5)
    expect(t.fontSize).toBe(before.find((o) => o.id === t.id).fontSize)
  }
  const placed = await page.evaluate(() =>
    window.__tables.document.pages[1].tables.find((t) => t.id === 'moving'),
  )
  expect(placed.y + placed.height).toBeCloseTo(297, 5)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(() => assets(page)).toEqual(before)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect.poll(() => assets(page)).toEqual(moved)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.tables)).toEqual(moved)
  expect(errors).toEqual([])
})

test('height and width resizing maximize text in both directions and persist through export', async ({
  page,
}) => {
  const errors = await open(page)
  const table = make('resize', 0, 40, 2, 130)
  table.rowHeights = [30, 30]
  table.cells[0].text = 'U_{H}/mV'
  table.cells[1].text = '两行\n文字'
  table.fontSize = 2
  await load(page, [table])
  const initialSize = await page.evaluate(
    () => window.__tables.document.pages[0].tables[0].fontSize,
  )
  expect(initialSize).toBeGreaterThanOrEqual(6 * 0.7 - 1e-7)
  expect(initialSize).toBeLessThanOrEqual(6)
  // The visible ink must use the effective native font, even for old projects
  // whose saved font is tiny. Comparing every path point catches re-shrinking.
  const geometryError = await page.evaluate(() => {
    const expected = window.__tables.document.pages[0].tables[0].strokes
      .flat()
      .flatMap((p) => [p.x, p.y])
    const shown = (
      document
        .querySelector('.table-ink path')
        .getAttribute('d')
        .match(/-?\d+(?:\.\d+)?(?:e[+-]?\d+)?/g) ?? []
    ).map(Number)
    return shown.length === expected.length
      ? Math.max(...shown.map((v, i) => Math.abs(v - expected[i])))
      : Infinity
  })
  expect(geometryError).toBeLessThan(1e-7)
  await page.locator('[data-table-cell]').first().click()
  await page.getByRole('button', { name: '表格设置', exact: true }).click()
  await expect(page.getByLabel('表格字号', { exact: true })).toHaveAttribute('readonly', '')
  await expect(page.getByText('文字同步缩放', { exact: true })).toHaveCount(0)
  const height = page.getByLabel('设置表格高度', { exact: true })
  await height.fill('50')
  await height.press('Tab')
  await expect.poll(async () => (await assets(page))[0].rowHeights[0]).toBeLessThanOrEqual(25)
  expect((await assets(page))[0].fontSize).toBeLessThanOrEqual(6)
  await height.fill('20')
  await height.press('Tab')
  await expect.poll(async () => (await assets(page))[0].fontSize).toBeLessThanOrEqual(6)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].height))
    .toBeLessThanOrEqual(20)
  expect((await assets(page))[0].columnWidths[0]).toBeLessThanOrEqual(130)
  const small = (await assets(page))[0]
  await height.fill('60')
  await height.press('Tab')
  await expect.poll(async () => (await assets(page))[0].rowHeights[0]).toBeLessThanOrEqual(30)
  expect((await assets(page))[0].fontSize).toBeLessThanOrEqual(6)
  await page.getByLabel('设置表格宽度', { exact: true }).fill('8')
  await page.getByLabel('设置表格宽度', { exact: true }).press('Tab')
  await expect.poll(async () => (await assets(page))[0].fontSize).toBeLessThan(small.fontSize)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].issues))
    .toEqual([])
  await page.getByRole('button', { name: '关闭表格设置' }).click()
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  const rendered = await page.evaluate(() => window.__tables.exported.document.pages[0].tables[0])
  for (const c of rendered.cells)
    for (const stroke of c.strokes)
      for (const p of stroke) {
        expect(p.x).toBeGreaterThanOrEqual(c.x - 1e-7)
        expect(p.x).toBeLessThanOrEqual(c.x + c.width + 1e-7)
        expect(p.y).toBeGreaterThanOrEqual(c.y - 1e-7)
        expect(p.y).toBeLessThanOrEqual(c.y + c.height + 1e-7)
      }
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved.tables[0].fontSize))
    .toBeCloseTo(rendered.fontSize, 7)
  await page.screenshot({ path: `/tmp/bambuscribe-table-fit-${test.info().project.name}.png` })
  expect(errors).toEqual([])
})

test('dragging the resize handle fits text immediately, supports cancel and one-step undo', async ({
  page,
}) => {
  const errors = await open(page)
  const table = make('resize-drag', 0, 40, 2, 90)
  table.rowHeights = [20, 20]
  table.cells[0].text = 'U_{H}'
  await load(page, [table])
  const initialSize = await page.evaluate(
    () => window.__tables.document.pages[0].tables[0].fontSize,
  )
  await page.getByLabel('页面缩放').selectOption('75')
  await page.locator('[data-table-cell]').first().click()
  const original = await assets(page)
  async function shrink() {
    const handle = await page.getByRole('button', { name: '调整表格大小' }).boundingBox(),
      k = await page
        .locator('[data-paper="0"]')
        .evaluate((el) => el.getBoundingClientRect().width / 210),
      ink = await page.locator('.table-ink path').getAttribute('d')
    await page.mouse.move(handle.x + handle.width / 2, handle.y + handle.height / 2)
    await page.mouse.down()
    await page.mouse.move(handle.x + handle.width / 2, handle.y + handle.height / 2 - 28 * k, {
      steps: 6,
    })
    expect(await page.locator('.table-ink path').getAttribute('d')).not.toBe(ink)
    expect(await assets(page)).toEqual(original)
  }
  await shrink()
  await page.keyboard.press('Escape')
  await page.mouse.up()
  expect(await assets(page)).toEqual(original)
  await shrink()
  await page.mouse.up()
  await expect.poll(async () => (await assets(page))[0].fontSize).toBeLessThan(initialSize)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].tables[0].height))
    .toBeLessThan(12.5)
  const fitted = (await assets(page))[0]
  expect(fitted.columnWidths[0]).toBeLessThanOrEqual(90)
  expect(fitted.cells).toEqual(table.cells)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(() => assets(page)).toEqual(original)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect.poll(async () => (await assets(page))[0]).toEqual(fitted)
  expect(errors).toEqual([])
})
