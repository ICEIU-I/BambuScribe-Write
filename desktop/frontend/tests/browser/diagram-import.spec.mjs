import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { test, open } from './tableHarness.mjs'
const scene = JSON.parse(
  readFileSync(new URL('../../../docs/diagram-fit-2026-10-09/first.json', import.meta.url)),
)
const png = readFileSync(
  new URL('../../../internal/engine/testdata/hall-switch.png', import.meta.url),
)

async function setup(page, delayed = false) {
  const errors = await open(page)
  await page.evaluate(
    ({ scene, delayed }) => {
      window.__diagram = { calls: 0, cancels: 0, delayed, scene }
      Object.assign(window.go.application.App, {
        AgentConfig: async () => ({
          baseURL: 'https://example.invalid/v1',
          model: 'gpt-6.1-sol',
          protocol: 'responses',
          hasKey: true,
        }),
        SaveAgentConfig: async () => {},
        RecognizeDiagram: async () => {
          window.__diagram.calls++
          if (window.__diagram.delayed)
            await new Promise((resolve, reject) =>
              Object.assign(window.__diagram, { resolve, reject }),
            )
          return window.__diagram.scene
        },
        CancelDiagramRecognition: async () => {
          window.__diagram.cancels++
        },
      })
    },
    { scene, delayed },
  )
  return errors
}
async function menu(page) {
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入图像', exact: true }).click()
}
async function start(page) {
  await menu(page)
  await page
    .getByLabel('图像文件', { exact: true })
    .setInputFiles({ name: 'diagram.png', mimeType: 'image/png', buffer: png })
  await page.getByRole('button', { name: '识别并插入', exact: true }).click()
}
test('one call inserts vector diagram directly, preserves resize/save/export and undo', async ({
  page,
}) => {
  const errors = await setup(page)
  await start(page)
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(1)
  expect(await page.evaluate(() => window.__tables.payload.images[0].diagram)).toEqual(scene)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  const handle = page.locator('.image-resize-handle')
  const box = await handle.boundingBox()
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await page.mouse.down()
  await page.mouse.move(box.x - 100, box.y - 80, { steps: 8 })
  await page.mouse.up()
  await expect
    .poll(() => page.evaluate(() => window.__tables.payload.images[0].width))
    .toBeLessThan(150)
  const size = await page.evaluate(() => window.__tables.payload.images[0])
  expect(size.height / size.width).toBeCloseTo(scene.height / scene.width, 6)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved?.images[0]?.diagram?.width))
    .toBe(1354)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => !!window.__tables.exported)).toBe(true)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.screenshot({ path: '/tmp/bambuscribe-diagram-agent-ui.png' })
  expect(errors).toEqual([])
})
test('explicit recognition cancellation prevents a late insert', async ({ page }) => {
  const errors = await setup(page, true)
  await start(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.getByRole('button', { name: '取消识别', exact: true }).click()
  await page.evaluate(() => window.__diagram.resolve())
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  expect(await page.evaluate(() => window.__diagram.cancels)).toBeGreaterThan(0)
  expect(errors).toEqual([])
})
test('project switch rejects an in-flight diagram result', async ({ page }) => {
  const errors = await setup(page, true)
  await start(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.getByRole('button', { name: '关闭图像导入', exact: true }).click()
  await expect(page.getByLabel('后台图像导入', { exact: true })).toBeVisible()
  await page.evaluate(() => window.__tables.menu['menu:action']('new'))
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  await page.evaluate(() => window.__diagram.resolve())
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  expect(await page.evaluate(() => window.__diagram.cancels)).toBe(1)
  await expect(page.getByLabel('后台图像导入', { exact: true })).not.toBeVisible()
  expect(errors).toEqual([])
})

test('closing and reopening keeps one background request and preserves ongoing edits', async ({
  page,
}) => {
  const errors = await setup(page, true)
  await page.clock.install()
  await start(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(1)
  const source = await page.getByAltText('待识别原图').getAttribute('src')
  await page.getByRole('button', { name: '关闭图像导入', exact: true }).click()
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  expect(await page.evaluate(() => window.__diagram.cancels)).toBe(0)
  await page.clock.fastForward(65000)
  await expect(page.getByRole('button', { name: '查看图像导入任务' })).toContainText('1:05')
  await page.locator('textarea.paper-input').fill('继续编辑')
  await menu(page)
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).toBeVisible()
  await expect(page.getByAltText('待识别原图')).toHaveAttribute('src', source)
  await expect(page.locator('.diagram-progress')).toContainText('1:05')
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  await page.screenshot({
    path: `/tmp/bambuscribe-diagram-background-${test.info().project.name}.png`,
  })
  await page.evaluate(() => window.__diagram.resolve())
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await expect(page.getByLabel('后台图像导入', { exact: true })).not.toBeVisible()
  await expect(page.locator('textarea.paper-input')).toHaveValue('继续编辑')
  expect(await page.evaluate(() => window.__diagram.cancels)).toBe(0)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  await expect(page.locator('textarea.paper-input')).toHaveValue('继续编辑')
  expect(errors).toEqual([])
})

test('explicit background cancellation drops a late result', async ({ page }) => {
  const errors = await setup(page, true)
  await start(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.getByRole('button', { name: '关闭图像导入', exact: true }).click()
  await page.getByRole('button', { name: '取消识别', exact: true }).click()
  await page.evaluate(() => window.__diagram.resolve())
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  expect(await page.evaluate(() => window.__diagram.cancels)).toBe(1)
  await expect(page.getByLabel('后台图像导入', { exact: true })).not.toBeVisible()
  expect(errors).toEqual([])
})

test('background failures stay visible and can be reopened and retried', async ({ page }) => {
  const errors = await setup(page, true)
  await start(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.getByRole('button', { name: '关闭图像导入', exact: true }).click()
  await page.evaluate(() => window.__diagram.reject(new Error('连接超时，请重试')))
  await expect(page.getByRole('alert')).toContainText('连接超时')
  await expect(page.getByRole('button', { name: '查看图像导入任务' })).toContainText('图像导入失败')
  await page.getByRole('button', { name: '查看图像导入任务' }).click()
  await expect(page.getByRole('dialog', { name: '智能导入图像' }).getByRole('alert')).toContainText(
    '连接超时',
  )
  await page.evaluate(() => {
    window.__diagram.delayed = false
  })
  await page.getByRole('button', { name: '识别并插入', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(2)
  expect(errors).toEqual([])
})
test('invalid model geometry is blocked before document mutation', async ({ page }) => {
  const errors = await setup(page)
  await page.evaluate(() => {
    window.__diagram.scene.circles[0].r = 10000
  })
  await start(page)
  await expect(page.getByRole('alert').filter({ hasText: '图像圆形坐标无效' })).toBeVisible()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(1)
  expect(errors).toEqual([])
})
