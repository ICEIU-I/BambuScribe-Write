import { test, expect } from '@playwright/test'

const images = (page) => page.evaluate(() => window.imageFixture.state.images)
const image = (page) => page.locator('[data-image-id="diagram"]')
async function point(page, index, x, y) {
  return page.locator(`[data-paper="${index}"]`).evaluate(
    (el, p) => {
      const r = el.getBoundingClientRect(),
        k = r.width / 163
      return { x: r.left + p.x * k, y: r.top + p.y * k }
    },
    { x, y },
  )
}
async function showBoundary(page) {
  await page.locator('.document-viewport').evaluate((view) => {
    const paper = view.querySelector('[data-paper="2"]')
    view.scrollTop +=
      paper.getBoundingClientRect().top -
      view.getBoundingClientRect().top -
      view.clientHeight * 0.65
  })
}
async function open(page, zoom = 65) {
  const errors = []
  page.on('pageerror', (error) => errors.push(error.message))
  await page.goto('/tests/browser/image.html')
  await page.evaluate((zoom) => {
    window.imageFixture.state.zoom = zoom
  }, zoom)
  await expect(image(page)).toHaveCount(1)
  await showBoundary(page)
  await expect(image(page)).toBeVisible()
  return errors
}
const move = (page, p) => page.mouse.move(p.x, p.y)

for (const zoom of [65, 100]) {
  test(`image drags from page 3 to page 2 at ${zoom}% with live ink and one undo step`, async ({
    page,
  }) => {
    const errors = await open(page, zoom),
      original = (await images(page))[0]
    await page.evaluate(() => window.imageFixture.freeze())
    await move(page, await point(page, 2, 45, 15))
    await page.mouse.down()
    const destination = await point(page, 1, 60, 215)
    await move(page, destination)
    const live = await image(page).boundingBox(),
      expected = await point(page, 1, 50, 200)
    expect(Math.abs(live.x - expected.x)).toBeLessThan(1)
    expect(Math.abs(live.y - expected.y)).toBeLessThan(1)
    expect((await images(page))[0]).toEqual(original)
    await expect(image(page).locator('.image-ink path').first()).toHaveAttribute('d', /M.+L/)
    await page.screenshot({ path: test.info().outputPath('cross-page-drag.png') })
    await page.mouse.up()
    await expect(page.locator('[data-paper="1"] [data-image-id="diagram"]')).toHaveCount(1)
    const moved = (await images(page))[0]
    expect(moved).toEqual({ ...original, page: 1, x: moved.x, y: moved.y })
    expect(Math.abs(moved.x - 50)).toBeLessThan(0.3)
    expect(moved.y).toBe(200)
    await expect(page.locator('[data-current-page]')).toHaveText('2')
    // The new page keeps the old vector preview while a new engine result is pending.
    await expect(image(page).locator('.image-ink path').first()).toHaveAttribute('d', /M.+L/)
    await page.keyboard.press('Meta+z')
    expect((await images(page))[0]).toEqual(original)
    await expect(page.locator('[data-current-page]')).toHaveText('3')
    await page.keyboard.press('Meta+Shift+z')
    expect((await images(page))[0]).toEqual(moved)
    expect(errors).toEqual([])
  })
}

test('drag scrolls at the viewport edge and Escape cancels without changing the image', async ({
  page,
}) => {
  await open(page, 100)
  await page.setViewportSize({ width: 1000, height: 650 })
  await showBoundary(page)
  const before = (await images(page))[0]
  const start = await point(page, 2, 45, 15),
    view = await page.locator('.document-viewport').boundingBox()
  await move(page, start)
  await page.mouse.down()
  const scroll = await page.locator('.document-viewport').evaluate((v) => v.scrollTop)
  await move(page, { x: start.x, y: view.y + 4 })
  await expect
    .poll(() => page.locator('.document-viewport').evaluate((v) => v.scrollTop))
    .toBeLessThan(scroll - 80)
  await page.mouse.up()
  expect((await images(page))[0].page).toBe(1)
  await page.keyboard.press('Meta+z')
  expect((await images(page))[0]).toEqual(before)
  await showBoundary(page)
  await move(page, await point(page, 2, 45, 15))
  await page.mouse.down()
  await move(page, await point(page, 1, 60, 215))
  await page.keyboard.press('Escape')
  await page.mouse.up()
  expect((await images(page))[0]).toEqual(before)
})

test('images also drag downward across pages; resize stays on its page and respects paper bounds', async ({
  page,
}) => {
  await open(page)
  await page.evaluate(() => {
    const state = window.imageFixture.state
    state.images[0] = { ...state.images[0], page: 1, y: 200 }
  })
  await showBoundary(page)
  const original = (await images(page))[0]
  await move(page, await point(page, 1, 45, 205))
  await page.mouse.down()
  await move(page, await point(page, 2, 150, 20))
  await page.mouse.up()
  const moved = (await images(page))[0]
  expect(moved.page).toBe(2)
  expect(moved.x).toBe(93)
  expect(Math.abs(moved.y - 15)).toBeLessThan(0.3)
  expect(moved.width).toBe(original.width)
  const handle = await page.getByRole('button', { name: '调整图片大小' }).boundingBox()
  await move(page, { x: handle.x + handle.width / 2, y: handle.y + handle.height / 2 })
  await page.mouse.down()
  await move(page, { x: handle.x + 80, y: handle.y + 70 })
  await page.mouse.up()
  const resized = (await images(page))[0]
  expect(resized.page).toBe(2)
  expect(resized.x + resized.width).toBeLessThanOrEqual(163)
  expect(resized.height).toBeGreaterThan(moved.height)
  await page.keyboard.press('Meta+z')
  expect((await images(page))[0]).toEqual(moved)
})

test('copy/paste preserves image edits, targets the clicked page, and creates independent copies', async ({
  page,
}) => {
  const errors = await open(page),
    original = (await images(page))[0]
  await image(page).click({ position: { x: 20, y: 10 } })
  await page.keyboard.press('Meta+c')
  await page.keyboard.press('Meta+v')
  await expect.poll(async () => (await images(page)).length).toBe(2)
  const firstCopy = (await images(page))[1]
  expect(firstCopy).toEqual({ ...original, id: firstCopy.id, x: 37, y: 12 })
  expect(firstCopy.id).not.toBe(original.id)
  const target = await point(page, 1, 25, 180)
  await page.mouse.click(target.x, target.y)
  await page.keyboard.press('Meta+v')
  await expect.poll(async () => (await images(page)).length).toBe(3)
  const second = (await images(page))[2]
  expect(second).toEqual({ ...original, id: second.id, page: 1, x: second.x, y: second.y })
  expect(Math.abs(second.x - 25)).toBeLessThan(0.3)
  expect(Math.abs(second.y - 180)).toBeLessThan(0.3)
  await page.keyboard.press('Meta+v')
  const third = (await images(page))[3]
  expect(third.id).not.toBe(second.id)
  expect(third.x).toBe(second.x + 2)
  expect(third.y).toBe(second.y + 2)
  await page.keyboard.press('Meta+z')
  expect((await images(page)).length).toBe(3)
  await page.keyboard.press('Meta+Shift+z')
  expect((await images(page)).length).toBe(4)
  await page.evaluate(() => {
    window.imageFixture.state.images[3].textRegions[1].text = '副本'
  })
  expect((await images(page))[0]).toEqual(original)
  expect((await images(page))[2].textRegions[1].text).toBe('光源')
  expect(await page.getByRole('textbox', { name: '书写内容' }).inputValue()).toBe('正文保持不变')
  expect(errors).toEqual([])
})

test('cut then paste moves an image to page 2 and undo restores each step', async ({ page }) => {
  const errors = await open(page),
    original = (await images(page))[0]
  await image(page).click({ position: { x: 20, y: 10 } })
  await page.keyboard.press('Meta+x')
  await expect(image(page)).toHaveCount(0)
  expect(await images(page)).toEqual([])
  const target = await point(page, 1, 30, 175)
  await page.mouse.click(target.x, target.y)
  await page.keyboard.press('Meta+v')
  await expect.poll(async () => (await images(page)).length).toBe(1)
  const pasted = (await images(page))[0]
  expect(pasted).toEqual({ ...original, id: pasted.id, page: 1, x: pasted.x, y: pasted.y })
  expect(Math.abs(pasted.x - 30)).toBeLessThan(0.3)
  expect(Math.abs(pasted.y - 175)).toBeLessThan(0.3)
  await page.keyboard.press('Meta+z')
  expect(await images(page)).toEqual([])
  await page.keyboard.press('Meta+z')
  expect(await images(page)).toEqual([original])
  await page.keyboard.press('Meta+Shift+z')
  expect(await images(page)).toEqual([])
  await page.keyboard.press('Meta+Shift+z')
  expect(await images(page)).toEqual([pasted])
  expect(errors).toEqual([])
})

test('paste uses the chosen page after a page jump and clipboard stays native inside other inputs', async ({
  page,
}) => {
  await open(page)
  await image(page).click({ position: { x: 20, y: 10 } })
  await page.keyboard.press('Meta+c')
  await page.getByRole('button', { name: '第 2 页', exact: true }).click()
  await page.keyboard.press('Meta+v')
  await expect.poll(async () => (await images(page)).length).toBe(2)
  expect((await images(page))[1].page).toBe(1)
  const input = page.getByRole('textbox', { name: '其他输入' })
  await input.fill('其他内容')
  await input.press('Meta+a')
  await page.keyboard.press('Meta+x')
  expect(await input.inputValue()).toBe('')
  expect((await images(page)).length).toBe(2)
  await page.mouse.click((await point(page, 1, 10, 100)).x, (await point(page, 1, 10, 100)).y)
  await page.keyboard.press('Meta+v')
  expect((await images(page)).length).toBe(2)
  await expect(page.getByRole('textbox', { name: '书写内容' })).toHaveValue(/其他内容/)
})

test('disabled and inactive editors ignore image cut/paste; pointer cancellation restores the original', async ({
  page,
}) => {
  await open(page)
  await image(page).click({ position: { x: 20, y: 10 } })
  await page.keyboard.press('Meta+c')
  const original = await images(page)
  for (const flag of ['disabled', 'active']) {
    await page.evaluate((flag) => {
      window.imageFixture.state[flag] = flag === 'disabled'
    }, flag)
    await page.keyboard.press('Meta+v')
    await page.keyboard.press('Meta+x')
    expect(await images(page)).toEqual(original)
    await page.evaluate((flag) => {
      window.imageFixture.state[flag] = flag !== 'disabled'
    }, flag)
  }
  await move(page, await point(page, 2, 45, 15))
  await page.mouse.down()
  await move(page, await point(page, 1, 60, 215))
  await image(page).dispatchEvent('pointercancel')
  await page.mouse.up()
  expect(await images(page)).toEqual(original)
})
