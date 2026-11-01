import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

test('icon rail stays narrow and exposes one hover or keyboard hint at a time', async ({
  page,
}, info) => {
  const errors = await open(page)
  const rail = page.getByRole('complementary', { name: '工作区导航' })
  await expect(rail).toHaveCSS('width', '60px')
  await expect(rail).not.toContainText(/BambuScribe|打开项目|另存为|工作区|当前项目|本地工作区/)
  const openButton = rail.getByRole('button', { name: '打开项目', exact: true })
  await openButton.hover()
  await expect(page.getByRole('tooltip')).toHaveText('打开项目 · ⌘O')
  const triggerBox = await openButton.boundingBox()
  const hintBox = await page.getByRole('tooltip').boundingBox()
  expect(hintBox.x).toBeGreaterThan(triggerBox.x + triggerBox.width)
  await page.getByRole('tooltip').hover()
  await expect(page.getByRole('tooltip')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('tooltip')).toHaveCount(0)
  await page.keyboard.press('Tab')
  await openButton.focus()
  await expect(page.getByRole('tooltip')).toHaveText('打开项目 · ⌘O')
  await openButton.press('Tab')
  // WebKit follows macOS's setting that skips buttons during plain Tab navigation.
  await rail.getByRole('button', { name: '另存为', exact: true }).focus()
  await expect(rail.getByRole('button', { name: '另存为', exact: true })).toBeFocused()
  await expect(page.getByRole('tooltip')).toHaveText('另存为 · ⌘⇧S')
  await rail.getByRole('tab', { name: '设备台' }).hover()
  await expect(page.getByRole('tooltip')).toHaveCount(1)
  await expect(page.getByRole('tooltip')).toHaveText('设备台')
  await page.screenshot({
    path: `/tmp/bambuscribe-icon-rail-${info.project.name}.png`,
    animations: 'disabled',
  })
  await rail.getByRole('tab', { name: '设备台' }).click()
  await expect(page.getByRole('tooltip')).toHaveCount(0)
  expect(errors).toEqual([])
})

test('page icon hints survive a scrollable rail without clipping or lost page navigation', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1100, height: 600 })
  await open(page)
  await page
    .locator('textarea.paper-input')
    .fill(Array.from({ length: 180 }, (_, i) => `第${i + 1}行实验记录`).join('\n'))
  const buttons = page.locator('.page-rail button')
  await expect.poll(() => buttons.count()).toBeGreaterThan(10)
  await buttons.first().focus()
  await buttons.first().press('End')
  await expect(buttons.last()).toBeFocused()
  await expect(buttons.last()).toHaveAttribute('aria-current', 'page')
  await buttons.last().hover()
  await expect(page.getByRole('tooltip')).toContainText('实验记录')
  const hint = await page.getByRole('tooltip').boundingBox()
  expect(hint.y).toBeGreaterThanOrEqual(8)
  expect(hint.y + hint.height).toBeLessThanOrEqual(592)
  await page.locator('.page-rail').evaluate((el) => {
    el.scrollTop = 0
  })
  await expect(page.getByRole('tooltip')).toHaveCount(0)
})

test('device and layout panels expose controls and status without instruction paragraphs', async ({
  page,
}, info) => {
  await open(page)
  await page.getByRole('tab', { name: '设备台' }).click()
  const device = page.locator('.device-page')
  await expect(device.locator('.preparation-panel, p.hint')).toHaveCount(0)
  await expect(device).not.toContainText(/开始前|卸笔|换纸后|X 正值向右|BAMBU LAB/)
  await expect(device.getByLabel('抬笔位置')).toBeVisible()
  await expect(device.getByLabel('落笔位置')).toBeVisible()
  await expect(device.getByLabel('原点模式')).toHaveAttribute('title', /请先定位笔尖/)
  await page.screenshot({
    path: `/tmp/bambuscribe-icon-device-${info.project.name}-light.png`,
    animations: 'disabled',
  })
  await page.getByRole('button', { name: '深色主题' }).click()
  await page.screenshot({
    path: `/tmp/bambuscribe-icon-device-${info.project.name}-dark.png`,
    animations: 'disabled',
  })
  await page.getByRole('tab', { name: '书写台' }).click()
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  const panel = page.getByRole('region', { name: '排版设置' })
  await expect(panel.locator('p.hint')).toHaveCount(0)
  await expect(panel).not.toContainText(/内置手写字库|按基准行高计算|最多.*行/)
  await expect(panel.getByLabel('字体大小')).toBeVisible()
  await expect(panel.locator('.number-field').filter({ hasText: '字体大小' })).toHaveAttribute(
    'title',
    /当前字框.*mm/,
  )
})
