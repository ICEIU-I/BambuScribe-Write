import { test, expect } from '@playwright/test'

async function position(page, x, y) {
  return page.locator('.image-ink').evaluate(
    (svg, p) => {
      const q = new DOMPoint(p.x, p.y).matrixTransform(svg.getScreenCTM())
      return { x: Math.round(q.x), y: Math.round(q.y) }
    },
    { x, y },
  )
}

async function move(page, x, y) {
  const p = await position(page, x, y)
  await page.mouse.move(p.x, p.y)
}

// Sample the browser's actual paint, not a re-serialized SVG: re-rendering the
// SVG on a canvas cannot detect WebKit's transient repaint artifacts.
async function visibleInk(page, points) {
  const locations = await Promise.all(points.map(([x, y]) => position(page, x, y)))
  const screenshot = await page.screenshot({
    scale: 'css',
    path: test.info().outputPath('hover.png'),
  })
  return page.evaluate(
    async ({ png, locations }) => {
      const image = new Image()
      image.src = `data:image/png;base64,${png}`
      await image.decode()
      const canvas = document.createElement('canvas')
      canvas.width = image.width
      canvas.height = image.height
      const ctx = canvas.getContext('2d')
      ctx.drawImage(image, 0, 0)
      return locations.map(({ x, y }) => {
        const pixels = ctx.getImageData(x - 2, y - 2, 5, 5).data
        for (let i = 0; i < pixels.length; i += 4) {
          if (pixels[i] < 230 && pixels[i + 1] < 230 && pixels[i + 2] < 230) return true
        }
        return false
      })
    },
    { png: screenshot.toString('base64'), locations },
  )
}

const regions = (page) => page.evaluate(() => window.eraserFixture.state.images[0].textRegions)

for (const [width, height, zoom, rotation] of [
  [160, 80, 3, 0],
  [80, 160, 2, 0],
  [160, 80, 4.68, 25],
]) {
  test(`hover preserves live ink: ${width}x${height}, scale ${zoom}, rotation ${rotation}`, async ({
    page,
  }) => {
    await page.goto('/tests/browser/eraser.html')
    await page.evaluate(
      (values) => {
        window.eraserFixture.setup(...values)
        window.eraserFixture.state.images[0].textRegions = [
          {
            id: 'existing-erase',
            source: 'manual',
            mode: 'erase',
            text: '',
            x: 0.8,
            y: 0.2,
            width: 0.001,
            height: 0.001,
            erasePath: [{ x: 0.8, y: 0.2 }],
            eraseRadius: 0.04,
          },
          {
            id: 'text',
            source: 'manual',
            mode: 'handwriting',
            text: 'V',
            x: 0.3,
            y: 0.55,
            width: 0.2,
            height: 0.1,
          },
        ]
      },
      [width, height, zoom, rotation],
    )
    const points = [
      [0.3, 0.2],
      [0.6, 0.4],
      [0.4, 0.6],
      [0.6, 0.8],
      [0.8, 0.2],
    ]
    const expected = [true, true, true, true, false]
    expect(await visibleInk(page, points)).toEqual(expected)
    const saved = await regions(page)
    await page.locator('.image-ink').evaluate((ink) => {
      window.inkMutations = []
      new MutationObserver((records) => {
        window.inkMutations.push(...records.map((r) => r.attributeName ?? r.type))
      }).observe(ink, { attributes: true, childList: true, subtree: true })
    })
    for (const [x, y] of [...points.slice(0, 4), ...points.slice(0, 4).toReversed()]) {
      await move(page, x, y)
      await expect(page.locator('.eraser-cursor-dot')).toBeVisible()
      expect(await visibleInk(page, points)).toEqual(expected)
      expect(await regions(page)).toEqual(saved)
    }
    await page.mouse.move(5, 5)
    await expect(page.locator('.eraser-cursor-dot')).toHaveCount(0)
    expect(await visibleInk(page, points)).toEqual(expected)
    // Hover must not invalidate any masked ink, even if a browser repaint
    // happens to hide the transient corruption in a particular screenshot.
    expect(await page.evaluate(() => window.inkMutations)).toEqual([])
  })
}
