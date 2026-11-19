import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

test('two-row fractions keep readable ink and editable lower halves through save and export', async ({
  page,
}) => {
  const errors = await open(page)
  const text = String.raw`数据处理
\(η=\frac{(ρ-ρ_{0})gd^{2}}{18v(1+\frac{2.4d}{D})}\)　\(x=1\)
结束`
  await page.evaluate((text) => {
    window.__tables.saved = {
      ...window.__tables.preferences,
      text,
      layout: {
        ...window.__tables.preferences.layout,
        paperWidth: 210,
        paperHeight: 297,
        marginTop: 24,
        marginBottom: 24,
        marginLeft: 17,
        marginRight: 17,
        fontSize: 8.3,
        lineCount: 30,
        lineSpacing: 85,
        fractionScale: 100,
      },
    }
    window.__tables.menu['menu:action']('open')
  }, text)
  const input = page.getByLabel('书写内容')
  await expect(input).toHaveValue(text)
  await expect.poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text)).toBe(text)
  const doc = await page.evaluate(() => window.__tables.document)
  const p = doc.pages[0],
    f = p.formulas[0]
  expect(f.height).toBeGreaterThan(1.7 * p.lineHeight)
  expect(p.cells.find((c) => c.start === f.start).height).toBeCloseTo(2 * p.lineHeight)
  expect(p.formulas[1].x).toBeGreaterThan(f.x + f.width + 4)
  const paper = page.locator('[data-paper="0"]')
  await paper.scrollIntoViewIfNeeded()
  const bounds = await paper.boundingBox()
  await page.screenshot({ path: test.info().outputPath('two-row-fraction.png') })
  // The denominator is below the first rule; hit testing must still edit the whole formula.
  await paper.dblclick({
    position: {
      x: ((f.x + f.width / 2) * bounds.width) / 210,
      y: ((f.y + f.height * 0.85) * bounds.width) / 210,
    },
  })
  const dialog = page.getByRole('dialog', { name: '编辑公式', exact: true })
  await expect(dialog).toBeVisible()
  await dialog.getByRole('textbox').first().fill('q=')
  await expect(dialog.getByRole('button', { name: '保存公式', exact: true })).toBeEnabled()
  await dialog.getByRole('button', { name: '保存公式', exact: true }).click()
  const edited = text.replace('η=', 'q=')
  await expect(input).toHaveValue(edited)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toBe(edited)
  await page.getByRole('button', { name: '另存为', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.text)).toBe(edited)
  await page.getByRole('button', { name: '打开项目', exact: true }).click()
  await expect(input).toHaveValue(edited)
  await page.getByRole('button', { name: '导出', exact: true }).click()
  await expect(page.getByRole('button', { name: /^导出当前页/ })).toBeEnabled()
  await page.getByRole('button', { name: /^导出当前页/ }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.exported?.gcode ?? ''))
    .toContain('G1')
  const exported = await page.evaluate(() => window.__tables.exported.document.pages[0])
  expect(exported.formulas[0].height).toBeGreaterThan(1.7 * p.lineHeight)
  expect(exported.issues ?? []).toEqual([])
  expect(errors).toEqual([])
})
