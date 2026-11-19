import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

for (const [opening, closing] of [
  ['\\[', '\\]'],
  ['\\(', '\\)'],
]) {
  test(`${opening} 公式前的空格和 Tab 移动笔画，支持撤销、编辑、保存和导出`, async ({ page }) => {
    const errors = await open(page)
    const input = page.getByRole('textbox', { name: '书写内容', exact: true })
    const source = String.raw`x=\frac{1}{2}`
    const formula = opening + source + closing
    const waitFor = async (text, x) => {
      await expect
        .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
        .toBe(text)
      await expect
        .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.formulas?.[0]?.x))
        .toBe(x)
    }
    await input.fill(formula)
    await waitFor(formula, 0)
    await page.keyboard.press('Home')
    await page.keyboard.press('Space')
    await waitFor(' ' + formula, 3)
    expect(await input.evaluate((el) => el.selectionStart)).toBe(1)
    await page.keyboard.press('Space')
    await waitFor('  ' + formula, 6)
    expect(await input.evaluate((el) => el.selectionStart)).toBe(2)
    await page.keyboard.press('Tab')
    await waitFor('  \t' + formula, 30)
    await expect(input).toBeFocused()
    expect(await input.evaluate((el) => el.selectionStart)).toBe(3)
    const cells = await page.evaluate(() => window.__tables.document.pages[0].cells)
    expect(cells.slice(0, 3).map((c) => [c.start, c.end, c.x, c.width])).toEqual([
      [0, 1, 0, 3],
      [1, 2, 3, 3],
      [2, 3, 6, 24],
    ])
    await page.keyboard.press('Backspace')
    await waitFor('  ' + formula, 6)
    await page.getByRole('button', { name: '撤销', exact: true }).click()
    await waitFor('  \t' + formula, 30)
    await page.getByRole('button', { name: '重做', exact: true }).click()
    await waitFor('  ' + formula, 6)
    await page.getByRole('button', { name: '撤销', exact: true }).click()
    await waitFor('  \t' + formula, 30)

    const f = await page.evaluate(() => window.__tables.document.pages[0].formulas[0])
    const paper = page.locator('[data-paper="0"]')
    await paper.scrollIntoViewIfNeeded()
    const bounds = await paper.boundingBox()
    const width = await page.evaluate(() => window.__tables.document.settings.paperWidth)
    // Click between the two spaces: individual whitespace remains reachable.
    await paper.click({
      position: { x: (3 * bounds.width) / width, y: (3 * bounds.width) / width },
    })
    expect(await input.evaluate((el) => el.selectionStart)).toBe(1)
    await paper.dblclick({
      position: {
        x: ((f.x + f.width / 2) * bounds.width) / width,
        y: ((f.y + f.height / 2) * bounds.width) / width,
      },
    })
    const dialog = page.getByRole('dialog', { name: '编辑公式', exact: true })
    await expect(dialog).toBeVisible()
    await dialog.getByRole('textbox').first().fill('q=')
    await expect(dialog.getByRole('button', { name: '保存公式', exact: true })).toBeEnabled()
    await dialog.getByRole('button', { name: '保存公式', exact: true }).click()
    const edited = '  \t' + opening + source.replace('x=', 'q=') + closing
    await waitFor(edited, 30)
    await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
    await expect.poll(() => page.evaluate(() => window.__tables.saved?.text)).toBe(edited)
    await page.evaluate(() => window.__tables.menu['menu:action']('open'))
    await expect(input).toHaveValue(edited)
    await waitFor(edited, 30)
    await page.getByRole('button', { name: '导出', exact: true }).click()
    await page.getByRole('button', { name: /^导出当前页/ }).click()
    await expect
      .poll(() => page.evaluate(() => window.__tables.exported?.gcode ?? ''))
      .toContain('G1')
    const exported = await page.evaluate(() => window.__tables.exported.document.pages[0])
    expect(exported.formulas[0].x).toBe(30)
    expect(exported.issues ?? []).toEqual([])
    await page.screenshot({ path: test.info().outputPath('formula-spacing.png') })
    expect(errors).toEqual([])
  })
}
