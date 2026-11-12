import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { open } from './tableHarness.mjs'
export const imagePath = new URL(
  '../../../internal/engine/testdata/ocr-formula-table.png',
  import.meta.url,
)
const hall = JSON.parse(
  readFileSync(
    new URL('../../../internal/calculation/testdata/hall.json', import.meta.url),
    'utf8',
  ),
)
export async function setup(page, options = {}) {
  const errors = await open(page)
  await page.evaluate(
    ({ plan, options }) => {
      const api = window.go.application.App
      window.__calc = { calls: [], cancelled: 0, plan }
      Object.assign(api, {
        AgentConfig: async () => ({
          baseURL: 'https://test.invalid/v1',
          model: 'test',
          protocol: 'responses',
          hasKey: true,
        }),
        SaveAgentConfig: async () => {},
        RecognizeCalculations: async (sources) => {
          window.__calc.calls.push(sources)
          if (options.defer)
            return new Promise((resolve, reject) => {
              window.__calc.finish = () => resolve(structuredClone(plan))
              window.__calc.fail = (message) => reject(new Error(message))
            })
          return structuredClone(plan)
        },
        CancelCalculationRecognition: async () => {
          window.__calc.cancelled++
        },
        PreviewCalculations: async (plan, settings) => {
          const r = await fetch('/__table-preview', {
            method: 'POST',
            body: JSON.stringify({ calculation: plan, settings }),
          })
          if (!r.ok) throw new Error(await r.text())
          const result = await r.json()
          window.__calc.preview = result
          return result
        },
      })
    },
    { plan: hall, options },
  )
  if (options.closed) return errors
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '数据处理', exact: true }).click()
  await expect(page.getByRole('dialog', { name: '数据处理', exact: true })).toBeVisible()
  return errors
}
export async function images(page) {
  await page.getByLabel('添加原始数据', { exact: true }).setInputFiles([imagePath.pathname])
  // File inputs can be populated by Playwright while disabled. Wait for the
  // first FileReader to finish before starting the next group, as a user does.
  await expect(page.getByRole('button', { name: '查看原始数据第1张', exact: true })).toBeVisible()
  await expect(page.getByLabel('添加参考样例', { exact: true })).toBeEnabled()
  await page
    .getByLabel('添加参考样例', { exact: true })
    .setInputFiles([imagePath.pathname, imagePath.pathname])
  await expect(page.getByRole('button', { name: '生成预览', exact: true })).toBeEnabled()
  await page.getByRole('button', { name: '查看参考样例第2张', exact: true }).click()
  await expect(page.getByRole('dialog', { name: '查看原图', exact: true })).toBeVisible()
  await page.getByRole('button', { name: '原始大小', exact: true }).click()
  await page.getByRole('button', { name: '关闭原图', exact: true }).click()
}
export async function generate(page) {
  await images(page)
  await page.getByRole('button', { name: '生成预览', exact: true }).click()
  await expect(page.getByRole('button', { name: '插入正文', exact: true })).toBeEnabled({
    timeout: 30000,
  })
}
export async function formulaClick(page) {
  const f = await page.evaluate(() => window.__tables.document.pages[0].formulas[0])
  const paper = page.locator('[data-paper="0"]')
  await paper.scrollIntoViewIfNeeded()
  const box = await paper.boundingBox()
  const width = await page.evaluate(() => window.__tables.document.settings.paperWidth)
  await paper.dblclick({
    position: {
      x: ((f.x + f.width / 2) * box.width) / width,
      y: ((f.y + f.height / 2) * box.width) / width,
    },
  })
}
