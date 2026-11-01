import { expect } from '@playwright/test'
import { test, open, insert } from './tableHarness.mjs'

const sample =
  '实验三十一 普朗克常量的测定\n\n一、实验目的\n了解光电效应的基本规律，测定普朗克常量。\n\n二、实验原理\n光照射到金属表面时，电子吸收光子能量。\n改变入射光频率，记录对应的截止电压。\n\n三、实验记录\n保持光源稳定，依次测量并记录数据。'
const viewportWidth = (page) => page.locator('.document-viewport').evaluate((el) => el.clientWidth)

for (const width of [1360, 1100]) {
  test(`workspace, live inspector and themes stay usable at ${width}px`, async ({ page }, info) => {
    await page.setViewportSize({ width, height: 900 })
    const errors = await open(page)
    const input = page.locator('textarea.paper-input')
    await input.fill(sample)
    await expect
      .poll(() => page.evaluate(() => window.__tables.document?.characterCount))
      .toBeGreaterThan(40)
    await page.getByLabel('页面缩放', { exact: true }).selectOption('65')
    await expect(page.locator('.document-breadcrumb')).toContainText('实验三十一')
    const originalWidth = await viewportWidth(page)
    await page.getByRole('button', { name: '收起侧栏' }).click()
    await expect.poll(() => viewportWidth(page)).toBeGreaterThan(originalWidth)
    await expect(page.getByRole('button', { name: '展开侧栏' })).toBeFocused()
    await expect(input).toHaveValue(sample)
    await page.getByRole('button', { name: '展开侧栏' }).click()
    await expect.poll(() => viewportWidth(page)).toBe(originalWidth)
    await expect(page.getByRole('button', { name: '收起侧栏' })).toBeFocused()

    await page.getByRole('button', { name: '排版设置', exact: true }).click()
    const inspector = page.getByRole('region', { name: '排版设置' })
    await expect(inspector).toBeVisible()
    const spacing = inspector.getByLabel('字距', { exact: true })
    await spacing.fill('85')
    await spacing.press('Tab')
    await expect
      .poll(() => page.evaluate(() => window.__tables.document?.settings.letterSpacing))
      .toBe(85)
    // The dock leaves the paper interactive and retains its content.
    await input.fill(sample + '\n重复测量三次。')
    await expect
      .poll(() => page.evaluate(() => window.__tables.payload?.text))
      .toContain('重复测量三次')
    await inspector.getByRole('button', { name: '关闭排版设置' }).click()
    await expect(page.getByRole('button', { name: '排版设置', exact: true })).toBeFocused()
    await page.screenshot({
      animations: 'disabled',
      path: `/tmp/bambuscribe-redesign-${info.project.name}-${width}-light.png`,
    })
    await page.getByRole('button', { name: '深色主题', exact: true }).click()
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
    await page.getByRole('button', { name: '排版设置', exact: true }).click()
    await page.screenshot({
      animations: 'disabled',
      path: `/tmp/bambuscribe-redesign-${info.project.name}-${width}-dark.png`,
    })
    await inspector.getByRole('button', { name: '关闭排版设置' }).click()
    await page.getByRole('tab', { name: '设备台', exact: true }).click()
    await expect(page.getByRole('button', { name: '新增实例', exact: true })).toBeVisible()
    await page.screenshot({
      animations: 'disabled',
      path: `/tmp/bambuscribe-redesign-${info.project.name}-${width}-device.png`,
    })
    await page.locator('.page-rail button').first().click()
    await expect(page.getByRole('tab', { name: '书写台' })).toHaveAttribute('aria-selected', 'true')
    await expect(input).toHaveValue(sample + '\n重复测量三次。')
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    expect(errors).toEqual([])
  })
}

test('sidebar project actions and header export use the existing document pipeline', async ({
  page,
}) => {
  const errors = await open(page)
  await page.getByRole('button', { name: '导出', exact: true }).click()
  await expect(page.getByRole('button', { name: '导出当前页' })).toBeDisabled()
  await expect(page.getByRole('dialog', { name: '导出文档' })).toContainText('当前页没有可书写内容')
  await page.getByRole('button', { name: '关闭导出文档' }).click()
  await page.locator('textarea.paper-input').fill('项目保存与导出测试')
  await insert(page, { rows: 2, cols: 3, width: 90 })
  await page.getByRole('button', { name: /另存为/ }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.tables.length)).toBe(1)
  await page.locator('textarea.paper-input').fill('修改后的文字')
  await page.getByRole('button', { name: /打开项目/ }).click()
  await expect(page.locator('textarea.paper-input')).toHaveValue('修改后的文字')
  await expect.poll(() => page.evaluate(() => window.__tables.saved.text)).toBe('修改后的文字')
  await page.getByRole('button', { name: '导出', exact: true }).click()
  await expect(page.getByRole('button', { name: '导出当前页' })).toBeEnabled()
  await page.getByRole('button', { name: '导出当前页' }).click()
  await expect.poll(() => page.evaluate(() => !!window.__tables.exported)).toBe(true)
  expect(errors).toEqual([])
})

test('sidebar page keys reveal the selected page and return from device mode', async ({ page }) => {
  const errors = await open(page)
  await page
    .locator('textarea.paper-input')
    .fill(Array.from({ length: 65 }, (_, i) => `第${i + 1}行实验记录`).join('\n'))
  await expect.poll(() => page.locator('.page-rail button').count()).toBeGreaterThan(3)
  await page.getByRole('tab', { name: '设备台' }).click()
  const first = page.locator('.page-rail button').first()
  await first.focus()
  await first.press('End')
  const last = page.locator('.page-rail button').last()
  await expect(last).toHaveAttribute('aria-current', 'page')
  await expect(last).toBeFocused()
  await expect(page.getByRole('tab', { name: '书写台' })).toHaveAttribute('aria-selected', 'true')
  await expect
    .poll(() => page.locator('.document-viewport').evaluate((el) => el.scrollTop))
    .toBeGreaterThan(500)
  expect(errors).toEqual([])
})

// Compare actual rendered surfaces to the installed Codex desktop defaults.
// This catches component overrides and old colored controls, not just token values.
test('Codex default palette reaches the shell and active controls in both themes', async ({
  page,
}) => {
  await open(page)
  for (const [theme, surface, sidebar, text, primary] of [
    ['light', 'rgb(255, 255, 255)', 'rgb(246, 246, 246)', 'rgb(26, 28, 31)', 'rgb(26, 28, 31)'],
    ['dark', 'rgb(24, 24, 24)', 'rgb(20, 20, 20)', 'rgb(223, 223, 223)', 'rgb(223, 223, 223)'],
  ]) {
    if (theme === 'dark') await page.getByRole('button', { name: '深色主题', exact: true }).click()
    // Wait for a painted frame: WebKit may expose stale inherited variables before painting.
    await page.screenshot({ animations: 'disabled' })
    for (const selector of [
      '.app-header',
      '.workspace-main',
      '.editor-toolbar',
      '.document-viewport',
    ])
      await expect(page.locator(selector)).toHaveCSS('background-color', surface)
    await expect(page.locator('.workspace-sidebar')).toHaveCSS('background-color', sidebar)
    await expect(page.locator('.document-breadcrumb strong')).toHaveCSS('color', text)
    await expect(page.locator('.upload-button')).toHaveCSS('background-color', primary)
    await expect(page.locator('.paper-page').first()).toHaveCSS(
      'background-color',
      'rgb(255, 255, 255)',
    )
    const selected = page.getByRole('button', { name: '选择', exact: true })
    await selected.hover()
    await expect(selected).toHaveCSS('color', text)
    await page.screenshot({ animations: 'disabled' })
    const channels = await selected.evaluate((el) =>
      getComputedStyle(el)
        .backgroundColor.match(/[\d.]+/g)
        .map(Number),
    )
    expect(channels.slice(0, 3)).toEqual(theme === 'light' ? [26, 28, 31] : [255, 255, 255])
    // Engines quantize transition alpha to 8 bits.
    expect(channels[3]).toBeCloseTo(theme === 'light' ? 0.078 : 0.1, 2)
    await page.keyboard.press('Tab')
    await page.getByRole('button', { name: /^打开项目/ }).focus()
    await expect(page.getByRole('button', { name: /^打开项目/ })).toHaveCSS(
      'outline-color',
      'rgb(51, 156, 255)',
    )
  }
})
