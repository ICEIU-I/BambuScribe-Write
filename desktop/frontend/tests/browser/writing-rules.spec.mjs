import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

test('icon opens categorized rules with navigation, copying and keyboard dismissal', async ({
  page,
}) => {
  const errors = await open(page)
  const button = page.getByRole('button', { name: '输入规则', exact: true })
  await expect(button).toHaveText('')
  await expect(button.locator('svg')).toBeVisible()
  await expect(page.locator('.sidebar-footer .input-rules-button')).toBeVisible()
  const anchor = await button.boundingBox()
  expect(anchor.x).toBeLessThan(100)
  expect(anchor.y).toBeGreaterThan(page.viewportSize().height - 200)
  await page.getByLabel('书写内容').fill('当前正文保持不变')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toBe('当前正文保持不变')
  const documentBefore = await page.evaluate(() => window.__tables.document.id)
  await page.evaluate(() => {
    window.runtime.ClipboardSetText = async (text) => {
      window.__copiedRule = text
      return true
    }
  })
  await button.click()
  const dialog = page.getByRole('dialog', { name: '输入规则', exact: true })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('tab')).toHaveCount(7)
  await expect(dialog.getByRole('tab', { name: '错误与兼容' })).toHaveCount(0)
  await expect(dialog.getByRole('tab', { name: '基本规则', exact: true })).toHaveAttribute(
    'aria-selected',
    'true',
  )
  await expect(dialog.getByRole('button', { name: '上一页', exact: true })).toBeDisabled()
  await dialog.getByRole('button', { name: '下一页', exact: true }).click()
  await expect(dialog.getByRole('tabpanel')).toContainText('x_{i}^{2}')
  await dialog.getByRole('tab', { name: '根号', exact: true }).click()
  await expect(dialog.getByRole('tabpanel')).toContainText('横线随内容的实际宽度伸缩')
  await expect(dialog.getByRole('tabpanel')).not.toContainText('同一基项最多')
  await expect(dialog.getByRole('img', { name: '示例实际手写预览' })).toBeVisible()
  await expect.poll(() => dialog.locator('.rule-preview path').count()).toBeGreaterThan(0)
  const firstPreview = await dialog.locator('.rule-preview svg').innerHTML()
  await expect
    .poll(() => page.evaluate(() => window.__rulePreview?.pages[0]?.text))
    .toBe(String.raw`距离d=\sqrt{x^{2}+y^{2}}。`)
  const enginePaths = await page.evaluate(() =>
    window.__rulePreview.pages[0].strokes.map((s) =>
      s.map((p, i) => `${i ? 'L' : 'M'}${p.x},${p.y}`).join(' '),
    ),
  )
  expect(
    await dialog
      .locator('.rule-preview path')
      .evaluateAll((paths) => paths.map((p) => p.getAttribute('d'))),
  ).toEqual(enginePaths)
  await page.screenshot({
    path: `/tmp/bambuscribe-writing-rules-${test.info().project.name}.png`,
    fullPage: true,
  })
  await dialog
    .getByRole('button', { name: /^复制示例/ })
    .first()
    .click()
  await expect
    .poll(() => page.evaluate(() => window.__copiedRule))
    .toBe(String.raw`距离d=\sqrt{x^{2}+y^{2}}。`)
  await expect(dialog).toContainText('已复制')
  await dialog.getByRole('button', { name: /换个示例/ }).click()
  await expect(dialog.locator('pre')).toContainText('几何平均值')
  await expect
    .poll(() => page.evaluate(() => window.__rulePreview?.pages[0]?.text))
    .toContain('几何平均值')
  await expect(dialog.getByRole('img', { name: '示例实际手写预览' })).toBeVisible()
  expect(await dialog.locator('.rule-preview svg').innerHTML()).not.toBe(firstPreview)
  expect(await page.evaluate(() => window.__tables.document.id)).toBe(documentBefore)
  await dialog.getByRole('tab', { name: '根号', exact: true }).focus()
  await page.keyboard.press('End')
  await expect(dialog.getByRole('tab', { name: '符号与空格' })).toBeFocused()
  await expect(dialog.getByRole('button', { name: '下一页', exact: true })).toBeDisabled()
  await page.screenshot({
    path: `/tmp/bambuscribe-writing-rules-focus-${test.info().project.name}.png`,
    fullPage: true,
  })
  await page.keyboard.press('Escape')
  await expect(dialog).not.toBeVisible()
  await expect(button).toBeFocused()
  expect(errors).toEqual([])
})

test('rules remain usable on a narrow dark window', async ({ page }) => {
  await open(page)
  await page.setViewportSize({ width: 760, height: 680 })
  await page.evaluate(() => (document.documentElement.dataset.theme = 'dark'))
  await page.getByRole('button', { name: '输入规则', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '输入规则', exact: true })
  await dialog.getByRole('tab', { name: '分式', exact: true }).click()
  await expect.poll(() => dialog.locator('.rule-preview path').count()).toBeGreaterThan(0)
  await expect(dialog.getByRole('img', { name: '示例实际手写预览' })).toBeVisible()
  await expect.poll(() => dialog.locator('.rule-preview path').count()).toBeGreaterThan(0)
  await page.screenshot({
    path: `/tmp/bambuscribe-writing-rules-dark-${test.info().project.name}.png`,
    fullPage: true,
  })
  const bounds = await dialog.boundingBox()
  expect(bounds.x).toBeGreaterThanOrEqual(0)
  expect(bounds.x + bounds.width).toBeLessThanOrEqual(760)
  expect(bounds.y + bounds.height).toBeLessThanOrEqual(680)
  await dialog.getByRole('button', { name: '关闭输入规则', exact: true }).click()
  await expect(dialog).not.toBeVisible()
})

test('new inline and display math survives preview, editing, saving and export', async ({
  page,
}) => {
  const errors = await open(page)
  const text = String.raw`电阻R=\frac{U_{1}+U_{2}}{I_{g}}。
距离d=\sqrt{x_{i}^{2}+y^{2}}，平均值\bar{U}。
\[z=\sqrt[3]{\frac{a+b}{c}}\]`
  await page.getByLabel('书写内容').fill(text)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages.map((p) => p.text).join('')))
    .toBe(text)
  const doc = await page.evaluate(() => window.__tables.document)
  expect(doc.missing).toEqual([])
  expect(doc.pages.flatMap((p) => p.issues ?? [])).toEqual([])
  expect(doc.pages[0].formulas).toHaveLength(1)
  const f = doc.pages[0].formulas[0]
  const paper = page.locator('[data-paper="0"]')
  await paper.scrollIntoViewIfNeeded()
  const bounds = await paper.boundingBox()
  await paper.dblclick({
    position: {
      x: ((f.x + f.width / 2) * bounds.width) / doc.settings.paperWidth,
      y: ((f.y + f.height / 2) * bounds.width) / doc.settings.paperWidth,
    },
  })
  const dialog = page.getByRole('dialog', { name: '编辑公式', exact: true })
  await expect(dialog).toBeVisible()
  await dialog.getByRole('textbox', { name: '根指数1', exact: true }).fill('4')
  await expect(dialog.getByRole('button', { name: '保存公式', exact: true })).toBeEnabled()
  await dialog.getByRole('button', { name: '保存公式', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue(text.replace('\sqrt[3]', '\sqrt[4]'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages.map((p) => p.text).join('')))
    .toBe(text.replace('\sqrt[3]', '\sqrt[4]'))
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved?.text))
    .toBe(text.replace('\sqrt[3]', '\sqrt[4]'))
  await page.getByRole('button', { name: '导出', exact: true }).click()
  await expect(page.getByRole('button', { name: /^导出当前页/ })).toBeEnabled()
  await page.getByRole('button', { name: /^导出当前页/ }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode ?? ''))
    .toContain('G1')
  await page.screenshot({ path: '/tmp/bambuscribe-math-input.png', fullPage: true })
  expect(errors).toEqual([])
})
