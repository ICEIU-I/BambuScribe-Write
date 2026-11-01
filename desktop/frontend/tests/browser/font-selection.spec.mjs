import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { test, open } from './tableHarness.mjs'

test('现有霍尔教案保持85%行距切换DC体，不误报行间重叠', async ({ page }, info) => {
  const errors = await open(page)
  const text = readFileSync(resolve('../internal/engine/testdata/dc-compact-lesson.txt'), 'utf8')
  await page.evaluate((text) => {
    window.__tables.saved = {
      ...window.__tables.preferences,
      text,
      layout: {
        ...window.__tables.preferences.layout,
        paperWidth: 210,
        paperHeight: 297,
        marginTop: 25.4,
        marginBottom: 25.4,
        marginLeft: 17,
        marginRight: 17,
        lineCount: 30,
        lineHeight: (297 - 2 * 25.4) / 30,
        fontSize: (297 - 2 * 25.4) / 30,
        letterSpacing: 76.5,
        lineSpacing: 85,
        pageLayouts: Array.from({ length: 5 }, () => ({ lineCount: 30 })),
      },
    }
    window.__tables.menu['menu:action']('open')
  }, text)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.settings.lineSpacing))
    .toBe(85)
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  await page.getByLabel('字体', { exact: true }).selectOption('dc')
  await expect.poll(() => page.evaluate(() => window.__tables.document?.fontName)).toBe('DC体')
  const doc = await page.evaluate(() => window.__tables.document)
  expect(doc.settings.lineSpacing).toBe(85)
  expect(doc.settings.letterSpacing).toBe(76.5)
  expect(doc.settings.fontSize).toBeCloseTo((297 - 2 * 25.4) / 30, 8)
  expect(doc.missing).toEqual([])
  await expect(page.getByText(/行距过小/)).toHaveCount(0)
  await page.screenshot({ path: `/tmp/bambuscribe-dc-compact-${info.project.name}.png` })
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect(errors).toEqual([])
})

test('字体切换同步正文与公式对象，保存重开仍使用所选字体', async ({ page }, info) => {
  const errors = await open(page)
  await page.evaluate(() => {
    window.__tables.saved = {
      ...window.__tables.preferences,
      text: '实验霍尔效应\n普朗克常量 e=1.602×10⁻¹⁹\n懈幛悻 ①②⑩ ½',
      tables: [
        {
          id: 'formula',
          page: 0,
          kind: 'textbox',
          x: 10,
          y: 90,
          columnWidths: [140],
          rowHeights: [20],
          cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: 'U̅_{H}=R_{H}·I_{S}B/d' }],
          fontSize: 5,
          padding: 1,
          align: 'left',
          verticalAlign: 'middle',
          style: 'none',
        },
      ],
    }
    window.__tables.menu['menu:action']('open')
  })
  await expect.poll(() => page.evaluate(() => window.__tables.document?.fontName)).toBe('教案专用')
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.tables?.length))
    .toBe(1)
  const original = await page.evaluate(() => window.__tables.document.pages[0])
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  const font = page.getByLabel('字体', { exact: true })
  await expect(font).toHaveValue('handwriting')
  await font.selectOption('dc')
  await expect.poll(() => page.evaluate(() => window.__tables.document?.fontName)).toBe('DC体')
  const dc = await page.evaluate(() => window.__tables.document)
  expect(dc.missing).toEqual([])
  expect(dc.pages[0].issues ?? []).toEqual([])
  expect(dc.pages[0].strokes).not.toEqual(original.strokes)
  expect(dc.pages[0].tables[0].strokes).not.toEqual(original.tables[0].strokes)
  await page.screenshot({ path: `/tmp/bambuscribe-dc-${info.project.name}.png` })
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved.layout.fontId)).toBe('dc')
  await font.selectOption('handwriting')
  await expect.poll(() => page.evaluate(() => window.__tables.document?.fontName)).toBe('教案专用')
  // Named projects now autosave subsequent font changes before reopening.
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved.layout.fontId))
    .toBe('handwriting')
  const reopened = page.waitForResponse('**/__table-preview')
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await reopened
  await expect(page.getByRole('button', { name: '导出', exact: true })).toBeEnabled()
  await expect.poll(() => page.evaluate(() => window.__tables.document?.fontName)).toBe('教案专用')
  await expect(font).toHaveValue('handwriting')
  expect(await page.evaluate(() => window.__tables.document.pages[0].strokes)).toEqual(
    original.strokes,
  )
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => window.__tables.exported?.gcode)).toContain('G1')
  expect(errors).toEqual([])
})
