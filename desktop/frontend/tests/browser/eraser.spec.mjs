import { test, expect } from '@playwright/test'

async function position(page, x, y) {
  return page.locator('.image-ink').evaluate(
    (svg, { x, y }) => {
      const p = new DOMPoint(x, y).matrixTransform(svg.getScreenCTM())
      return { x: p.x, y: p.y }
    },
    { x, y },
  )
}
async function move(page, x, y) {
  const p = await position(page, x, y)
  // WebKit sends pointer events on integer CSS pixels.
  const actual = { x: Math.round(p.x), y: Math.round(p.y) }
  await page.mouse.move(actual.x, actual.y)
  return actual
}
async function pixels(page, points) {
  return page.locator('.image-ink').evaluate(async (svg, points) => {
    const clone = svg.cloneNode(true)
    clone.querySelectorAll('.eraser-cursor-dot,.erase-preview-path').forEach((el) => el.remove())
    const canvas = document.createElement('canvas')
    canvas.width = 1000
    canvas.height = 500
    clone.setAttribute('width', '1000')
    clone.setAttribute('height', '500')
    const img = new Image()
    img.src = 'data:image/svg+xml;base64,' + btoa(new XMLSerializer().serializeToString(clone))
    await img.decode()
    const ctx = canvas.getContext('2d')
    ctx.drawImage(img, 0, 0)
    return points.map(([x, y]) =>
      Array.from(ctx.getImageData(Math.floor(x * 1000), Math.floor(y * 500), 1, 1).data),
    )
  }, points)
}
for (const [width, height, zoom, rotation] of [
  [160, 80, 3, 0],
  [80, 160, 2, 0],
  [160, 80, 4.68, 25],
]) {
  test(`L stroke preserves untouched interior: ${width}x${height}, scale ${zoom}, rotation ${rotation}`, async ({
    page,
  }) => {
    const errors = []
    page.on('pageerror', (e) => errors.push(e.message))
    await page.goto('/tests/browser/eraser.html')
    await page.evaluate((v) => window.eraserFixture.setup(...v), [width, height, zoom, rotation])
    await page.evaluate(() => scrollTo(0, 60))
    await move(page, 0.2, 0.2)
    await page.mouse.down()
    await move(page, 0.2, 0.8)
    await move(page, 0.8, 0.8)
    let sample = await pixels(page, [
      [0.6, 0.4],
      [0.6, 0.8],
      [0.2, 0.4],
    ])
    expect(sample[0][3]).toBeGreaterThan(0)
    expect(sample[1][3]).toBe(0)
    expect(sample[2][3]).toBe(0)
    await page.mouse.up()
    await expect
      .poll(() => page.evaluate(() => window.eraserFixture.state.images[0].textRegions.length))
      .toBe(1)
    expect(
      await pixels(page, [
        [0.6, 0.4],
        [0.6, 0.8],
      ]),
    ).toEqual([sample[0], sample[1]])
    expect(
      await page.locator('.image-selection').evaluate((el) => getComputedStyle(el).cursor),
    ).toBe('none')
    await expect(page.locator('.image-text-region,.erase-preview-path')).toHaveCount(0)
    await page.getByRole('button', { name: '撤销', exact: true }).click()
    expect((await pixels(page, [[0.6, 0.8]]))[0][3]).toBeGreaterThan(0)
    await page.getByRole('button', { name: '重做', exact: true }).click()
    expect((await pixels(page, [[0.6, 0.8]]))[0][3]).toBe(0)
    // Reload saved correction into a new object as when reopening the draft.
    await page.evaluate(() => {
      const s = window.eraserFixture.state
      s.images = JSON.parse(JSON.stringify(s.images))
    })
    expect((await pixels(page, [[0.6, 0.4]]))[0][3]).toBeGreaterThan(0)
    expect(errors).toEqual([])
  })
}
test('brush circle follows pointer and size, leaves no tooltip, and a click erases only the dot', async ({
  page,
}) => {
  await page.goto('/tests/browser/eraser.html')
  await page.evaluate(() => {
    window.eraserFixture.setup(160, 80, 3, 25)
    window.eraserFixture.radius.value = 0.05
  })
  const p = await move(page, 0.6, 0.4)
  const box = await page.locator('.eraser-cursor-dot').boundingBox()
  expect(Math.abs(box.x + box.width / 2 - p.x)).toBeLessThan(0.1)
  expect(Math.abs(box.y + box.height / 2 - p.y)).toBeLessThan(0.1)
  expect(Math.abs(box.width - box.height)).toBeLessThan(0.1)
  const diameters = await page.locator('.eraser-cursor-dot').evaluate((circle) => {
    const style = getComputedStyle(circle)
    return [parseFloat(style.width), parseFloat(style.height)]
  })
  expect(diameters[0]).toBeCloseTo(24, 4)
  expect(diameters[1]).toBeCloseTo(24, 4)
  await page.mouse.down()
  await page.mouse.up()
  const samples = await pixels(page, [
    [0.6, 0.4],
    [0.6, 0.2],
    [0.6, 0.48],
  ])
  expect(samples[0][3]).toBe(0)
  expect(samples[1][3]).toBeGreaterThan(0)
  expect(samples[2][3]).toBeGreaterThan(0)
  await expect(page.locator('.eraser-cursor-dot span')).toHaveCount(0)
  await page.screenshot({ path: test.info().outputPath('cursor.png') })
  await page.selectOption('[aria-label="工具"]', 'select')
  await expect(page.locator('.eraser-cursor-dot')).toHaveCount(0)
  await page.screenshot({ path: test.info().outputPath('eraser-result.png') })
})
test('rectangle eraser still clears the selected area', async ({ page }) => {
  await page.goto('/tests/browser/eraser.html')
  await page.selectOption('[aria-label="工具"]', 'erase')
  await page.selectOption('[aria-label="擦除方式"]', 'box')
  await move(page, 0.2, 0.2)
  await page.mouse.down()
  await move(page, 0.8, 0.8)
  await page.mouse.up()
  expect((await pixels(page, [[0.6, 0.4]]))[0][3]).toBe(0)
})
