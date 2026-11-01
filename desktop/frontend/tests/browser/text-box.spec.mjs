import { test, expect } from '@playwright/test'
async function position(page, x, y) {
  return page.locator('.image-ink').evaluate(
    (svg, p) => {
      const q = new DOMPoint(p.x, p.y).matrixTransform(svg.getScreenCTM())
      return { x: q.x, y: q.y }
    },
    { x, y },
  )
}
async function move(page, x, y) {
  const p = await position(page, x, y)
  await page.mouse.move(p.x, p.y)
}
async function sample(page, x, y) {
  return page.locator('.image-ink').evaluate(
    async (svg, p) => {
      const clone = svg.cloneNode(true)
      clone.setAttribute('width', '1000')
      clone.setAttribute('height', '500')
      const img = new Image()
      img.src = 'data:image/svg+xml;base64,' + btoa(new XMLSerializer().serializeToString(clone))
      await img.decode()
      const c = document.createElement('canvas')
      c.width = 1000
      c.height = 500
      const ctx = c.getContext('2d')
      ctx.drawImage(img, 0, 0)
      return ctx.getImageData(Math.floor(p.x * 1000), Math.floor(p.y * 500), 1, 1).data[3]
    },
    { x, y },
  )
}
async function seed(page) {
  await page.goto('/tests/browser/eraser.html')
  await page.selectOption('[aria-label="工具"]', 'select')
  await page.evaluate(() => {
    window.eraserFixture.state.images[0].textRegions = [
      {
        id: 'unit',
        source: 'manual',
        mode: 'handwriting',
        text: 'W/m²',
        x: 0.3,
        y: 0.35,
        width: 0.2,
        height: 0.1,
      },
    ]
  })
  await expect(page.locator('[data-text-box="unit"]')).toBeVisible()
}
const boxes = (page) => page.evaluate(() => window.eraserFixture.state.images[0].textRegions)

test('manual text can be selected, copied and pasted at the clicked position', async ({ page }) => {
  await seed(page)
  await page.locator('[data-text-box="unit"]').click()
  await expect(page.locator('.manual-text-box.is-selected')).toHaveCount(1)
  await page.keyboard.press('Meta+c')
  await move(page, 0.3, 0.55)
  await page.mouse.click((await position(page, 0.3, 0.55)).x, (await position(page, 0.3, 0.55)).y)
  await page.keyboard.press('Meta+v')
  await expect.poll(async () => (await boxes(page)).length).toBe(2)
  const [original, copy] = await boxes(page)
  expect(copy.id).not.toBe(original.id)
  expect(copy.text).toBe('W/m²')
  expect(copy.width).toBe(original.width)
  expect(copy.height).toBe(original.height)
  expect(copy.x).toBeCloseTo(0.3, 2)
  expect(copy.y).toBeCloseTo(0.55, 2)
  expect(await sample(page, 0.48, 0.4)).toBe(0)
  expect(await sample(page, 0.48, 0.6)).toBe(0)
  await page.keyboard.press('Meta+z')
  await expect.poll(async () => (await boxes(page)).length).toBe(1)
  expect(await sample(page, 0.48, 0.6)).toBeGreaterThan(0)
  await page.keyboard.press('Meta+Shift+z')
  await expect.poll(async () => (await boxes(page)).length).toBe(2)
  // Inline text still uses native clipboard and delete semantics.
  await page.locator(`[data-text-box="${copy.id}"]`).dblclick()
  const input = page.getByRole('textbox', { name: '输入图片文字' })
  await expect(input).toBeFocused()
  await input.fill('400 W/m²')
  await input.press('Backspace')
  expect((await boxes(page)).length).toBe(2)
  await input.fill('400 W/m²')
  await input.press('Enter')
  await expect.poll(async () => (await boxes(page))[1].text).toBe('400 W/m²')
})

test('move and resize restore the old background live, deleting affects only the textbox', async ({
  page,
}) => {
  const errors = []
  page.on('pageerror', (e) => errors.push(e.message))
  await seed(page)
  await page.evaluate(() => window.eraserFixture.freeze())
  expect(await sample(page, 0.48, 0.4)).toBe(0)
  await move(page, 0.35, 0.38)
  await page.mouse.down()
  await move(page, 0.65, 0.68)
  expect(await sample(page, 0.48, 0.4)).toBeGreaterThan(0)
  expect(await sample(page, 0.78, 0.7)).toBe(0)
  await page.mouse.up()
  let value = (await boxes(page))[0]
  expect(value.x).toBeCloseTo(0.6, 2)
  expect(value.y).toBeCloseTo(0.65, 2)
  const handle = await page.getByRole('button', { name: '调整文本框大小' }).boundingBox()
  await page.mouse.move(handle.x + handle.width / 2, handle.y + handle.height / 2)
  await page.mouse.down()
  await move(page, 0.9, 0.85)
  await page.mouse.up()
  value = (await boxes(page))[0]
  expect(value.width).toBeGreaterThan(0.28)
  expect(value.height).toBeGreaterThan(0.18)
  await page.keyboard.press('Meta+z')
  value = (await boxes(page))[0]
  expect(value.width).toBe(0.2)
  await page.keyboard.press('Meta+z')
  value = (await boxes(page))[0]
  expect(value.x).toBe(0.3)
  await page.locator('[data-text-box="unit"]').click()
  await page.keyboard.press('Delete')
  await expect.poll(async () => (await boxes(page)).length).toBe(0)
  expect(await page.evaluate(() => window.eraserFixture.state.images.length)).toBe(1)
  expect(await sample(page, 0.48, 0.4)).toBeGreaterThan(0)
  await page.keyboard.press('Meta+z')
  await expect.poll(async () => (await boxes(page)).length).toBe(1)
  await page.getByRole('button', { name: '删除文本框', exact: true }).click()
  await expect.poll(async () => (await boxes(page)).length).toBe(0)
  expect(errors).toEqual([])
})

test('new text starts typing immediately and an erased textbox never brings back OCR controls', async ({
  page,
}) => {
  await page.goto('/tests/browser/eraser.html')
  await page.selectOption('[aria-label="工具"]', 'text')
  await move(page, 0.3, 0.35)
  await page.mouse.down()
  await move(page, 0.5, 0.45)
  await page.mouse.up()
  const input = page.getByRole('textbox', { name: '输入图片文字' })
  await expect(input).toBeFocused()
  await input.fill('W/m²')
  await input.press('Enter')
  await expect.poll(async () => (await boxes(page))[0].text).toBe('W/m²')
  await page.selectOption('[aria-label="工具"]', 'erase')
  await page.selectOption('[aria-label="擦除方式"]', 'free')
  await expect(page.locator('.manual-text-box.is-selected')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '删除文本框' })).toHaveCount(0)
  await expect(page.locator('.image-text-region')).toHaveCount(0)
})
