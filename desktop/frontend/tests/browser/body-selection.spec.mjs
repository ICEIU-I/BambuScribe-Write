import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

async function point(page, x, y) {
  return page.locator('[data-paper="0"]').evaluate(
    (paper, p) => {
      const r = paper.getBoundingClientRect(),
        scale = r.width / window.__tables.document.settings.paperWidth
      return { x: r.left + p.x * scale, y: r.top + p.y * scale }
    },
    { x, y },
  )
}

for (const [zoom, fontId, lineSpacing] of [
  ['50', 'dc', 85],
  ['100', 'handwriting', 100],
]) {
  test(`row text selection and image marquee coexist at ${zoom}% with ${fontId}`, async ({
    page,
  }) => {
    const errors = await open(page)
    await page.evaluate(
      ({ fontId, lineSpacing }) => {
        const canvas = document.createElement('canvas')
        canvas.width = 100
        canvas.height = 50
        const ctx = canvas.getContext('2d')
        ctx.fillStyle = 'white'
        ctx.fillRect(0, 0, 100, 50)
        ctx.strokeStyle = 'black'
        ctx.strokeRect(5, 5, 90, 40)
        window.__tables.saved = {
          ...window.__tables.preferences,
          layout: { ...window.__tables.preferences.layout, fontId, lineSpacing },
          text: '第一行正文用于跨行选择。\n这是 半行文字。\n\n\n\n\n图下正文。',
          images: [
            {
              id: 'diagram',
              page: 0,
              x: 30,
              y: 55,
              width: 45,
              height: 20,
              rotation: 0,
              mode: 'line',
              data: canvas.toDataURL(),
              textRegions: [],
            },
          ],
          tables: [],
        }
        window.__tables.menu['menu:action']('open')
      },
      { fontId, lineSpacing },
    )
    await expect
      .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.images?.length))
      .toBe(1)
    await page.getByLabel('页面缩放').selectOption(zoom)
    await page.locator('[data-paper="0"]').evaluate((paper) => {
      const view = paper.closest('.document-viewport')
      view.scrollTop += paper.getBoundingClientRect().top - view.getBoundingClientRect().top - 25
    })
    const before = await page.evaluate(() => window.__tables.payload)
    const { first, last, space, fontSize, lineHeight } = await page.evaluate(() => {
      const doc = window.__tables.document,
        p = doc.pages[0],
        second = p.cells.filter((c) => c.y === p.cells.find((c) => c.y > p.cells[0].y).y)
      return {
        first: p.cells[0],
        last: second.at(-1),
        space: second.find((c) => window.__tables.payload.text.slice(c.start, c.end) === ' '),
        fontSize: doc.settings.fontSize,
        lineHeight: p.lineHeight,
      }
    })
    const start = await point(
      page,
      last.x + last.width + fontSize * 0.7,
      last.y + last.height + (lineHeight - last.height) * 0.25,
    )
    const end = await point(page, first.x, first.y + first.height / 2)
    const image = await point(page, 50, 65)
    const input = page.getByLabel('书写内容')
    await page.mouse.move(start.x, start.y)
    await page.mouse.down()
    await expect(input).toBeFocused()
    expect(await input.evaluate((el) => el.selectionStart)).toBe(last.end)
    // Passing over an object during a text gesture must not start a marquee.
    await page.mouse.move(image.x, image.y, { steps: 5 })
    await page.mouse.move(end.x, end.y, { steps: 8 })
    await page.mouse.up()
    await expect(page.locator('.paper-marquee-layer')).toHaveCount(0)
    expect(await input.evaluate((el) => [el.selectionStart, el.selectionEnd])).toEqual([
      first.start,
      last.end,
    ])
    await expect(page.locator('.text-selection')).not.toHaveCount(0)
    await page.screenshot({ path: test.info().outputPath('row-text-selection.png') })

    // The space between words also belongs to its text row.
    const gap = await point(page, space.x + space.width / 2, space.y + space.height / 2)
    await page.mouse.move(gap.x, gap.y)
    await page.mouse.down()
    await page.mouse.move(end.x, end.y, { steps: 5 })
    await page.mouse.up()
    await expect(page.locator('.paper-marquee-layer')).toHaveCount(0)
    expect(await input.evaluate((el) => el.selectionEnd - el.selectionStart)).toBeGreaterThan(0)

    // Farther along the same half-empty row, dragging still frames the image.
    const blank = await point(page, 120, last.y + last.height / 2)
    const corner = await point(page, 20, 80)
    await page.mouse.move(blank.x, blank.y)
    await page.mouse.down()
    await page.mouse.move(corner.x, corner.y, { steps: 8 })
    await page.mouse.up()
    await expect(page.locator('.marquee-object')).toHaveAttribute('data-marquee-object', 'diagram')
    await page.keyboard.press('Escape')

    // Starting alongside an image, on its own row, is still a marquee.
    const beside = await point(page, 20, 60)
    const across = await point(page, 80, 75)
    await page.mouse.move(beside.x, beside.y)
    await page.mouse.down()
    await page.mouse.move(across.x, across.y, { steps: 8 })
    await page.mouse.up()
    await expect(page.locator('.marquee-object')).toHaveAttribute('data-marquee-object', 'diagram')
    expect(await page.evaluate(() => window.__tables.payload)).toEqual(before)
    expect(errors).toEqual([])
  })
}
