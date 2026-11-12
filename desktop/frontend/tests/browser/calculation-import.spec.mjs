import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { test, open } from './tableHarness.mjs'
const imagePath = new URL(
  '../../../internal/engine/testdata/ocr-formula-table.png',
  import.meta.url,
)
const hall = JSON.parse(
  readFileSync(
    new URL('../../../internal/calculation/testdata/hall.json', import.meta.url),
    'utf8',
  ),
)
async function setup(page, options = {}) {
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
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '数据处理', exact: true }).click()
  await expect(page.getByRole('dialog', { name: '数据处理', exact: true })).toBeVisible()
  return errors
}
async function images(page) {
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
async function generate(page) {
  await images(page)
  await page.getByRole('button', { name: '生成预览', exact: true }).click()
  await expect(page.getByRole('button', { name: '插入正文', exact: true })).toBeEnabled({
    timeout: 30000,
  })
}
async function formulaClick(page) {
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
test('multiple references become continuous editable body formulas; reopen and recalculate', async ({
  page,
}) => {
  const errors = await setup(page)
  await generate(page)
  expect(await page.evaluate(() => window.__calc.calls[0].references.length)).toBe(2)
  const before = await page.evaluate(() => window.__calc.preview)
  expect(before.values.b0).toBe('0.0290')
  expect(Object.keys(before.values)).toHaveLength(51)
  await page.getByLabel('分式字号比例', { exact: true }).fill('75')
  await expect(page.getByRole('button', { name: '插入正文', exact: true })).toBeEnabled()
  await expect
    .poll(() => page.evaluate(() => window.__calc.preview.document.settings.fractionScale))
    .toBe(75)
  await page.screenshot({ path: '/tmp/bambuscribe-calculation-import.png', fullPage: true })
  await page.getByRole('button', { name: '插入正文', exact: true }).click()
  await expect(page.getByRole('dialog', { name: '数据处理', exact: true })).not.toBeVisible()
  await expect
    .poll(() => page.evaluate(() => window.__tables.preferences.calculations?.length))
    .toBe(1)
  await expect
    .poll(() =>
      page.evaluate(() => window.__tables.document.pages.flatMap((p) => p.formulas ?? []).length),
    )
    .toBeGreaterThanOrEqual(51)
  const inserted = await page.getByLabel('书写内容').inputValue()
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue('')
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue(inserted)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0]?.formulas?.length ?? 0))
    .toBeGreaterThan(0)
  await formulaClick(page)
  await expect(page.getByRole('heading', { name: '编辑数据处理' })).toBeVisible()
  await page.getByLabel('霍尔灵敏度', { exact: true }).fill('121')
  await expect.poll(() => page.evaluate(() => window.__calc.preview.values.b0)).toBe('0.0580')
  await page.getByRole('button', { name: '更新正文', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue(/0\.0580/)
  // Use the app's exposed sidebar save action (the native menu key is platform-dependent).
  await page.getByRole('button', { name: '另存为', exact: true }).click()
  await expect
    .poll(() =>
      page.evaluate(() => window.__tables.saved?.calculations?.[0]?.plan.variables[0].value),
    )
    .toBe('121')
  await page.getByRole('button', { name: '打开项目', exact: true }).click()
  await expect
    .poll(() =>
      page.evaluate(() => window.__tables.document.pages.flatMap((p) => p.formulas ?? []).length),
    )
    .toBeGreaterThanOrEqual(51)
  await formulaClick(page)
  await expect(page.getByLabel('霍尔灵敏度', { exact: true })).toHaveValue('121')
  expect(errors).toEqual([])
})
test('uncertain readings and invalid arithmetic block insertion', async ({ page }) => {
  const errors = await setup(page)
  await page.evaluate(() => {
    window.__calc.plan.variables[0].uncertain = true
  })
  // Recognition closure and test plan share the same object.
  await images(page)
  await page.getByRole('button', { name: '生成预览', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('请核对霍尔灵敏度')
  await expect(page.getByRole('button', { name: '插入正文', exact: true })).toBeDisabled()
  await page.getByLabel('霍尔灵敏度', { exact: true }).fill('0')
  await expect(page.getByRole('alert')).toContainText('分母为0')
  await page.getByLabel('霍尔灵敏度', { exact: true }).fill('242')
  await expect(page.getByRole('button', { name: '插入正文', exact: true })).toBeEnabled({
    timeout: 30000,
  })
  expect(errors).toEqual([])
})
test('cancel discards late model response without inserting', async ({ page }) => {
  const errors = await setup(page, { defer: true })
  await images(page)
  await page.getByRole('button', { name: '生成预览', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__calc.calls.length)).toBe(1)
  await page.getByRole('button', { name: '取消', exact: true }).click()
  await page.evaluate(() => window.__calc.finish())
  await expect.poll(() => page.evaluate(() => window.__calc.cancelled)).toBe(1)
  await expect(page.getByLabel('书写内容')).toHaveValue('')
  expect(errors).toEqual([])
})
test('reference ordering and formula deletion preserve source integrity', async ({ page }) => {
  const errors = await setup(page)
  await page.getByLabel('添加原始数据', { exact: true }).setInputFiles(imagePath.pathname)
  await expect(page.getByRole('button', { name: '查看原始数据第1张', exact: true })).toBeVisible()
  await expect(page.getByLabel('添加参考样例', { exact: true })).toBeEnabled()
  const buffer = readFileSync(imagePath)
  await page.getByLabel('添加参考样例', { exact: true }).setInputFiles([
    { name: '第一页.png', mimeType: 'image/png', buffer },
    { name: '第二页.png', mimeType: 'image/png', buffer },
  ])
  await page.getByRole('button', { name: '前移参考样例第2张', exact: true }).click()
  await expect(
    page.locator('.calculation-images').nth(1).locator('.calculation-thumbnail').first(),
  ).toContainText('第二页.png')
  await page.getByRole('button', { name: '生成预览', exact: true }).click()
  await expect(page.getByRole('button', { name: '插入正文', exact: true })).toBeEnabled({
    timeout: 30000,
  })
  await page.getByRole('button', { name: '插入正文', exact: true }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages[0]?.formulas?.length ?? 0))
    .toBeGreaterThan(0)
  const before = await page.getByLabel('书写内容').inputValue()
  const end = before.indexOf('\\)') + 2
  await page.getByLabel('书写内容').evaluate((input, end) => {
    input.focus()
    input.setSelectionRange(end, end)
    input.dispatchEvent(new Event('select'))
  }, end)
  await page.getByLabel('书写内容').press('Backspace')
  const after = await page.getByLabel('书写内容').inputValue()
  expect(after.match(/\\\(/g)?.length).toBe(50)
  expect(after.match(/\\\)/g)?.length).toBe(50)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages.map((p) => p.text).join('')))
    .toBe(after)
  await formulaClick(page)
  await expect(page.getByRole('dialog', { name: '编辑公式', exact: true })).toBeVisible()
  await page.getByLabel('分母1', { exact: true }).first().fill('8')
  await expect(page.getByRole('button', { name: '保存公式', exact: true })).toBeEnabled()
  await page.getByRole('button', { name: '保存公式', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue(/\}\{8\}/)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue(after)

  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.getByLabel('书写内容')).toHaveValue(before)
  expect(errors).toEqual([])
})

test('recognition reports real stages and elapsed time; timeout preserves images for retry', async ({
  page,
}) => {
  const errors = await setup(page, { defer: true })
  await images(page)
  await page.clock.install()
  await page.getByRole('button', { name: '生成预览', exact: true }).click()
  const footer = page
    .getByRole('dialog', { name: '数据处理', exact: true })
    .locator('.calculation-footer')
  await expect(footer).toContainText('正在连接模型… · 已用时 0 秒')
  await page.evaluate(() => {
    window.__tables.menu['calculation:progress']({ requestId: 'old-request', stage: 'receiving' })
  })
  await expect(footer).toContainText('正在连接模型')
  await page.evaluate(() => {
    window.__tables.menu['calculation:progress']({
      requestId: window.__calc.calls[0].requestId,
      stage: 'processing',
    })
  })
  await page.clock.runFor(5000)
  await expect(footer).toContainText('模型正在识别图片… · 已用时 5 秒')
  await page.evaluate(() => {
    window.__tables.menu['calculation:progress']({
      requestId: window.__calc.calls[0].requestId,
      stage: 'receiving',
    })
  })
  await expect(footer).toContainText('正在接收计算步骤')
  await page.evaluate(() => window.__calc.fail('模型服务响应超时（HTTP 524），请稍后重试'))
  await expect(page.getByRole('alert')).toContainText('响应超时（HTTP 524）')
  await expect(page.getByRole('button', { name: '生成预览', exact: true })).toBeEnabled()
  await expect(page.locator('.calculation-thumbnail')).toHaveCount(3)
  await expect(page.getByLabel('书写内容')).toHaveValue('')
  await page.getByRole('button', { name: '生成预览', exact: true }).click()
  await expect(footer).toContainText('正在连接模型… · 已用时 0 秒')
  await page.evaluate(() => {
    window.__tables.menu['calculation:progress']({
      requestId: window.__calc.calls[0].requestId,
      stage: 'receiving',
    })
  })
  await expect(footer).toContainText('正在连接模型')
  await page.evaluate(() => window.__calc.finish())
  await expect(page.getByRole('button', { name: '插入正文', exact: true })).toBeEnabled({
    timeout: 30000,
  })
  expect(errors).toEqual([])
})

test('reference constants preview immediately and keep their chosen precision when edited and saved', async ({
  page,
}) => {
  const errors = await setup(page)
  const plan = JSON.parse(
    readFileSync(
      new URL('../../../internal/calculation/testdata/sound-reference.json', import.meta.url),
      'utf8',
    ),
  )
  await page.evaluate((plan) => {
    window.go.application.App.RecognizeCalculations = async () => structuredClone(plan)
  }, plan)
  await generate(page)
  for (const v of plan.variables)
    await expect(page.getByLabel(v.label, { exact: true })).toHaveValue(v.value)
  await expect(page.getByRole('alert')).not.toBeVisible()
  await expect(page.getByRole('button', { name: '确认读数', exact: true })).toHaveCount(0)
  await page.getByLabel('空气摩尔质量 M', { exact: true }).fill('')
  await expect(page.getByRole('alert')).toContainText('请核对空气摩尔质量')
  await expect(page.getByRole('button', { name: '插入正文', exact: true })).toBeDisabled()
  await page.getByLabel('空气摩尔质量 M', { exact: true }).fill('0.029')
  await expect(page.getByRole('button', { name: '插入正文', exact: true })).toBeEnabled()
  await page.screenshot({ path: '/tmp/bambuscribe-reference-constants.png', fullPage: true })
  await page.getByRole('button', { name: '插入正文', exact: true }).click()
  await page.getByRole('button', { name: '另存为', exact: true }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved?.calculations?.[0]?.plan.variables))
    .toEqual(plan.variables)
  expect(errors).toEqual([])
})
