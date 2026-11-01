import { expect } from '@playwright/test'
import { test, seed, rail, payload, doc, ink } from './pageHarness.mjs'
async function copy(page, index, count) {
  await rail(page).nth(index).click({ button: 'right' })
  await page.getByRole('menuitem', { name: '复制整页', exact: true }).click()
  await expect(rail(page)).toHaveCount(count)
  await expect(rail(page).nth(index + 1)).toHaveAttribute('aria-current', 'page')
}

async function blank(page, index, count) {
  await rail(page).nth(index).click({ button: 'right' })
  await page.getByRole('menuitem', { name: '插入空白页', exact: true }).click()
  await expect(rail(page)).toHaveCount(count)
  await expect(rail(page).nth(index + 1)).toHaveAttribute('aria-current', 'page')
}

test('insert a blank page below the selected page, retaining later content and independent undo', async ({
  page,
}) => {
  const errors = await seed(page, undefined, {}, true)
  await expect(rail(page)).toHaveCount(3)
  const before = await payload(page),
    original = await doc(page)
  await blank(page, 1, 4)
  await expect(rail(page).nth(2)).toHaveAttribute('aria-label', '第 3 页，0 字')
  const after = await payload(page)
  expect((await doc(page)).pages.map(ink)).toEqual([
    ink(original.pages[0]),
    ink(original.pages[1]),
    [],
    ink(original.pages[2]),
  ])
  expect(after.images.length).toBe(before.images.length)
  expect(after.tables.length).toBe(before.tables.length)
  expect(after.erasures.length).toBe(before.erasures.length)
  expect(after.tables.find((t) => t.id === 'later').page).toBe(3)
  await page.keyboard.press('Meta+z')
  await expect(rail(page)).toHaveCount(3)
  expect(await payload(page)).toEqual(before)
  await page.keyboard.press('Meta+Shift+z')
  await expect(rail(page)).toHaveCount(4)
  expect(await payload(page)).toEqual(after)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved?.layout.pageLayouts?.length))
    .toBe(4)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-paper]')).toHaveCount(4)
  await rail(page).nth(2).click()
  const paper = page.locator('[data-paper="2"]')
  const scale = await paper.evaluate((el) => el.getBoundingClientRect().width / 163)
  await paper.click({ position: { x: 50 * scale, y: 63 * scale } })
  await page.keyboard.insertText('新页第二行')
  await expect.poll(async () => (await doc(page)).pages[2].cells?.[0]?.y).toBe(60)
  const edited = await doc(page)
  expect([edited.pages[0], edited.pages[1], edited.pages[3]].map(ink)).toEqual(
    original.pages.map(ink),
  )
  expect(errors).toEqual([])
})

test('insert blank pages after a partial final sheet or another blank sheet and delete them again', async ({
  page,
}) => {
  const errors = await seed(page, '原页正文')
  await expect(rail(page)).toHaveCount(1)
  const original = ink((await doc(page)).pages[0])
  await blank(page, 0, 2)
  expect((await payload(page)).text).toBe('原页正文\f')
  await rail(page).nth(1).focus()
  await page.keyboard.press('Shift+F10')
  await page.keyboard.press('ArrowDown')
  await page.keyboard.press('Enter')
  await expect(rail(page)).toHaveCount(3)
  expect((await doc(page)).pages.map(ink)).toEqual([original, [], []])
  for (const count of [2, 1]) {
    await rail(page).last().click({ button: 'right' })
    await page.getByRole('menuitem', { name: '删除整页' }).click()
    await expect(rail(page)).toHaveCount(count)
  }
  expect((await payload(page)).text).toBe('原页正文')
  expect(ink((await doc(page)).pages[0])).toEqual(original)
  expect(errors).toEqual([])
})

test('copy page two inserts an independent complete page three and shifts the original third page', async ({
  page,
}) => {
  const errors = await seed(page, undefined, {}, true)
  await expect(rail(page)).toHaveCount(3)
  const before = await payload(page),
    original = await doc(page)
  await copy(page, 1, 4)
  const after = await payload(page),
    result = await doc(page)
  expect(result.pages.map(ink)).toEqual([0, 1, 1, 2].map((i) => ink(original.pages[i])))
  const tables = after.tables.filter((t) => t.page === 2)
  expect(tables).toHaveLength(2)
  expect(new Set(after.tables.map((t) => t.id)).size).toBe(5)
  expect(tables[0].groupId).toBe(tables[1].groupId)
  expect(tables[0].groupId).not.toBe('original-group')
  expect(after.images[1].id).not.toBe(after.images[0].id)
  expect(after.images[1].data).toBe(before.images[0].data)
  const { cuts: sourceCuts, objectIds: sourceIds, ...sourceGesture } = before.erasures[0]
  const { cuts: copiedCuts, objectIds: copiedIds, ...copiedGesture } = after.erasures[1]
  expect(copiedGesture).toEqual({ ...sourceGesture, id: copiedGesture.id, page: 2 })
  expect(copiedCuts.length).toBe(sourceCuts.length)
  expect(copiedCuts.map((c) => [c.source, c.stroke, c.ranges])).toEqual(
    sourceCuts.map((c) => [c.source, c.stroke, c.ranges]),
  )
  expect((copiedIds ?? []).every((id) => !sourceIds.includes(id))).toBe(true)
  await page.keyboard.press('Meta+z')
  await expect(rail(page)).toHaveCount(3)
  expect(await payload(page)).toEqual(before)
  await page.keyboard.press('Meta+Shift+z')
  await expect(rail(page)).toHaveCount(4)
  expect(await payload(page)).toEqual(after)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved.layout.pageLayouts?.length))
    .toBe(4)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-paper]')).toHaveCount(4)
  await page.getByRole('button', { name: '导出', exact: true }).click()
  await page.getByRole('button', { name: /导出当前页/ }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect((await doc(page)).pages.map(ink)).toEqual(result.pages.map(ink))
  // Editing the copied native caption must leave the source caption intact.
  await rail(page).nth(2).click()
  const box = page.locator(`[data-free-text-box="${tables.find((t) => t.kind === 'textbox').id}"]`)
  await box.scrollIntoViewIfNeeded()
  await box.dblclick()
  await page.getByLabel('文本框文字', { exact: true }).fill('副本')
  await page.getByLabel('文本框文字', { exact: true }).press('Enter')
  await expect
    .poll(async () => (await payload(page)).tables.find((t) => t.id === 'caption').cells[0].text)
    .toBe('表格')
  await expect
    .poll(async () => (await payload(page)).tables.find((t) => t.id === tables[1].id).cells[0].text)
    .toBe('副本')
  expect(errors).toEqual([])
})

test('a partial last page duplicates onto a separate sheet, supports repeated copying and deletion', async ({
  page,
}) => {
  const errors = await seed(page, '短页正文')
  await expect(rail(page)).toHaveCount(1)
  const original = ink((await doc(page)).pages[0])
  await copy(page, 0, 2)
  expect((await doc(page)).pages.map(ink)).toEqual([original, original])
  await copy(page, 1, 3)
  expect((await doc(page)).pages.map(ink)).toEqual([original, original, original])
  await rail(page).nth(1).click({ button: 'right' })
  await page.getByRole('menuitem', { name: '删除整页' }).click()
  await expect(rail(page)).toHaveCount(2)
  expect((await doc(page)).pages.map(ink)).toEqual([original, original])
  await rail(page).nth(1).click({ button: 'right' })
  await page.getByRole('menuitem', { name: '删除整页' }).click()
  await expect(rail(page)).toHaveCount(1)
  expect((await doc(page)).pages.map(ink)).toEqual([original])
  expect(errors).toEqual([])
})

test('random grids and wrapped paragraph continuations retain identical ink on every shifted page', async ({
  page,
}) => {
  const errors = await seed(page, '连续正文测试'.repeat(55) + '\n第二段正文', {
    lineMode: 'random',
    indent: 2,
    paragraphGap: 3,
  })
  await expect.poll(() => rail(page).count()).toBeGreaterThan(2)
  const original = await doc(page)
  await copy(page, 1, original.pages.length + 1)
  const expected = [...original.pages]
  expected.splice(2, 0, original.pages[1])
  expect((await doc(page)).pages.map(ink)).toEqual(expected.map(ink))
  expect((await doc(page)).pages.map((p) => p.lineHeight)).toEqual(
    expected.map((p) => p.lineHeight),
  )
  expect(errors).toEqual([])
})

test('blank pages can be copied, saved and edited on any row of the new sheet', async ({
  page,
}) => {
  const errors = await seed(page, '')
  await expect(rail(page)).toHaveCount(1)
  await copy(page, 0, 2)
  expect((await doc(page)).pages.every((p) => !p.strokes.length)).toBe(true)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.text)).toBe('\f')
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-paper]')).toHaveCount(2)
  await rail(page).nth(1).click()
  const target = await page.locator('[data-paper="1"]').evaluate((el) => {
    const r = el.getBoundingClientRect(),
      k = r.width / 163
    return { x: r.left + 50 * k, y: r.top + 63 * k }
  })
  await page.mouse.click(target.x, target.y)
  await page.keyboard.insertText('第二行')
  await expect.poll(async () => (await payload(page)).text).toBe('\f\n第二行')
  await expect.poll(async () => (await doc(page)).pages[1].cells?.[0]?.y).toBe(60)
  expect((await doc(page)).pages[0].strokes).toEqual([])
  expect(errors).toEqual([])
})

test('object-only copying includes all native objects and deleting a full final copy leaves no ghost sheet', async ({
  page,
}) => {
  const errors = await seed(page, '', {}, true)
  await expect(rail(page)).toHaveCount(3)
  const original = await doc(page)
  await copy(page, 1, 4)
  expect((await doc(page)).pages.map(ink)).toEqual([0, 1, 1, 2].map((i) => ink(original.pages[i])))
  await page.evaluate(() => {
    window.__tables.saved = {
      ...window.__tables.preferences,
      text: '一\n二\n三\n四\n最后',
      images: [],
      tables: [],
      erasures: [],
      layout: { ...window.__tables.preferences.layout, lineCount: 4, pageLayouts: undefined },
    }
    window.__tables.menu['menu:action']('open')
  })
  await expect(rail(page)).toHaveCount(2)
  await copy(page, 0, 3)
  for (const count of [2, 1]) {
    await rail(page).last().click({ button: 'right' })
    await page.getByRole('menuitem', { name: '删除整页' }).click()
    await expect(rail(page)).toHaveCount(count)
  }
  expect((await payload(page)).text).toBe('一\n二\n三\n四')
  expect(errors).toEqual([])
})
