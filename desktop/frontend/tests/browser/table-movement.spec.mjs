import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

const assets = (page) => page.evaluate(() => window.__tables.payload.tables)
const selectors = [
  '[data-free-text-box="title"]',
  '[data-free-text-box="note"]',
  '[data-table="grid"]',
]
async function boxes(page) {
  return Promise.all(selectors.map((s) => page.locator(s).boundingBox()))
}
async function drag(page, handle, dx, dy) {
  const r = await handle.boundingBox(),
    k = await page
      .locator('[data-paper="0"]')
      .evaluate((el) => el.getBoundingClientRect().width / 163)
  await page.mouse.move(r.x + r.width / 2, r.y + r.height / 2)
  await page.mouse.down()
  await page.mouse.move(r.x + r.width / 2 + dx * k, r.y + r.height / 2 + dy * k, { steps: 6 })
}
test('imported caption and table move independently; explicit group drag preserves adjusted spacing', async ({
  page,
}) => {
  const errors = await open(page)
  await page.evaluate(() => {
    const make = (id, y, kind) => ({
      id,
      groupId: 'import-group',
      kind,
      page: 0,
      x: 20,
      y,
      columnWidths: [80],
      rowHeights: [10],
      cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '数据' }],
      style: kind === 'textbox' ? 'none' : 'grid',
      fontSize: 5,
      padding: 1,
      align: 'center',
      verticalAlign: 'middle',
    })
    window.__tables.saved = {
      version: 4,
      text: '',
      images: [],
      tables: [make('title', 20, 'textbox'), make('note', 35, 'textbox'), make('grid', 50)],
      layout: window.__tables.payload.settings,
      writing: window.__tables.preferences.writing,
    }
    window.__tables.menu['menu:action']('open')
  })
  await expect.poll(async () => (await assets(page)).length).toBe(3)
  const note = page.locator(selectors[1]),
    grid = page.locator(selectors[2])
  await note.click()
  const original = await assets(page),
    originalBoxes = await boxes(page)
  await drag(page, note, 12, -2)
  const during = await boxes(page)
  expect(during[0]).toEqual(originalBoxes[0])
  expect(during[2]).toEqual(originalBoxes[2])
  expect(during[1].x).toBeGreaterThan(originalBoxes[1].x + 10)
  expect(await assets(page)).toEqual(original)
  await page.keyboard.press('Escape')
  await page.mouse.up()
  expect(await assets(page)).toEqual(original)
  await drag(page, note, 12, -2)
  await page.mouse.up()
  await expect.poll(async () => (await assets(page))[1].x).toBeGreaterThan(30)
  const adjustedNote = await assets(page)
  expect(adjustedNote[0]).toEqual(original[0])
  expect(adjustedNote[2]).toEqual(original[2])
  expect(adjustedNote[1].groupId).toBe('import-group')
  // Accept an overlapping drop without displacing its table.
  await drag(page, note, 0, 18)
  await page.mouse.up()
  await expect.poll(async () => (await assets(page))[1].y).toBeCloseTo(adjustedNote[1].y + 18, 1)
  const overlapping = await assets(page)
  expect(overlapping[2]).toEqual(adjustedNote[2])
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(() => assets(page)).toEqual(adjustedNote)
  await grid.locator('[data-table-cell]').click()
  const fixed = await boxes(page)
  await drag(page, grid.getByRole('button', { name: '移动表格', exact: true }), 18, 20)
  const solo = await boxes(page)
  expect(solo[0]).toEqual(fixed[0])
  expect(solo[1]).toEqual(fixed[1])
  await page.mouse.up()
  await expect.poll(async () => (await assets(page))[2].y).toBeGreaterThan(65)
  const adjusted = await assets(page)
  expect(adjusted[0]).toEqual(adjustedNote[0])
  expect(adjusted[1]).toEqual(adjustedNote[1])
  await note.click()
  const groupBefore = await boxes(page)
  await drag(page, note.getByRole('button', { name: '整组移动', exact: true }), 6, 10)
  const groupDuring = await boxes(page)
  for (let i = 1; i < 3; i++) {
    expect(groupDuring[i].x - groupDuring[0].x).toBeCloseTo(groupBefore[i].x - groupBefore[0].x, 1)
    expect(groupDuring[i].y - groupDuring[0].y).toBeCloseTo(groupBefore[i].y - groupBefore[0].y, 1)
  }
  await page.screenshot({
    path: `/tmp/bambuscribe-independent-move-${test.info().project.name}.png`,
  })
  await page.mouse.up()
  await expect.poll(async () => (await assets(page))[0].y).toBeGreaterThan(25)
  const moved = await assets(page)
  for (let i = 1; i < 3; i++) {
    expect(moved[i].x - moved[0].x).toBeCloseTo(adjusted[i].x - adjusted[0].x, 6)
    expect(moved[i].y - moved[0].y).toBeCloseTo(adjusted[i].y - adjusted[0].y, 6)
  }
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.tables)).toEqual(moved)
  for (const expected of [adjusted, adjustedNote, original]) {
    await page.getByRole('button', { name: '撤销', exact: true }).click()
    await expect.poll(() => assets(page)).toEqual(expected)
  }
  for (const expected of [adjustedNote, adjusted, moved]) {
    await page.getByRole('button', { name: '重做', exact: true }).click()
    await expect.poll(() => assets(page)).toEqual(expected)
  }
  expect(errors).toEqual([])
})
