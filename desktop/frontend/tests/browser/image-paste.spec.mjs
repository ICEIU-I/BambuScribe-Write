import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { test, open } from './tableHarness.mjs'

const png = readFileSync(
  new URL('../../../internal/engine/testdata/hall-fraction.png', import.meta.url),
)
async function paste(page) {
  await page.evaluate((base64) => {
    const data = new DataTransfer()
    data.items.add(
      new File([Uint8Array.from(atob(base64), (c) => c.charCodeAt(0))], 'formula.png', {
        type: 'image/png',
      }),
    )
    document
      .querySelector('.document-viewport')
      .dispatchEvent(
        new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: data }),
      )
  }, png.toString('base64'))
}
async function setup(page) {
  const errors = await open(page)
  await page.evaluate(() => {
    window.__modelCalls = 0
    for (const name of ['AgentConfig', 'SaveAgentConfig', 'RecognizeDiagram']) {
      window.go.application.App[name] = async () => {
        window.__modelCalls++
        throw new Error('Paste must not access the model')
      }
    }
  })
  return errors
}
test('pasting a formula uses raster OCR without model setup, supports undo, save and export', async ({
  page,
}) => {
  const errors = await setup(page)
  await paste(page)
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await expect
    .poll(() =>
      page.evaluate(() => window.__tables.document?.pages[0]?.images?.[0]?.strokes?.length ?? 0),
    )
    .toBeGreaterThan(0)
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  const asset = await page.evaluate(() => window.__tables.payload.images[0])
  expect(asset.diagram).toBeUndefined()
  expect(asset.width).toBe(80)
  expect(asset.height / asset.width).toBeCloseTo(98 / 170, 6)
  expect(await page.evaluate(() => window.__modelCalls)).toBe(0)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.images.length)).toBe(1)
  await page.getByRole('button', { name: '导出', exact: true }).click()
  await page.getByRole('button', { name: '导出当前页 G-code', exact: true }).click()
  await expect.poll(() => page.evaluate(() => !!window.__tables.exported)).toBe(true)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  expect(await page.evaluate(() => window.__tables.payload.images[0].data)).toBe(asset.data)
  expect(await page.evaluate(() => window.__modelCalls)).toBe(0)
  await page.screenshot({ path: `/tmp/bambuscribe-formula-paste-${test.info().project.name}.png` })
  expect(errors).toEqual([])
})

test('back-to-back pastes create separate images and undo separately', async ({ page }) => {
  const errors = await setup(page)
  await paste(page)
  await paste(page)
  await expect(page.locator('[data-image-id]')).toHaveCount(2)
  await expect.poll(() => page.evaluate(() => window.__tables.payload.images.length)).toBe(2)
  const assets = await page.evaluate(() => window.__tables.payload.images)
  expect(assets[0].id).not.toBe(assets[1].id)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  expect(await page.evaluate(() => window.__modelCalls)).toBe(0)
  expect(errors).toEqual([])
})

test('switching projects while reading a pasted file prevents a stale insert', async ({ page }) => {
  const errors = await setup(page)
  await page.evaluate(() => {
    const read = FileReader.prototype.readAsDataURL
    FileReader.prototype.readAsDataURL = function (file) {
      window.__finishRead = () => read.call(this, file)
      this.addEventListener('loadend', () => {
        window.__readFinished = true
      })
    }
  })
  await paste(page)
  await page.evaluate(() => window.__tables.menu['menu:action']('new'))
  await expect(page.locator('textarea.paper-input')).toBeEnabled()
  await page.evaluate(() => window.__finishRead())
  await expect.poll(() => page.evaluate(() => !!window.__readFinished)).toBe(true)
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  expect(await page.evaluate(() => window.__modelCalls)).toBe(0)
  expect(errors).toEqual([])
})
