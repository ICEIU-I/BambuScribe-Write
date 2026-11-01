import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { test, open } from './tableHarness.mjs'

const note = 'The shallow rays near the center of the second lens are faint.'
const data = `data:image/png;base64,${readFileSync(new URL('../../../internal/engine/testdata/hall-switch.png', import.meta.url)).toString('base64')}`
const image = (id = 'warning-image', page = 0, uncertainties = [note]) => ({
  id,
  page,
  data,
  x: 15,
  y: 25,
  width: 80,
  height: 48,
  rotation: 0,
  mode: 'line',
  diagram: {
    width: 100,
    height: 60,
    strokes: [
      {
        id: 'line',
        points: [
          { x: 10, y: 10 },
          { x: 90, y: 10 },
        ],
      },
    ],
    circles: [],
    labels: [],
    uncertainties,
  },
})
async function load(page, images = [image()]) {
  await page.evaluate((images) => {
    window.__tables.document = null
    window.__tables.saved = {
      version: 7,
      text: '正文\f第二页',
      images,
      tables: [],
      layout: window.__tables.preferences.layout,
      writing: window.__tables.preferences.writing,
    }
    // The old project's close-time autosave may finish before OpenProject.
    // Keep the selected file independent from the autosave fixture.
    const project = structuredClone(window.__tables.saved)
    const api = window.go.application.App
    const original = api.OpenProject
    api.OpenProject = async () => {
      api.OpenProject = original
      return { path: '/tmp/table.ccf', project }
    }
    window.__tables.menu['menu:action']('open')
  }, images)
  await expect.poll(() => page.evaluate(() => window.__tables.document?.pages.length)).toBe(2)
  await expect
    .poll(() =>
      page.evaluate(() =>
        window.__tables.document.pages.flatMap((page, index) =>
          (page.images ?? []).map((image) => [image.id, index]),
        ),
      ),
    )
    .toEqual(images.map((image) => [image.id, image.page]))
  await page.getByRole('button', { name: /^第 1 页/ }).click()
  await expect(page.getByRole('button', { name: /^第 1 页/ })).toHaveAttribute(
    'aria-current',
    'page',
  )
  for (const image of images.filter((image) => image.page === 0))
    await expect(page.locator(`[data-paper="0"] [data-image-id="${image.id}"]`)).toHaveCount(1)
}
const warning = (page) => page.locator('.document-alert[role="status"]')
const close = (page) => page.getByRole('button', { name: '关闭页面提示', exact: true }).click()

test('dismissed image warning stays closed through relayout, tabs, save/reopen and restart', async ({
  page,
}) => {
  const errors = await open(page)
  await load(page)
  await expect(warning(page)).toContainText(note)
  await close(page)
  await expect(warning(page)).toHaveCount(0)
  await page.getByRole('textbox', { name: '书写内容', exact: true }).fill('编辑正文\f第二页')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0].text))
    .toContain('编辑正文')
  await expect(warning(page)).toHaveCount(0)
  await page.getByRole('tab', { name: '设备台', exact: true }).click()
  await page.getByRole('tab', { name: '书写台', exact: true }).click()
  await expect(warning(page)).toHaveCount(0)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.text)).toContain('编辑正文')
  await page.evaluate(() => {
    window.__tables.document = null
    window.__tables.menu['menu:action']('open')
  })
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.warnings?.length))
    .toBe(1)
  await expect(warning(page)).toHaveCount(0)
  await page.reload()
  await expect(page.getByRole('button', { name: '表格', exact: true })).toBeEnabled()
  await load(page)
  await expect(warning(page)).toHaveCount(0)
  expect(errors).toEqual([])
})

test('new warnings and another image remain visible while the dismissed source moves pages', async ({
  page,
}) => {
  await open(page)
  await load(page)
  await close(page)
  await load(page, [image('warning-image', 1)])
  await page.getByRole('button', { name: /^第 2 页/ }).click()
  await expect(warning(page)).toHaveCount(0)
  await load(page, [image('another-image')])
  await expect(warning(page)).toContainText(note)
  await close(page)
  const changed = 'The lens edge is unclear; please check the original image.'
  await load(page, [image('warning-image', 0, [note, changed])])
  await expect(warning(page)).toContainText(changed)
  await expect(warning(page)).not.toContainText(note)
  await close(page)
  await page.getByRole('textbox', { name: '书写内容', exact: true }).fill('🙂\f第二页')
  await page.getByRole('button', { name: /^第 1 页.*缺字/ }).click()
  await expect(page.locator('.document-alert[role="alert"]')).toContainText('缺字')
})

for (const unavailable of [false, true]) {
  test(`warning dismissal still works with ${unavailable ? 'unavailable' : 'corrupt'} local storage`, async ({
    page,
  }) => {
    const errors = await open(page)
    await page.evaluate((unavailable) => {
      if (unavailable)
        Object.defineProperty(window, 'localStorage', {
          get() {
            throw new DOMException('Storage disabled', 'SecurityError')
          },
        })
      else localStorage.setItem('bambuscribe.dismissed-image-warnings.v1', '{broken')
    }, unavailable)
    await load(page)
    await expect(warning(page)).toContainText(note)
    await close(page)
    await expect(warning(page)).toHaveCount(0)
    await page.getByRole('tab', { name: '设备台', exact: true }).click()
    await page.getByRole('tab', { name: '书写台', exact: true }).click()
    await expect(warning(page)).toHaveCount(0)
    expect(errors).toEqual([])
  })
}
