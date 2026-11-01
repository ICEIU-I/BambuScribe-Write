import { expect } from '@playwright/test'
import { test, seed, rail, payload, doc, ink } from './pageHarness.mjs'

async function startDrag(page, from) {
  const button = rail(page).nth(from)
  await button.scrollIntoViewIfNeeded()
  const r = await button.boundingBox()
  await page.mouse.move(r.x + r.width / 2, r.y + r.height / 2)
  await page.mouse.down()
  await page.mouse.move(r.x + r.width / 2, r.y + r.height / 2 + 6)
  await expect(page.locator('.page-drag-preview')).toBeVisible()
}
async function drag(page, from, to) {
  const before = (await doc(page)).id
  await startDrag(page, from)
  const r = await rail(page).nth(to).boundingBox()
  await page.mouse.move(r.x + r.width / 2, r.y + (to > from ? r.height - 3 : 3), { steps: 6 })
  await expect(
    page.locator(
      to > from && to === (await rail(page).count()) - 1 ? '.page-drop-after' : '.page-drop-before',
    ),
  ).toHaveCount(1)
  await page.mouse.up()
  await expect.poll(async () => (await doc(page)).id).not.toBe(before)
  await expect(rail(page).nth(to)).toHaveAttribute('aria-current', 'page')
  await expect(page.locator('.page-drag-preview')).toHaveCount(0)
}

test('drag pages in both directions with all objects, exact ink, undo, save and export', async ({
  page,
}) => {
  const errors = await seed(page, undefined, {}, true)
  await expect(rail(page)).toHaveCount(3)
  const before = await payload(page),
    original = await doc(page)
  await drag(page, 1, 0)
  expect((await doc(page)).pages.map(ink)).toEqual([1, 0, 2].map((i) => ink(original.pages[i])))
  const after = await payload(page)
  expect(after.images[0]).toEqual({ ...before.images[0], page: 0 })
  expect(after.tables.map((t) => [t.id, t.page, t.groupId])).toEqual(
    before.tables.map((t) => [t.id, t.page === 1 ? 0 : t.page, t.groupId]),
  )
  expect(after.erasures[0]).toEqual({ ...before.erasures[0], page: 0 })
  await page.keyboard.press('Meta+z')
  await expect.poll(() => payload(page)).toEqual(before)
  await page.keyboard.press('Meta+Shift+z')
  await expect.poll(() => payload(page)).toEqual(after)
  await drag(page, 0, 2)
  await expect(rail(page)).toHaveCount(3)
  expect((await doc(page)).pages.map(ink)).toEqual([0, 2, 1].map((i) => ink(original.pages[i])))
  const saved = await payload(page)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.text)).toBe(saved.text)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect.poll(() => payload(page)).toEqual(saved)
  await rail(page).nth(2).click()
  await page.getByRole('button', { name: '导出', exact: true }).click()
  await page.getByRole('button', { name: /导出当前页/ }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect(errors).toEqual([])
})

test('random rows, wrapped continuations and blank pages keep their own geometry when reordered', async ({
  page,
}) => {
  const errors = await seed(page, '连续正文测试'.repeat(55) + '\n尾段正文', {
    lineMode: 'random',
    indent: 2,
    paragraphGap: 3,
  })
  await expect.poll(() => rail(page).count()).toBeGreaterThan(2)
  const original = await doc(page),
    last = original.pages.length - 1
  await drag(page, last, 0)
  const order = [last, ...original.pages.slice(0, last).map((_, i) => i)]
  expect((await doc(page)).pages.map(ink)).toEqual(order.map((i) => ink(original.pages[i])))
  expect((await doc(page)).pages.map((p) => p.lineHeight)).toEqual(
    order.map((i) => original.pages[i].lineHeight),
  )
  await rail(page).nth(0).click({ button: 'right' })
  await page.getByRole('menuitem', { name: '插入空白页' }).click()
  await expect(rail(page)).toHaveCount(last + 2)
  await drag(page, 1, 0)
  expect((await doc(page)).pages[0].strokes).toEqual([])
  expect((await doc(page)).pages.slice(1).map(ink)).toEqual(
    order.map((i) => ink(original.pages[i])),
  )
  expect(errors).toEqual([])
})

test('ordinary clicks, same-slot drops, outside drops and Escape preserve content', async ({
  page,
}) => {
  const errors = await seed(page)
  await expect(rail(page)).toHaveCount(3)
  const before = await payload(page)
  await rail(page).nth(1).click()
  await expect(rail(page).nth(1)).toHaveAttribute('aria-current', 'page')
  await startDrag(page, 1)
  await page.mouse.up()
  expect(await payload(page)).toEqual(before)
  await startDrag(page, 1)
  await page.mouse.move(300, 450)
  await expect(page.locator('.page-drop-before,.page-drop-after')).toHaveCount(0)
  await page.mouse.up()
  expect(await payload(page)).toEqual(before)
  await startDrag(page, 1)
  const r = await rail(page).first().boundingBox()
  await page.mouse.move(r.x + r.width / 2, r.y + 2)
  await expect(page.locator('.page-drop-before')).toHaveCount(1)
  await page.screenshot({ path: test.info().outputPath('page-reorder.png') })
  await page.keyboard.press('Escape')
  await expect(page.locator('.page-drag-preview')).toHaveCount(0)
  await page.mouse.up()
  expect(await payload(page)).toEqual(before)
  await rail(page).nth(1).click({ button: 'right' })
  await expect(page.getByRole('menuitem', { name: '复制整页' })).toBeVisible()
  expect(errors).toEqual([])
})

test('dragging at the rail edge scrolls to later pages before dropping', async ({ page }) => {
  const errors = await seed(page, Array.from({ length: 20 }, (_, i) => `第${i + 1}页`).join('\f'))
  await expect(rail(page)).toHaveCount(20)
  const original = await doc(page)
  await startDrag(page, 0)
  const root = page.locator('.page-rail'),
    r = await root.boundingBox()
  await page.mouse.move(r.x + r.width / 2, r.y + r.height - 3)
  await expect
    .poll(async () => {
      const last = await rail(page).last().boundingBox()
      return last.y + last.height <= r.y + r.height
    })
    .toBe(true)
  const end = await rail(page).last().boundingBox()
  await page.mouse.move(end.x + end.width / 2, end.y + end.height - 3)
  await expect(page.locator('.page-drop-after')).toHaveCount(1)
  await page.mouse.up()
  await expect(rail(page).last()).toHaveAttribute('aria-current', 'page')
  await expect
    .poll(async () => (await doc(page)).pages.map(ink))
    .toEqual([...original.pages.slice(1), original.pages[0]].map(ink))
  expect(errors).toEqual([])
})

test('pending layout blocks dragging and replacing the project cancels a held page', async ({
  page,
}) => {
  const errors = await seed(page)
  await expect(rail(page)).toHaveCount(3)
  await page.evaluate(() => {
    const preview = window.go.application.App.PreviewWithImages
    window.go.application.App.PreviewWithImages = (...args) =>
      new Promise((resolve) => {
        window.__finishPreview = () => {
          window.go.application.App.PreviewWithImages = preview
          return preview(...args).then(resolve)
        }
      })
    document.querySelector('textarea.paper-input').focus()
  })
  await page.keyboard.insertText('待排版')
  await expect(rail(page).first()).toHaveAttribute('data-reorder-enabled', 'false')
  const r = await rail(page).first().boundingBox()
  await page.mouse.move(r.x + r.width / 2, r.y + r.height / 2)
  await page.mouse.down()
  await page.mouse.move(r.x + r.width / 2, r.y + r.height + 40)
  await page.mouse.up()
  await expect(page.locator('.page-drag-preview')).toHaveCount(0)
  await expect.poll(() => page.evaluate(() => typeof window.__finishPreview)).toBe('function')
  await page.evaluate(() => window.__finishPreview())
  await expect(rail(page).first()).toHaveAttribute('data-reorder-enabled', 'true')
  await startDrag(page, 0)
  await page.evaluate(() => window.__tables.menu['menu:action']('new'))
  await expect(page.locator('.page-drag-preview')).toHaveCount(0)
  await page.mouse.up()
  await expect(rail(page)).toHaveCount(1)
  expect((await payload(page)).text).toBe('')
  expect(errors).toEqual([])
})
