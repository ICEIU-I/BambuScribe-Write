import { expect } from '@playwright/test'
import { test, open } from './tableHarness.mjs'

test('正文 Tab 插入四格并保持焦点，替换选区和撤销重做保持光标', async ({ page }) => {
  const errors = await open(page)
  const input = page.getByRole('textbox', { name: '书写内容', exact: true })
  await input.fill('你四、实验')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toBe('你四、实验')
  await input.evaluate((el) => {
    el.setSelectionRange(1, 1)
    el.dispatchEvent(new Event('select'))
  })
  await page.keyboard.press('Tab')
  await expect(input).toBeFocused()
  await expect(input).toHaveValue('你\t四、实验')
  expect(await input.evaluate((el) => el.selectionStart)).toBe(2)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toBe('你\t四、实验')
  const cells = await page.evaluate(() => window.__tables.document.pages[0].cells)
  expect(cells[2].x - cells[1].x).toBeCloseTo(24, 8)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(input).toHaveValue('你四、实验')
  expect(await input.evaluate((el) => el.selectionStart)).toBe(1)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(input).toHaveValue('你\t四、实验')
  expect(await input.evaluate((el) => el.selectionStart)).toBe(2)
  await input.evaluate((el) => {
    el.setSelectionRange(2, 4)
    el.dispatchEvent(new Event('select'))
  })
  await page.keyboard.press('Tab')
  await expect(input).toHaveValue('你\t\t实验')
  await expect(input).toBeFocused()
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(input).toHaveValue('你\t四、实验')
  expect(await input.evaluate((el) => [el.selectionStart, el.selectionEnd])).toEqual([2, 4])
  await input.focus()
  await page.keyboard.press('Shift+Tab')
  await expect(input).not.toBeFocused()
  await expect(input).toHaveValue('你\t四、实验')
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.text)).toBe('你\t四、实验')
  const reopened = page.waitForResponse('**/__table-preview')
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await reopened
  await expect(input).toHaveValue('你\t四、实验')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toBe('你\t四、实验')
  expect(errors).toEqual([])
})
