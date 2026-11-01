import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

async function seed(page, { image = false, secondPage = false } = {}) {
  await page.evaluate(
    ({ image, secondPage }) => {
      const p = JSON.parse(JSON.stringify(window.__tables.preferences))
      p.version = 3
      p.text = '正文测试'
      p.erasures = []
      const table = {
        id: 'grid',
        page: 0,
        x: 15,
        y: 70,
        columnWidths: [35, 35],
        rowHeights: [12],
        cells: [
          { row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '实验' },
          { row: 0, col: 1, rowSpan: 1, colSpan: 1, text: '数据' },
        ],
        style: 'grid',
        fontSize: 4,
        padding: 1,
        align: 'center',
        verticalAlign: 'middle',
      }
      p.tables = [
        table,
        {
          ...table,
          id: 'label',
          kind: 'textbox',
          y: 105,
          columnWidths: [90],
          rowHeights: [12],
          cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '独立文本框' }],
          style: 'none',
          align: 'left',
        },
      ]
      if (secondPage) p.tables.push({ ...table, id: 'page-two', page: 1 })
      if (image) {
        const canvas = document.createElement('canvas')
        canvas.width = 240
        canvas.height = 80
        const ctx = canvas.getContext('2d')
        ctx.fillStyle = 'white'
        ctx.fillRect(0, 0, 240, 80)
        ctx.strokeStyle = 'black'
        ctx.lineWidth = 3
        ctx.beginPath()
        ctx.moveTo(10, 40)
        ctx.lineTo(230, 40)
        ctx.moveTo(120, 10)
        ctx.lineTo(120, 70)
        ctx.stroke()
        p.images = [
          {
            id: 'diagram',
            page: 0,
            x: 25,
            y: 30,
            width: 70,
            height: 20,
            rotation: 25,
            mode: 'line',
            data: canvas.toDataURL(),
          },
        ]
      }
      window.__tables.saved = p
      window.__tables.menu['menu:action']('open')
    },
    { image, secondPage },
  )
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.tables?.length))
    .toBe(2)
}
async function point(page, x, y, index = 0) {
  return page.locator(`[data-paper="${index}"]`).evaluate(
    (el, { x, y }) => {
      const r = el.getBoundingClientRect(),
        k = r.width / window.__tables.document.settings.paperWidth
      return { x: r.left + x * k, y: r.top + y * k }
    },
    { x, y },
  )
}
async function drag(page, a, b, index = 0, release = true) {
  const start = await point(page, ...a, index),
    end = await point(page, ...b, index)
  await page.mouse.move(start.x, start.y)
  await page.mouse.down()
  await page.mouse.move(end.x, end.y, { steps: 12 })
  if (release) await page.mouse.up()
}
async function count(page) {
  return page.evaluate(() => window.__tables.payload?.erasures?.length ?? 0)
}

for (const mode of ['rectangle', 'free', 'legacy']) {
  test(`old ${mode} erasure leaves newly added and moved captions intact`, async ({ page }) => {
    const errors = await open(page)
    await seed(page)
    await page.getByRole('button', { name: '擦除', exact: true }).click()
    if (mode === 'free') {
      await page.getByRole('button', { name: '自由', exact: true }).click()
      await drag(page, [50, 65], [50, 95])
    } else await drag(page, [10, 65], [100, 95])
    await expect.poll(() => count(page)).toBe(1)
    expect(await page.evaluate(() => window.__tables.payload.erasures[0].objectIds)).toEqual([
      'grid',
    ])
    if (mode === 'legacy') {
      await page.evaluate(() => {
        window.__tables.saved = {
          ...window.__tables.preferences,
          text: window.__tables.payload.text,
          layout: window.__tables.payload.settings,
          images: window.__tables.payload.images,
          tables: window.__tables.payload.tables,
          erasures: window.__tables.payload.erasures.map(({ objectIds, cuts, ...e }) => e),
        }
        window.__tables.menu['menu:action']('open')
      })
      await expect
        .poll(() => page.evaluate(() => window.__tables.payload.erasures[0].cuts))
        .toBeDefined()
    }
    await page.getByRole('button', { name: '选择', exact: true }).click()
    const caption = page.locator('[data-free-text-box="label"]')
    await caption.click()
    const handle = await page.getByRole('button', { name: '移动文本框' }).boundingBox()
    const k = await page
      .locator('[data-paper="0"]')
      .evaluate((el) => el.getBoundingClientRect().width / 163)
    await page.mouse.move(handle.x + 8, handle.y + 8)
    await page.mouse.down()
    await page.mouse.move(handle.x + 8, handle.y + 8 - 30 * k, { steps: 6 })
    await page.mouse.up()
    await expect
      .poll(() =>
        page.evaluate(() => window.__tables.payload.tables.find((t) => t.id === 'label').y),
      )
      .toBeLessThan(80)
    await expect(caption.locator('.free-text-ink')).not.toHaveAttribute('style', /mask-image/)
    expect(await page.evaluate(() => window.__tables.payload.erasures[0].objectIds)).toEqual([
      'grid',
    ])

    await page.getByRole('button', { name: '文本框', exact: true }).click()
    const p = await point(page, 20, 90)
    await page.mouse.click(p.x, p.y)
    await page.getByLabel('文本框文字').fill('新图注')
    await page.getByLabel('文本框文字').press('Enter')
    await expect(page.locator('[data-free-text-box]')).toHaveCount(2)
    await expect(
      page.locator('[data-free-text-box]').last().locator('.free-text-ink'),
    ).not.toHaveAttribute('style', /mask-image/)
    await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
    await expect
      .poll(() => page.evaluate(() => window.__tables.saved.erasures[0].objectIds))
      .toEqual(['grid'])
    await page.evaluate(() => window.__tables.menu['menu:action']('open'))
    await expect(page.locator('[data-free-text-box]')).toHaveCount(2)
    await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
    await expect
      .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
      .toContain('A1 handwriting')
    const missing = await page.evaluate(() => {
      const p = window.__tables.exported.document.pages[0]
      const keys = new Set(p.strokes.map((s) => JSON.stringify(s)))
      return p.tables
        .filter((t) => t.kind === 'textbox')
        .flatMap((t) => t.strokes.map((s) => s.map((p) => ({ x: p.x + t.x, y: p.y + t.y }))))
        .filter((s) => !keys.has(JSON.stringify(s))).length
    })
    expect(missing).toBe(0)
    await page.screenshot({ path: test.info().outputPath(`caption-after-${mode}-erase.png`) })
    expect(errors).toEqual([])
  })
}
async function painted(page, area = { x: 1, y: 1, width: 140, height: 118 }) {
  const a = await point(page, area.x, area.y),
    b = await point(page, area.x + area.width, area.y + area.height)
  const png = await page.screenshot({ scale: 'css' })
  return page.evaluate(
    async ({ png, a, b }) => {
      const image = new Image()
      image.src = `data:image/png;base64,${png}`
      await image.decode()
      const canvas = document.createElement('canvas')
      canvas.width = image.width
      canvas.height = image.height
      const ctx = canvas.getContext('2d')
      ctx.drawImage(image, 0, 0)
      const pixels = ctx.getImageData(
        Math.ceil(a.x),
        Math.ceil(a.y),
        Math.floor(b.x - a.x),
        Math.floor(b.y - a.y),
      ).data
      let ink = 0,
        rules = 0
      for (let i = 0; i < pixels.length; i += 4) {
        if (pixels[i] < 80 && pixels[i + 1] < 80 && pixels[i + 2] < 80) ink++
        if (pixels[i] >= 150 && pixels[i] < 200 && pixels[i + 1] >= 150 && pixels[i + 1] < 200)
          rules++
      }
      return { ink, rules }
    },
    { png: png.toString('base64'), a, b },
  )
}

test('paper rectangle erases body, rotated image, grid and textbox immediately and preserves rules, history and saved project', async ({
  page,
}) => {
  const errors = await open(page)
  await seed(page, { image: true, secondPage: true })
  await page.getByRole('button', { name: '擦除', exact: true }).click()
  const before = await painted(page)
  expect(before.ink).toBeGreaterThan(100)
  const originals = await page.evaluate(() => ({
    text: window.__tables.payload.text,
    images: window.__tables.payload.images,
    tables: window.__tables.payload.tables,
  }))
  await drag(page, [0, 0], [150, 120], 0, false)
  await page.mouse.move((await point(page, 155, 125)).x, (await point(page, 155, 125)).y)
  const during = await painted(page)
  expect(during.ink).toBe(0)
  expect(during.rules).toBeGreaterThan(100)
  await page.mouse.up()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].strokes.length))
    .toBe(0)
  expect(await count(page)).toBe(1)
  const erasures = await page.evaluate(() => window.__tables.payload.erasures)
  expect(
    await page.evaluate(() => ({
      text: window.__tables.payload.text,
      images: window.__tables.payload.images,
      tables: window.__tables.payload.tables,
    })),
  ).toEqual(originals)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(() => count(page)).toBe(0)
  expect((await painted(page)).ink).toBeGreaterThan(100)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect.poll(() => count(page)).toBe(1)
  expect((await painted(page)).ink).toBe(0)
  await page.keyboard.press('Control+Shift+S')
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.erasures?.length)).toBe(1)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(() => count(page)).toBe(0)
  // Named projects autosave undo and redo; reopening restores the latest edit.
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.erasures?.length)).toBe(0)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.erasures?.length)).toBe(1)
  await page.keyboard.press('Control+O')
  await expect.poll(() => count(page)).toBe(1)
  expect(await page.evaluate(() => window.__tables.payload.erasures)).toEqual(erasures)
  expect((await painted(page)).ink).toBe(0)
  await expect(page.locator('.paper-caret,.text-selection')).toHaveCount(0)
  expect(
    await page.evaluate(() => window.__tables.document.pages[1].strokes.length),
  ).toBeGreaterThan(0)
  await page.screenshot({ path: test.info().outputPath('paper-eraser.png') })
  expect(errors).toEqual([])
})

test('free paper brush crosses blank paper and table without bounding-box erasure, matches G-code and works after zoom', async ({
  page,
}) => {
  const errors = await open(page)
  await seed(page)
  const before = await page.evaluate(() => window.__tables.document.pages[0].strokes)
  await page.getByRole('button', { name: '擦除', exact: true }).click()
  await page.getByRole('button', { name: '自由', exact: true }).click()
  const a = await point(page, 30, 55),
    b = await point(page, 30, 85),
    c = await point(page, 100, 85)
  await page.mouse.move(a.x, a.y)
  await page.mouse.down()
  await page.mouse.move(b.x, b.y, { steps: 8 })
  await page.mouse.move(c.x, c.y, { steps: 10 })
  await page.mouse.up()
  await expect.poll(() => count(page)).toBe(1)
  const e = await page.evaluate(() => window.__tables.payload.erasures[0])
  expect(e.radius).toBeCloseTo(1.8, 8)
  expect(e.path.length).toBeGreaterThan(10)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].strokes))
    .not.toEqual(before)
  const after = await page.evaluate(() => window.__tables.document.pages[0].strokes)
  // Vertical rule at x=50 lies inside the gesture's bounding box, outside its brush.
  expect(after.some((s) => s.some((p) => Math.abs(p.x - 50) < 0.001 && p.y === 70))).toBe(true)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  expect(await page.evaluate(() => window.__tables.exported.document.pages[0].strokes)).toEqual(
    after,
  )
  await page.getByRole('button', { name: '选择', exact: true }).click()
  await page.evaluate(() => window.__tables.menu['menu:action']('zoomOut'))
  await page.getByRole('button', { name: '擦除', exact: true }).click()
  const hover = await point(page, 50, 70)
  await page.mouse.move(hover.x, hover.y)
  const cursor = await page.locator('.paper-eraser .eraser-cursor-dot').first().boundingBox()
  expect(Math.abs(cursor.width / ((96 / 25.4) * 0.9) - 3.6)).toBeLessThan(0.05)
  await page.mouse.click(hover.x, hover.y)
  await expect.poll(() => count(page)).toBe(2)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(() => count(page)).toBe(1)
  expect(errors).toEqual([])
})

test('cancelled gesture is discarded and erasing another page keeps the first page untouched', async ({
  page,
}) => {
  const errors = await open(page)
  await seed(page, { secondPage: true })
  const before = await page.evaluate(() => window.__tables.document.pages[0].strokes)
  await page.getByRole('button', { name: '擦除', exact: true }).click()
  await drag(page, [10, 60], [100, 90], 0, false)
  await page.keyboard.press('Escape')
  await page.mouse.up()
  expect(await count(page)).toBe(0)
  await page.locator('[data-paper="1"]').scrollIntoViewIfNeeded()
  await page.getByRole('button', { name: '擦除', exact: true }).click()
  await drag(page, [0, 60], [100, 95], 1)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[1].strokes.length))
    .toBe(0)
  expect(await page.evaluate(() => window.__tables.document.pages[0].strokes)).toEqual(before)
  expect((await page.evaluate(() => window.__tables.payload.erasures[0])).page).toBe(1)
  expect(errors).toEqual([])
})

test('a circular paper erasure clips a rotated image at the pointer and hover leaves ink unchanged', async ({
  page,
}) => {
  const errors = await open(page)
  await seed(page, { image: true })
  // Keep the 2 mm pixel sample large enough for WebKit antialiasing after sidebar reflow.
  await page.getByLabel('页面缩放', { exact: true }).selectOption('150')
  await page.getByRole('button', { name: '擦除', exact: true }).click()
  await page.getByRole('button', { name: '自由', exact: true }).click()
  const center = { x: 59, y: 39, width: 2, height: 2 }
  const original = await painted(page, center)
  expect(original.ink).toBeGreaterThan(5)
  const markup = await page.locator('.image-ink').evaluate((el) => el.outerHTML)
  const p = await point(page, 60, 40)
  await page.mouse.move(p.x, p.y)
  expect(await count(page)).toBe(0)
  expect(await page.locator('.image-ink').evaluate((el) => el.outerHTML)).toBe(markup)
  expect((await painted(page, center)).ink).toBeGreaterThan(5)
  await page.mouse.click(p.x, p.y)
  await expect.poll(() => count(page)).toBe(1)
  await page.mouse.move((await point(page, 110, 50)).x, (await point(page, 110, 50)).y)
  expect((await painted(page, center)).ink).toBe(0)
  await expect.poll(() => page.evaluate(() => window.__tables.payload.erasures.length)).toBe(1)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  const distance = await page.evaluate(() => {
    const e = window.__tables.payload.erasures[0],
      center = e.path[0]
    let nearest = Infinity
    for (const stroke of window.__tables.exported.document.pages[0].strokes)
      for (let i = 1; i < stroke.length; i++) {
        const a = stroke[i - 1],
          b = stroke[i],
          dx = b.x - a.x,
          dy = b.y - a.y
        const t = Math.max(
          0,
          Math.min(1, ((center.x - a.x) * dx + (center.y - a.y) * dy) / (dx * dx + dy * dy || 1)),
        )
        nearest = Math.min(nearest, Math.hypot(a.x + t * dx - center.x, a.y + t * dy - center.y))
      }
    return nearest
  })
  expect(distance).toBeCloseTo(1.8, 5)
  expect(errors).toEqual([])
})

for (const mode of ['rectangle', 'free']) {
  test(`one-shot ${mode} cuts follow the touched caption through drag, resize and reload, then editing restores complete new text`, async ({
    page,
  }) => {
    const errors = await open(page)
    await seed(page)
    await page.getByRole('button', { name: '擦除', exact: true }).click()
    if (mode === 'free') {
      await page.getByRole('button', { name: '自由', exact: true }).click()
      await drag(page, [35, 100], [35, 120])
    } else await drag(page, [25, 104], [40, 118])
    await expect.poll(() => count(page)).toBe(1)
    const cuts = await page.evaluate(() => window.__tables.payload.erasures[0].cuts)
    expect(cuts.some((c) => c.objectId === 'label')).toBe(true)
    const caption = page.locator('[data-free-text-box="label"]')
    await page.getByRole('button', { name: '选择', exact: true }).click()
    await caption.click()
    const path = await caption.locator('.free-text-ink path').getAttribute('d')
    const handle = await page.getByRole('button', { name: '移动文本框', exact: true }).boundingBox()
    const k = await page
      .locator('[data-paper="0"]')
      .evaluate((el) => el.getBoundingClientRect().width / 163)
    await page.mouse.move(handle.x + 8, handle.y + 8)
    await page.mouse.down()
    await page.mouse.move(handle.x + 8 + 20 * k, handle.y + 8 + 25 * k, { steps: 6 })
    // The ink is already cut during the drag; there is no stationary mask.
    await expect(caption.locator('.free-text-ink')).not.toHaveAttribute('style', /mask-image/)
    expect(await caption.locator('.free-text-ink path').getAttribute('d')).toBe(path)
    await page.mouse.up()
    await expect
      .poll(() =>
        page.evaluate(() => window.__tables.payload.tables.find((t) => t.id === 'label').y),
      )
      .toBeGreaterThan(125)
    await expect(caption.locator('.free-text-ink path')).toHaveAttribute('d', path)
    const resize = await page
      .getByRole('button', { name: '调整文本框大小', exact: true })
      .boundingBox()
    await page.mouse.move(resize.x + resize.width / 2, resize.y + resize.height / 2)
    await page.mouse.down()
    await page.mouse.move(
      resize.x + resize.width / 2 + 12 * k,
      resize.y + resize.height / 2 + 7 * k,
      { steps: 6 },
    )
    await page.mouse.up()
    await expect
      .poll(() =>
        page.evaluate(
          () => window.__tables.payload.tables.find((t) => t.id === 'label').columnWidths[0],
        ),
      )
      .toBeGreaterThan(98)
    await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
    await expect
      .poll(() => page.evaluate(() => window.__tables.saved.erasures[0].cuts))
      .toEqual(cuts)
    await page.evaluate(() => window.__tables.menu['menu:action']('open'))
    await expect(caption.locator('.free-text-ink')).not.toHaveAttribute('style', /mask-image/)
    await caption.dblclick()
    await page.getByLabel('文本框文字').fill('图一 霍尔效应原理图')
    await page.getByLabel('文本框文字').press('Enter')
    await expect
      .poll(() =>
        page.evaluate(
          () =>
            window.__tables.document.pages[0].tables.find((t) => t.id === 'label').cells[0].text,
        ),
      )
      .toBe('图一 霍尔效应原理图')
    await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
    await expect
      .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
      .toContain('A1 handwriting')
    const missing = await page.evaluate(() => {
      const p = window.__tables.exported.document.pages[0],
        t = p.tables.find((t) => t.id === 'label'),
        keys = new Set(p.strokes.map((s) => JSON.stringify(s)))
      return t.strokes.filter(
        (s) => !keys.has(JSON.stringify(s.map((p) => ({ x: p.x + t.x, y: p.y + t.y })))),
      ).length
    })
    expect(missing).toBe(0)
    expect(errors).toEqual([])
  })
}
