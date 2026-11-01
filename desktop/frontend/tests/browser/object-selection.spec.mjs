import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

async function seed(page, zoom = '75') {
  const errors = await open(page)
  await page.evaluate(() => {
    const c = document.createElement('canvas')
    c.width = 200
    c.height = 100
    const ctx = c.getContext('2d')
    ctx.fillStyle = 'white'
    ctx.fillRect(0, 0, 200, 100)
    ctx.strokeStyle = 'black'
    ctx.lineWidth = 2
    ctx.strokeRect(5, 5, 190, 90)
    const table = {
      id: 'data',
      page: 0,
      x: 75,
      y: 40,
      columnWidths: [25, 25],
      rowHeights: [20],
      cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 2, text: '合并单元格' }],
      style: 'grid',
      fontSize: 5,
      padding: 1,
      align: 'center',
      verticalAlign: 'middle',
    }
    window.__tables.saved = {
      ...window.__tables.preferences,
      text: '正文只按原来的文字方式复制。\n'.repeat(18),
      images: [
        {
          id: 'diagram',
          page: 0,
          x: 20,
          y: 40,
          width: 40,
          height: 20,
          rotation: 15,
          mode: 'line',
          data: c.toDataURL(),
          textRegions: [],
        },
      ],
      tables: [
        table,
        {
          ...table,
          id: 'caption',
          kind: 'textbox',
          x: 20,
          y: 80,
          columnWidths: [70],
          rowHeights: [12],
          style: 'none',
          cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '图片和表格的说明' }],
        },
      ],
    }
    window.__tables.menu['menu:action']('open')
  })
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.objectLayout?.length))
    .toBe(3)
  await page.getByLabel('页面缩放').selectOption(zoom)
  await show(page, 0)
  // Opening a project remounts the canvas. Observe copy after its handler.
  await page.evaluate(() => {
    window.addEventListener('copy', (e) => {
      window.__copied = Object.fromEntries(
        [...e.clipboardData.types].map((t) => [t, e.clipboardData.getData(t)]),
      )
    })
  })
  return errors
}
async function show(page, index) {
  await page.locator(`[data-paper="${index}"]`).evaluate((paper) => {
    const view = paper.closest('.document-viewport')
    view.scrollTop += paper.getBoundingClientRect().top - view.getBoundingClientRect().top - 25
  })
}
async function point(page, x, y, index = 0) {
  return page.locator(`[data-paper="${index}"]`).evaluate(
    (el, p) => {
      const r = el.getBoundingClientRect(),
        k = r.width / 163
      return { x: r.left + p.x * k, y: r.top + p.y * k }
    },
    { x, y },
  )
}
async function drag(page, a = [132, 24], b = [10, 100]) {
  const start = await point(page, ...a),
    end = await point(page, ...b)
  await page.mouse.move(start.x, start.y)
  await page.mouse.down()
  await page.mouse.move(end.x, end.y, { steps: 8 })
  await page.mouse.up()
}
const selected = (page) => page.locator('.marquee-object')
const payload = (page) => page.evaluate(() => window.__tables.payload)
async function clipboardText(page) {
  // Native text is written after the copy event's default action, so verify
  // it through a real paste into an unrelated input, not the copy observer.
  await page.evaluate(() => {
    const sink = document.createElement('textarea')
    sink.id = 'clipboard-sink'
    sink.style.cssText = 'position:fixed;right:0;bottom:0'
    document.body.append(sink)
    sink.focus()
  })
  await page.keyboard.press('Meta+v')
  return page.locator('#clipboard-sink').evaluate((sink) => {
    const value = sink.value
    sink.remove()
    return value
  })
}

test('real blank-area marquee copies images, merged tables and textboxes, preserves relative layout and has one undo step', async ({
  page,
}) => {
  const errors = await seed(page),
    before = await payload(page)
  await drag(page)
  await expect(selected(page)).toHaveCount(3)
  await expect(page.locator('.document-viewport')).toBeFocused()
  await page.keyboard.press('Meta+c')
  const copied = await page.evaluate(() =>
    JSON.parse(window.__copied['application/x-bambuscribe-objects-v1']),
  )
  expect(copied.images).toHaveLength(1)
  expect(copied.tables).toHaveLength(2)
  expect(copied.tables[1].kind).toBe('textbox')
  expect(copied.tables[0].cells[0].colSpan).toBe(2)
  expect(copied.text).toBeUndefined()
  // An explicit near-bottom target sends the intact group to the next page.
  await show(page, 0)
  const target = await point(page, 150, 220)
  await page.mouse.click(target.x, target.y)
  await page.keyboard.press('Meta+v')
  await expect.poll(async () => (await payload(page)).tables.length).toBe(4)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.objectLayout?.length))
    .toBe(6)
  const after = await payload(page),
    im = after.images[1],
    grid = after.tables[2],
    caption = after.tables[3]
  expect(after.text).toBe(before.text)
  expect(im.page).toBe(1)
  expect(grid.page).toBe(1)
  expect(caption.page).toBe(1)
  expect(im.x).toBeCloseTo(before.images[0].x, 8)
  expect(grid.x).toBeCloseTo(before.tables[0].x, 8)
  expect(caption.x).toBeCloseTo(before.tables[1].x, 8)
  expect(grid.x - im.x).toBeCloseTo(before.tables[0].x - before.images[0].x, 6)
  expect(caption.y - im.y).toBeCloseTo(before.tables[1].y - before.images[0].y, 6)
  expect(new Set([...after.images, ...after.tables].map((o) => o.id)).size).toBe(6)
  await expect(page.locator('[data-paper="1"] [data-image-id]')).toHaveCount(1)
  await expect(page.locator('[data-paper="1"] [data-free-text-box]')).toHaveCount(1)
  await page.keyboard.press('Meta+z')
  await expect.poll(async () => (await payload(page)).tables.length).toBe(2)
  expect((await payload(page)).images).toEqual(before.images)
  await page.keyboard.press('Meta+Shift+z')
  await expect.poll(async () => (await payload(page)).tables.length).toBe(4)
  expect((await payload(page)).tables).toEqual(after.tables)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.tables.length)).toBe(4)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.objectLayout?.length))
    .toBe(6)
  expect((await payload(page)).tables).toEqual(after.tables)
  await page.screenshot({ path: test.info().outputPath('marquee-pasted.png') })
  expect(errors).toEqual([])
})

test('wide A4 tables and captions preserve original X when pasting near the right edge and repeatedly', async ({
  page,
}) => {
  const errors = await seed(page)
  await page.evaluate(() => {
    const table = window.__tables.saved.tables[0]
    window.__tables.saved = {
      ...window.__tables.saved,
      text: '\n'.repeat(31),
      images: [],
      layout: {
        ...window.__tables.saved.layout,
        paperWidth: 210,
        paperHeight: 297,
        marginTop: 25.4,
        marginBottom: 25.4,
        lineCount: 30,
      },
      tables: [
        { ...table, id: 'wide', x: 20.262, y: 49.634, columnWidths: [80, 90] },
        { ...window.__tables.saved.tables[1], id: 'wide-caption', x: 89.434, y: 35.375 },
      ],
    }
    // Flushing the current project before opening must not overwrite the fixture.
    const project = window.__tables.saved
    window.go.application.App.OpenProject = async () => ({ path: '/tmp/wide.ccf', project })
    window.__tables.menu['menu:action']('open')
  })
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.settings.paperWidth))
    .toBe(210)
  await page.getByLabel('页面缩放').selectOption('50')
  await show(page, 0)
  const pos = async (x, y, index) =>
    page.locator(`[data-paper="${index}"]`).evaluate(
      (el, p) => {
        const r = el.getBoundingClientRect(),
          k = r.width / 210
        return { x: r.left + p.x * k, y: r.top + p.y * k }
      },
      { x, y },
    )
  const a = await pos(10, 30, 0),
    b = await pos(200, 80, 0)
  await page.mouse.move(a.x, a.y)
  await page.mouse.down()
  await page.mouse.move(b.x, b.y, { steps: 6 })
  await page.mouse.up()
  await expect(selected(page)).toHaveCount(2)
  await page.keyboard.press('Meta+c')
  await show(page, 1)
  const target = await pos(195, 40, 1)
  await page.mouse.click(target.x, target.y)
  await page.keyboard.press('Meta+v')
  await expect.poll(async () => (await payload(page)).tables.length).toBe(4)
  const check = async () => {
    const objects = (await payload(page)).tables
    objects.forEach((t, i) => expect(t.x).toBeCloseTo(i % 2 ? 89.434 : 20.262, 8))
  }
  await check()
  await page.keyboard.press('Meta+v')
  await expect.poll(async () => (await payload(page)).tables.length).toBe(6)
  await check()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.objectLayout?.length))
    .toBe(6)
  await page.screenshot({ path: test.info().outputPath('a4-paste-columns.png') })
  expect(errors).toEqual([])
})

for (const zoom of ['50', '100'])
  test(`reverse marquee at ${zoom}% and Escape cancel leave the document intact`, async ({
    page,
  }) => {
    const errors = await seed(page, zoom),
      before = await payload(page)
    await drag(page, [132, 100], [10, 24])
    await expect(selected(page)).toHaveCount(3)
    await page.screenshot({ path: test.info().outputPath('marquee-selected.png') })
    await page.keyboard.press('Escape')
    await expect(page.locator('.paper-marquee-layer')).toHaveCount(0)
    expect(await payload(page)).toEqual(before)
    expect(errors).toEqual([])
  })

test('body dragging still selects and copies text; a blank single click still places the caret', async ({
  page,
}) => {
  const errors = await seed(page)
  const cells = await page.evaluate(() => window.__tables.document.pages[0].cells.slice(0, 9))
  const a = await point(page, cells[0].x + 1, cells[0].y + 2)
  const b = await point(page, cells[8].x + cells[8].width - 1, cells[8].y + 2)
  await page.mouse.move(a.x, a.y)
  await page.mouse.down()
  await page.mouse.move(b.x, b.y, { steps: 6 })
  await page.mouse.up()
  await expect(page.getByLabel('书写内容')).toBeFocused()
  await expect(page.locator('.paper-marquee-layer')).toHaveCount(0)
  const range = await page
    .getByLabel('书写内容')
    .evaluate((el) => [el.selectionStart, el.selectionEnd])
  expect(range[1]).toBeGreaterThan(range[0])
  await page.keyboard.press('Meta+c')
  const data = await page.evaluate(() => window.__copied)
  expect(data['application/x-bambuscribe-objects-v1']).toBeUndefined()
  expect(await clipboardText(page)).toBe((await payload(page)).text.slice(...range))
  const blank = await point(page, 150, 100)
  await page.mouse.click(blank.x, blank.y)
  await expect(page.getByLabel('书写内容')).toBeFocused()
  expect(
    await page.getByLabel('书写内容').evaluate((el) => el.selectionStart === el.selectionEnd),
  ).toBe(true)
  expect(errors).toEqual([])
})

test('partial rectangle selects a whole textbox; empty copy and pointer cancellation never copy the body', async ({
  page,
}) => {
  const errors = await seed(page),
    before = await payload(page)
  await drag(page, [15, 70], [22, 84])
  await expect(selected(page)).toHaveCount(1)
  await expect(selected(page)).toHaveAttribute('data-marquee-object', 'caption')
  await page.keyboard.press('Meta+c')
  const copied = await page.evaluate(() =>
    JSON.parse(window.__copied['application/x-bambuscribe-objects-v1']),
  )
  expect(copied.images).toHaveLength(0)
  expect(copied.tables[0].kind).toBe('textbox')
  await drag(page, [135, 110], [155, 125])
  await expect(selected(page)).toHaveCount(0)
  await page.keyboard.press('Meta+c')
  await expect(page.locator('.notice-text')).toHaveText('框内没有可复制的图片、表格或文本框。')
  expect(await page.evaluate(() => window.__copied['text/plain'])).toBeUndefined()
  const a = await point(page, 132, 24),
    b = await point(page, 10, 100)
  await page.mouse.move(a.x, a.y)
  await page.mouse.down()
  await page.mouse.move(b.x, b.y)
  await expect(selected(page)).toHaveCount(3)
  await page.locator('.document-stack').dispatchEvent('pointercancel')
  await page.mouse.up()
  await expect(page.locator('.paper-marquee-layer')).toHaveCount(0)
  expect(await payload(page)).toEqual(before)
  expect(errors).toEqual([])
})

test('textbox editing uses normal text copy and unrelated clipboard text does not paste a stale object group', async ({
  page,
}) => {
  const errors = await seed(page),
    before = await payload(page)
  await drag(page)
  await page.keyboard.press('Meta+c')
  await page.keyboard.press('Escape')
  await page.locator('[data-free-text-box="caption"]').dblclick()
  const editor = page.getByLabel('文本框文字', { exact: true })
  await expect(editor).toBeFocused()
  await editor.press('Meta+a')
  await editor.press('Meta+c')
  expect(await clipboardText(page)).toBe('图片和表格的说明')
  expect(
    await page.evaluate(() => window.__copied['application/x-bambuscribe-objects-v1']),
  ).toBeUndefined()
  // Pasting into the unrelated sink blurred and saved the textbox editor.
  const blank = await point(page, 150, 100)
  await page.mouse.click(blank.x, blank.y)
  await page.keyboard.press('Meta+v')
  await expect
    .poll(async () => (await payload(page)).text.length)
    .toBeGreaterThan(before.text.length)
  expect((await payload(page)).tables).toEqual(before.tables)
  expect((await payload(page)).images).toEqual(before.images)
  expect(errors).toEqual([])
})

test('marquee auto scrolls within its starting page; disabled and inactive canvases ignore selection and paste', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1000, height: 650 })
  await page.goto('/tests/browser/image.html')
  await page.evaluate(() => {
    const s = window.imageFixture.state
    s.zoom = 100
    s.index = 0
    s.images[0] = { ...s.images[0], page: 0, y: 200, height: 20 }
  })
  await show(page, 0)
  const before = await page.evaluate(() => JSON.parse(JSON.stringify(window.imageFixture.state)))
  const a = await point(page, 10, 20),
    view = await page.locator('.document-viewport').boundingBox()
  await page.mouse.move(a.x, a.y)
  await page.mouse.down()
  await page.mouse.move(a.x + 400, view.y + view.height - 3)
  await expect
    .poll(() => page.locator('.document-viewport').evaluate((v) => v.scrollTop))
    .toBeGreaterThan(200)
  await expect(selected(page)).toHaveCount(1)
  await page.mouse.up()
  await expect(page.locator('[data-paper="0"] .paper-marquee-layer')).toHaveCount(1)
  await page.keyboard.press('Meta+c')
  for (const key of ['disabled', 'active']) {
    await page.evaluate((key) => {
      window.imageFixture.state[key] = key === 'disabled'
    }, key)
    await expect(page.locator('.paper-marquee-layer')).toHaveCount(0)
    await page.keyboard.press('Meta+v')
    expect(await page.evaluate(() => window.imageFixture.state.images)).toEqual(before.images)
    await show(page, 0)
    await drag(page)
    await expect(page.locator('.paper-marquee-layer')).toHaveCount(0)
    await page.evaluate((key) => {
      window.imageFixture.state[key] = key !== 'disabled'
    }, key)
  }
})

const objectFrames = (page) =>
  page.evaluate(() =>
    ['[data-image-id="diagram"]', '[data-table="data"]', '[data-free-text-box="caption"]'].map(
      (s) => {
        const el = document.querySelector(s)
        return { x: parseFloat(el.style.left), y: parseFloat(el.style.top) }
      },
    ),
  )
const members = (p) => [...p.images, ...p.tables]
function expectMoved(before, after, dx, dy, page = 0) {
  expect(after.text).toBe(before.text)
  expect(after.erasures).toEqual(before.erasures)
  members(after).forEach((o, i) => {
    const original = members(before)[i]
    expect(o).toEqual({ ...original, page, x: expect.any(Number), y: expect.any(Number) })
    expect(o.x).toBeCloseTo(original.x + dx, 6)
    expect(o.y).toBeCloseTo(original.y + dy, 6)
  })
}
for (const zoom of ['50', '100'])
  test(`marquee group movement at ${zoom}% previews all objects, keeps selection and saves one undo step`, async ({
    page,
  }) => {
    const errors = await seed(page, zoom),
      before = await payload(page)
    await drag(page)
    const frames = await objectFrames(page),
      a = await point(page, 90, 50),
      b = await point(page, 100, 65)
    await page.mouse.move(a.x, a.y)
    await page.mouse.down()
    await page.mouse.move(b.x, b.y, { steps: 8 })
    expect(await payload(page)).toEqual(before) // no intermediate writes
    const preview = await objectFrames(page),
      k = ((96 / 25.4) * Number(zoom)) / 100
    preview.forEach((o, i) => {
      expect(Math.abs(o.x - frames[i].x - 10 * k)).toBeLessThan(1.1)
      expect(Math.abs(o.y - frames[i].y - 15 * k)).toBeLessThan(1.1)
      expect(o.x - frames[i].x).toBeCloseTo(preview[0].x - frames[0].x, 2)
      expect(o.y - frames[i].y).toBeCloseTo(preview[0].y - frames[0].y, 2)
    })
    await page.screenshot({ path: test.info().outputPath('group-moving.png') })
    await page.mouse.up()
    await expect
      .poll(async () => Math.abs((await payload(page)).tables[0].x - 85))
      .toBeLessThan(0.6)
    const after = await payload(page)
    expectMoved(
      before,
      after,
      after.images[0].x - before.images[0].x,
      after.images[0].y - before.images[0].y,
    )
    await expect
      .poll(() =>
        page.evaluate(() => window.__tables.document.objectLayout.find((o) => o.id === 'data').x),
      )
      .toBeCloseTo(after.tables[0].x, 6)
    await expect(selected(page)).toHaveCount(3)
    // Another member can drag the same selection without reselecting it.
    await drag(page, [40, 60], [45, 65])
    await expect
      .poll(async () => Math.abs((await payload(page)).images[0].x - 35))
      .toBeLessThan(0.7)
    const repeated = await payload(page)
    expectMoved(
      after,
      repeated,
      repeated.images[0].x - after.images[0].x,
      repeated.images[0].y - after.images[0].y,
    )
    await page.keyboard.press('Meta+z')
    await expect.poll(() => payload(page)).toEqual(after)
    await page.keyboard.press('Meta+z')
    await expect.poll(() => payload(page)).toEqual(before)
    await page.keyboard.press('Meta+Shift+z')
    await expect.poll(() => payload(page)).toEqual(after)
    await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
    await expect.poll(() => page.evaluate(() => window.__tables.saved.tables)).toEqual(after.tables)
    await page.evaluate(() => window.__tables.menu['menu:action']('open'))
    await expect.poll(() => payload(page)).toEqual(after)
    await expect
      .poll(() =>
        page.evaluate(() => window.__tables.document.objectLayout.find((o) => o.id === 'data').x),
      )
      .toBeCloseTo(after.tables[0].x, 6)
    await page.getByRole('button', { name: '导出', exact: true }).click()
    const exportButton = page.getByRole('button', { name: '导出当前页 G-code' })
    await expect(exportButton).toBeEnabled()
    await exportButton.click()
    await expect
      .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
      .toContain('A1 handwriting')
    const exported = await page.evaluate(() => window.__tables.exported.document)
    expect(
      exported.objectLayout
        .filter((o) => ['data', 'caption', 'diagram'].includes(o.id))
        .map((o) => [o.id, o.x, o.y]),
    ).toEqual(members(after).map((o) => [o.id, o.x, o.y]))
    expect(errors).toEqual([])
  })

test('marquee group movement cancels with Escape, pointer cancellation or capture loss and ignores a click', async ({
  page,
}) => {
  const errors = await seed(page),
    before = await payload(page)
  for (const cancel of ['click', 'Escape', 'pointercancel', 'capture']) {
    await drag(page)
    const frames = await objectFrames(page),
      a = await point(page, 35, 85),
      b = await point(page, 43, 99)
    await page.mouse.move(a.x, a.y)
    await page.mouse.down()
    if (cancel !== 'click') {
      await page.mouse.move(b.x, b.y, { steps: 6 })
      if (cancel === 'Escape') await page.keyboard.press('Escape')
      else if (cancel === 'pointercancel')
        await page.locator('.document-stack').dispatchEvent('pointercancel')
      else
        await page.locator('.document-stack').evaluate((el) => {
          for (let i = 1; i < 10; i++) if (el.hasPointerCapture(i)) el.releasePointerCapture(i)
        })
    }
    await page.mouse.up()
    expect(await payload(page)).toEqual(before)
    expect(await objectFrames(page)).toEqual(frames)
    await page.keyboard.press('Escape')
  }
  expect(errors).toEqual([])
})

test('marquee group movement clamps all objects together and leaves unselected import siblings alone', async ({
  page,
}) => {
  const errors = await seed(page, '50')
  await page.evaluate(() => {
    const p = window.__tables.saved
    const next = {
      ...p,
      tables: [
        ...p.tables.map((t) => ({ ...t, groupId: 'imported' })),
        { ...p.tables[1], id: 'unselected', groupId: 'imported', x: 10, y: 180 },
      ],
    }
    window.go.application.App.OpenProject = async () => ({
      path: '/tmp/boundary.ccf',
      project: next,
    })
    window.__tables.menu['menu:action']('open')
  })
  await expect.poll(async () => (await payload(page)).tables.length).toBe(3)
  await expect.poll(() => page.evaluate(() => window.__tables.document.objectLayout.length)).toBe(4)
  await show(page, 0)
  const before = await payload(page)
  await drag(page)
  await expect(selected(page)).toHaveCount(3)
  await drag(page, [90, 50], [250, 210])
  await expect.poll(async () => (await payload(page)).tables[0].x).toBeCloseTo(113, 6)
  const after = await payload(page),
    dx = after.images[0].x - before.images[0].x,
    dy = after.images[0].y - before.images[0].y
  expectMoved(
    { ...before, tables: before.tables.slice(0, 2) },
    { ...after, tables: after.tables.slice(0, 2) },
    dx,
    dy,
  )
  expect(after.tables[2]).toEqual(before.tables[2])
  expect(after.tables[1].y + after.tables[1].rowHeights[0]).toBeCloseTo(240, 6)
  await page.keyboard.press('Meta+z')
  await expect.poll(() => payload(page)).toEqual(before)
  expect(errors).toEqual([])
})

test('marquee group movement follows scrolling across pages with a single drop and undo', async ({
  page,
}) => {
  const errors = await seed(page, '75'),
    before = await payload(page)
  await drag(page)
  const a = await point(page, 90, 50)
  await page.mouse.move(a.x, a.y)
  await page.mouse.down()
  await page.mouse.move(a.x + 10, a.y + 10)
  await show(page, 1)
  const b = await point(page, 100, 65, 1)
  await page.mouse.move(b.x, b.y, { steps: 8 })
  expect(await payload(page)).toEqual(before)
  await page.screenshot({ path: test.info().outputPath('group-next-page.png') })
  await page.mouse.up()
  await expect.poll(async () => (await payload(page)).tables[0].page).toBe(1)
  const after = await payload(page)
  expect(Math.abs(after.images[0].x - before.images[0].x - 10)).toBeLessThan(0.6)
  expect(Math.abs(after.images[0].y - before.images[0].y - 15)).toBeLessThan(0.6)
  expectMoved(
    before,
    after,
    after.images[0].x - before.images[0].x,
    after.images[0].y - before.images[0].y,
    1,
  )
  await expect(page.locator('[data-paper="1"] .marquee-object')).toHaveCount(3)
  await expect(page.locator('[data-paper="1"] [data-image-id]')).toHaveCount(1)
  await page.keyboard.press('Meta+z')
  await expect.poll(() => payload(page)).toEqual(before)
  expect(errors).toEqual([])
})

test('marquee group movement auto scrolls at the viewport edge without committing intermediate positions', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1100, height: 650 })
  const errors = await seed(page, '100'),
    before = await payload(page)
  await drag(page)
  const a = await point(page, 90, 50),
    view = await page.locator('.document-viewport').boundingBox()
  await page.mouse.move(a.x, a.y)
  await page.mouse.down()
  await page.mouse.move(a.x, view.y + view.height - 4, { steps: 6 })
  await expect
    .poll(() => page.locator('.document-viewport').evaluate((el) => el.scrollTop))
    .toBeGreaterThan(1000)
  expect(await payload(page)).toEqual(before)
  const b = await point(page, 100, 65, 1)
  await page.mouse.move(b.x, b.y)
  await page.mouse.up()
  await expect.poll(async () => (await payload(page)).images[0].page).toBe(1)
  const after = await payload(page)
  expectMoved(
    before,
    after,
    after.images[0].x - before.images[0].x,
    after.images[0].y - before.images[0].y,
    1,
  )
  await page.keyboard.press('Meta+z')
  await expect.poll(() => payload(page)).toEqual(before)
  expect(errors).toEqual([])
})
