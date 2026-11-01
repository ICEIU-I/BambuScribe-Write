import { expect } from '@playwright/test'
import { tmpdir } from 'node:os'
import { test, open } from './tableHarness.mjs'

test('body remains in whole rows below figures after moving, undo, save and export', async ({
  page,
}) => {
  const errors = await open(page)
  const text = '速率为V，载流子受到磁场作用。\n正文继续。'
  await page.evaluate((text) => {
    const canvas = document.createElement('canvas')
    canvas.width = 200
    canvas.height = 100
    const ctx = canvas.getContext('2d')
    ctx.fillStyle = 'white'
    ctx.fillRect(0, 0, 200, 100)
    ctx.strokeStyle = 'black'
    ctx.lineWidth = 2
    ctx.strokeRect(0, 0, 200, 100)
    const image = (id, x, width) => ({
      id,
      x,
      y: 0,
      width,
      height: 40,
      page: 0,
      rotation: 0,
      mode: 'line',
      data: canvas.toDataURL(),
      textRegions: [],
    })
    window.__tables.saved = {
      ...window.__tables.preferences,
      text,
      images: [image('left-figure', 0, 100), image('right-figure', 110, 53)],
    }
    window.__tables.menu['menu:action']('open')
  }, text)
  await expect.poll(() => page.evaluate(() => window.__tables.document?.pages[0].text)).toBe(text)
  const initial = await page.evaluate(() => window.__tables.document)
  expect(initial.pages[0].cells[0].x).toBe(0)
  expect(initial.pages[0].cells[0].y).toBe(45)
  expect(initial.pages[0].cells.every((c) => c.y >= 45)).toBe(true)
  expect(initial.pages[0].cells[1].y).toBe(initial.pages[0].cells[0].y)
  const figure = page.locator('[data-image-id="right-figure"]')
  const bounds = await figure.boundingBox()
  const paper = await page.locator('[data-paper="0"]').boundingBox()
  await page.mouse.move(bounds.x + bounds.width / 2, bounds.y + bounds.height / 2)
  await page.mouse.down()
  await page.mouse.move(
    bounds.x + bounds.width / 2,
    bounds.y + bounds.height / 2 + (75 * paper.width) / 163,
    { steps: 8 },
  )
  await page.mouse.up()
  await expect
    .poll(() =>
      page.evaluate(
        () => window.__tables.document?.pages[0].images.find((im) => im.id === 'right-figure')?.y,
      ),
    )
    .toBeGreaterThan(70)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0].cells[0].y))
    .toBe(45)
  expect(await page.evaluate(() => window.__tables.document.pages[0].cells[0].x)).toBe(0)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0].cells))
    .toEqual(initial.pages[0].cells)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.text)).toBe(text)
  const reopened = page.waitForResponse('**/__table-preview')
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await reopened
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0].cells))
    .toEqual(initial.pages[0].cells)
  await page.getByRole('button', { name: '导出', exact: true }).click()
  const exportPage = page.getByRole('button', { name: /导出当前页/ })
  await expect(exportPage).toBeEnabled()
  await exportPage.click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode))
    .toContain('A1 handwriting')
  await page.screenshot({
    path: `${tmpdir()}/bambuscribe-readable-wrap-${test.info().project.name}.png`,
  })
  expect(errors).toEqual([])
})
