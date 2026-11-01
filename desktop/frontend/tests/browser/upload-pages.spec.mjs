import { test, expect } from '@playwright/test'

async function open(page, options = {}) {
  await page.addInitScript((options) => {
    const layout = {
      templateName: '新建笔记',
      lineMode: 'fixed',
      lineCount: 16,
      paperWidth: 163,
      paperHeight: 240,
      marginTop: 0,
      marginRight: 0,
      marginBottom: 0,
      marginLeft: 0,
      fontSize: 6,
      letterGap: 0,
      lineHeight: 15,
      paragraphGap: 0,
      indent: 0,
      align: 'left',
      ...options.layout,
    }
    const writing = {
      offsetX: 0,
      offsetY: 0,
      penUp: 7,
      penDown: 3.4,
      originMode: 'auto',
      sound: true,
      ...options.writing,
    }
    const handlers = {}
    let state = {
      connected: true,
      phase: 'idle',
      message: '',
      progress: 0,
      jobId: '',
      documentId: '',
      page: 0,
      deviceState: 'IDLE',
      totalPages: 0,
      pageOrdinal: 0,
      uploadedPages: 0,
    }
    const buildDocument = (text) => ({
      id: `doc-${text}`,
      settings: layout,
      missing: [],
      characterCount: text.length,
      fontName: '测试字库',
      pages: [1, 2, 3, 4, 5].map((number) => ({
        number,
        lineHeight: 15,
        lineCount: 16,
        strokes:
          options.emptyFirst && number === 1
            ? []
            : options.pageStrokes?.[number - 1] || [
                [
                  { x: 20, y: 20 },
                  { x: 30, y: 30 },
                ],
              ],
        textStrokes: [],
        images: [],
        missing: options.missing && number === 5 ? [{ char: '🙂', x: 20, y: 20, size: 6 }] : [],
        text: number === 1 ? text : '',
        cells: [],
        carets: [],
      })),
    })
    window.__uploads = []
    window.__previews = []
    window.__savedPreferences = null
    window.__cancels = 0
    window.__emitPrinter = (next) => {
      state = { ...state, ...next }
      ;(handlers['printer:state'] || []).forEach((h) => h(state))
    }
    window.__menu = (action) => (handlers['menu:action'] || []).forEach((h) => h(action))
    window.runtime = {
      EventsOn: (name, callback) => {
        ;(handlers[name] ||= []).push(callback)
        return () => {}
      },
      WindowSetTitle: () => {},
    }
    window.go = {
      application: {
        App: {
          Bootstrap: async () => ({
            preferences: {
              theme: 'light',
              language: options.language || 'zh',
              text: options.text || '实验三十一 普朗克常量测定',
              layout,
              writing,
              images: [],
            },
            printer: { ip: '', serial: '', hasAccessCode: false },
            state,
            warning: '',
          }),
          PreviewWithImages: async (text, settings) => {
            window.__previews.push(settings)
            return { ...buildDocument(text), settings }
          },
          SavePreferences: async (preferences) => {
            window.__savedPreferences = preferences
          },
          SetLanguage: async () => {},
          SetMenuState: async () => {},
          PrinterState: async () => state,
          UploadDevicePages: async (deviceId, id, prefix, indices, settings) => {
            window.__uploads.push({ id, prefix, indices, settings })
            const totalPages = indices.length || (options.emptyFirst ? 4 : 5)
            const lastPage = indices.length ? indices.at(-1) + 1 : 5
            window.__emitPrinter({
              phase: 'uploaded',
              jobId: '',
              documentId: id,
              page: lastPage,
              totalPages,
              pageOrdinal: totalPages,
              uploadedPages: totalPages,
              progress: 100,
              message: `全部 ${totalPages} 页已上传，请在 A1 屏幕选择文件开始书写`,
            })
            return 'page-1'
          },
          CancelDeviceUpload: async () => {
            window.__cancels++
            window.__emitPrinter({ phase: 'idle', jobId: '', message: '上传已取消' })
          },
        },
      },
    }
  }, options)
  await page.goto('/')
  await expect(page.locator('.editor-toolbar')).toBeVisible()
  await expect(page.locator('.paper-input')).toHaveValue(
    options.text || '实验三十一 普朗克常量测定',
  )
}

async function uploadAll(page) {
  await page.locator('.upload-button').click()
  await page.getByRole('button', { name: '上传全部', exact: true }).click()
  await page.getByRole('button', { name: '完成', exact: true }).click()
}

test('old draft corrections are discarded and uploads use only base writing settings', async ({
  page,
}) => {
  const writing = {
    offsetX: 0,
    offsetY: 0,
    penUp: 7,
    penDown: 3.4,
    originMode: 'auto',
    sound: true,
  }
  await open(page, { writing: { ...writing, bottomLineDrop: 2, bottomLineDrops: [2, 3.4, 3.4] } })
  await page.getByRole('tab', { name: '设备台', exact: true }).click()
  await expect(page.getByRole('spinbutton', { name: /降笔/ })).toHaveCount(0)
  await uploadAll(page)
  expect(await page.evaluate(() => window.__uploads.at(-1).settings)).toEqual(writing)
})

test('one request uploads every page regardless of the viewed page and exposes no writing controls', async ({
  page,
}) => {
  await open(page)
  await page.locator('.page-rail button').nth(2).click()
  await uploadAll(page)
  await expect(page.locator('.task-page')).toHaveText('已上传 5 页')
  expect(await page.evaluate(() => window.__uploads.length)).toBe(1)
  expect(await page.evaluate(() => window.__uploads[0].indices)).toEqual([])
  await expect(page.locator('.status-message')).toContainText('A1 屏幕')
  await expect(
    page.getByRole('button', { name: /开始书写|开始下一页|暂停|继续|停止|取消任务/ }),
  ).toHaveCount(0)
  await page.evaluate(() => window.__emitPrinter({ phase: 'external', deviceState: 'RUNNING' }))
  await expect(page.locator('.job-summary .badge')).toHaveText('设备忙碌')
  await expect(page.locator('.job-actions')).toHaveCount(0)
})

test('a missing glyph blocks all-page upload while allowing a valid specific page from the dialog', async ({
  page,
}) => {
  await open(page, { missing: true })
  await page.evaluate(() => window.__menu('upload'))
  const dialog = page.getByRole('dialog', { name: '上传到 A1' })
  await expect(dialog).toBeVisible()
  const upload = dialog.getByRole('button', { name: '上传全部', exact: true })
  await expect(upload).toBeDisabled()
  await expect(upload).toHaveAttribute('title', /第 5 页缺字/)
  expect(await page.evaluate(() => window.__uploads.length)).toBe(0)
  await dialog.getByLabel('特定页', { exact: true }).selectOption('2')
  await dialog.getByRole('button', { name: '上传特定页', exact: true }).click()
  await expect(dialog.locator('.upload-suffix')).toHaveText('-page3')
  await dialog.getByRole('button', { name: '完成', exact: true }).click()
  expect(await page.evaluate(() => window.__uploads[0].indices)).toEqual([2])
  await expect(page.locator('.task-page')).toHaveText('已上传 1 页')
})

test('empty pages are skipped and the all-page filename keeps the original page numbers', async ({
  page,
}) => {
  await open(page, { emptyFirst: true })
  await page.locator('.upload-button').click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.locator('.upload-suffix')).toHaveText('-page2~page5')
  await expect(dialog.getByRole('button', { name: '上传特定页', exact: true })).toBeDisabled()
  await dialog.getByRole('button', { name: '上传全部', exact: true }).click()
  await dialog.getByRole('button', { name: '完成', exact: true }).click()
  await expect(page.locator('.task-page')).toHaveText('已上传 4 页')
  expect(await page.evaluate(() => window.__uploads[0].indices)).toEqual([])
})

test('content and writing settings changes are included in the next upload', async ({ page }) => {
  await open(page)
  await uploadAll(page)
  await page.locator('textarea.paper-input').fill('实验32 修改后的内容')
  await page.getByRole('tab', { name: '设备台', exact: true }).click()
  await page.getByLabel('X 轴偏移').fill('1')
  await page.getByLabel('X 轴偏移').blur()
  await uploadAll(page)
  const last = await page.evaluate(() => window.__uploads[1])
  expect(last.prefix).toBe('32')
  expect(last.settings.offsetX).toBe(1)
  expect(last.id).toContain('修改后的内容')
})

test('upload dialog shows a safe physical start for the whole batch without changing calibration', async ({
  page,
}, testInfo) => {
  const writing = {
    offsetX: 0,
    offsetY: 0,
    penUp: 7,
    penDown: 3.4,
    originMode: 'auto',
    sound: true,
  }
  await open(page, {
    writing,
    layout: { paperHeight: 297, marginLeft: 25.4, marginTop: 25.4 },
    pageStrokes: Array.from({ length: 5 }, (_, i) => [
      [
        { x: 25.4, y: 25.4 },
        { x: 135.4, y: i === 4 ? 270.078782 : 260 },
      ],
    ]),
  })
  await page.locator('.upload-button').click()
  const dialog = page.getByRole('dialog', { name: '上传到 A1' })
  await expect(dialog.locator('.upload-origin')).toContainText('X 0–146 mm，Y 244.7–256 mm')
  await expect(dialog.locator('.upload-origin')).toContainText('全部页')
  await page.screenshot({ path: testInfo.outputPath('upload-origin-range.png') })
  await dialog.getByRole('button', { name: '上传全部', exact: true }).click()
  await expect(dialog.locator('.upload-origin')).toContainText('Y 244.7–256 mm')
  expect(await page.evaluate(() => window.__uploads[0].settings)).toEqual(writing)
  await dialog.getByRole('button', { name: '完成', exact: true }).click()
  await page.getByRole('tab', { name: '设备台', exact: true }).click()
  await page.getByLabel('Y 轴偏移').fill('1')
  await page.getByLabel('Y 轴偏移').blur()
  await page.locator('.upload-button').click()
  await expect(dialog.locator('.upload-origin')).toContainText('Y 245.7–256 mm')
  await page.keyboard.press('Escape')
  await page.getByLabel('原点模式').selectOption('manual')
  await page.locator('.upload-button').click()
  await expect(dialog.locator('.upload-origin')).toHaveCount(0)
})

test('Chixie parameter controls preserve numeric calibration while changing origin and sound', async ({
  page,
}) => {
  const writing = {
    offsetX: 2.5,
    offsetY: -3.5,
    penUp: 8.2,
    penDown: 4.1,
    originMode: 'auto',
    sound: true,
  }
  await open(page, { writing })
  await page.getByRole('tab', { name: '设备台', exact: true }).click()
  const card = page.locator('.writing-settings-card')
  const inputs = card.locator('input[type="number"]')
  expect(
    await inputs.evaluateAll((fields) => fields.map((field) => field.getAttribute('aria-label'))),
  ).toEqual(['抬笔位置', 'X 轴偏移', '落笔位置', 'Y 轴偏移'])
  expect(await inputs.evaluateAll((fields) => fields.map((field) => field.valueAsNumber))).toEqual([
    8.2, 2.5, 4.1, -3.5,
  ])
  await expect(card.getByLabel('原点模式')).toHaveValue('auto')
  await expect(card.getByLabel('提示音')).toHaveValue('true')
  await expect(card).toContainText('自动 XY 原点')
  await expect(card).not.toContainText('自动居中')
  await expect(card.getByLabel('原点模式')).toHaveAttribute('title', /书写区域左上角/)

  await uploadAll(page)
  expect(await page.evaluate(() => window.__uploads[0].settings)).toEqual(writing)
  await card.getByLabel('原点模式').selectOption('manual')
  await card.getByLabel('提示音').selectOption('false')
  await expect(card.getByLabel('原点模式')).toHaveAttribute('title', '沿用打印机当前 XY 坐标')
  await uploadAll(page)
  expect(await page.evaluate(() => window.__uploads[1].settings)).toEqual({
    ...writing,
    originMode: 'manual',
    sound: false,
  })
  expect(await inputs.evaluateAll((fields) => fields.map((field) => field.valueAsNumber))).toEqual([
    8.2, 2.5, 4.1, -3.5,
  ])
})

test('font percentage preserves saved millimeters and follows row height after editing', async ({
  page,
}) => {
  await open(page, {
    layout: {
      paperWidth: 210,
      paperHeight: 297,
      marginTop: 25.4,
      marginBottom: 25.4,
      lineCount: 30,
      lineHeight: 246.2 / 30,
      fontSize: 7,
    },
  })
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  const panel = page.getByRole('region', { name: '排版设置' })
  const size = panel.getByLabel('字体大小', { exact: true })
  await expect(size).toHaveValue('85.3')
  await expect(size.locator('..')).toContainText('%')
  expect(await page.evaluate(() => window.__previews.at(-1).fontSize)).toBe(7)
  await size.fill('90')
  await size.press('Tab')
  await expect
    .poll(() => page.evaluate(() => window.__previews.at(-1).fontSize))
    .toBeCloseTo(7.386, 8)
  await expect(panel.getByLabel('基准行数', { exact: true })).toHaveValue('30')
  await panel.getByLabel('基准行数', { exact: true }).fill('20')
  await panel.getByLabel('基准行数', { exact: true }).press('Tab')
  await expect(size).toHaveValue('90')
  await expect
    .poll(() => page.evaluate(() => window.__savedPreferences?.layout.fontSize))
    .toBeCloseTo(11.079, 8)
  expect(await page.evaluate(() => window.__savedPreferences.writing)).toEqual({
    offsetX: 0,
    offsetY: 0,
    penUp: 7,
    penDown: 3.4,
    originMode: 'auto',
    sound: true,
  })
})

test('restoring layout defaults displays font size 110 percent', async ({ page }) => {
  await open(page)
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  const panel = page.getByRole('region', { name: '排版设置' })
  await panel.getByRole('button', { name: '恢复默认设置', exact: true }).click()
  await expect(panel.getByLabel('字体大小', { exact: true })).toHaveValue('110')
  await expect(panel.locator('.number-field').filter({ hasText: '字体大小' })).toHaveAttribute(
    'title',
    '当前字框 16.50 mm',
  )
  await expect.poll(() => page.evaluate(() => window.__previews.at(-1).fontSize)).toBe(16.5)
  await expect(panel.getByLabel('基准行数', { exact: true })).toHaveValue('16')
})

test('spacing controls retain font millimeters and accept 110 percent type', async ({ page }) => {
  await open(page, { layout: { fontSize: 15, lineHeight: 15, lineCount: 16 } })
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  const panel = page.getByRole('region', { name: '排版设置' })
  for (const label of ['字距', '行距']) {
    const field = panel.getByLabel(label, { exact: true })
    await field.fill('85')
    await field.press('Tab')
  }
  await expect
    .poll(() => page.evaluate(() => window.__savedPreferences?.layout.lineSpacing))
    .toBe(85)
  expect(await page.evaluate(() => window.__savedPreferences.layout.fontSize)).toBe(15)
  expect(await page.evaluate(() => window.__savedPreferences.layout.letterSpacing)).toBe(85)
  await expect(panel.locator('.number-field').filter({ hasText: '行距' })).toHaveAttribute(
    'title',
    '当前行距 12.75 mm，最多 18 行',
  )
  const size = panel.getByLabel('字体大小', { exact: true })
  await size.fill('110')
  await size.press('Tab')
  await expect.poll(() => page.evaluate(() => window.__previews.at(-1).fontSize)).toBe(16.5)
  await expect(size).toHaveValue('110')
})

test('partial upload exposes total progress and cancellation in the footer', async ({ page }) => {
  await open(page)
  await page.evaluate(() =>
    window.__emitPrinter({
      phase: 'uploading',
      page: 3,
      pageOrdinal: 3,
      totalPages: 5,
      uploadedPages: 2,
      progress: 46,
      jobId: 'batch',
    }),
  )
  await expect(page.locator('.task-page')).toHaveText('第 3 页 · 3/5 · 46%')
  await expect(page.getByRole('progressbar', { name: '全部页面上传进度' })).toHaveAttribute(
    'aria-valuenow',
    '46',
  )
  await page.getByRole('button', { name: '取消上传', exact: true }).click()
  await expect(page.locator('.job-progress')).toHaveCount(0)
  expect(await page.evaluate(() => window.__cancels)).toBe(1)
})

test('English labels describe uploads and control on A1', async ({ page }) => {
  await open(page, { language: 'en' })
  await page.locator('.upload-button').click()
  await page.getByRole('button', { name: 'Upload All', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Done', exact: true })).toHaveAttribute(
    'title',
    /A1 screen/,
  )
  await page.getByRole('button', { name: 'Done', exact: true }).click()
  await expect(page.locator('.task-page')).toHaveText('5 pages uploaded')
  await expect(
    page.getByRole('button', { name: /Start Writing|Start Next Page|Pause|Resume|Stop/ }),
  ).toHaveCount(0)
})

test('the dialog derives an editable prefix from the first line and defaults to the viewed page', async ({
  page,
}) => {
  await open(page)
  await page.locator('.document-viewport').evaluate((view) => {
    view.scrollTop = view.querySelector('[data-paper="3"]').offsetTop
  })
  await expect(page.locator('.page-rail button').nth(3)).toHaveAttribute('aria-current', 'page')
  await page.locator('.upload-button').click()
  const dialog = page.getByRole('dialog', { name: '上传到 A1' })
  await expect(dialog.getByLabel('文件名前缀')).toHaveValue('31')
  await expect(dialog.locator('.upload-suffix')).toHaveText('-page1~page5')
  await expect(dialog.getByLabel('特定页', { exact: true })).toHaveValue('3')
  expect(await page.evaluate(() => window.__uploads.length)).toBe(0)
  await dialog.getByLabel('文件名前缀').fill('32')
  await expect(dialog.locator('.upload-specific-name')).toHaveText('32-page4')
  await dialog.getByRole('button', { name: '上传特定页', exact: true }).click()
  await expect(dialog.locator('.upload-suffix')).toHaveText('-page4')
  await expect(dialog.getByRole('button', { name: '完成', exact: true })).toHaveAttribute(
    'title',
    /A1 屏幕/,
  )
  await dialog.getByRole('button', { name: '完成', exact: true }).click()
  expect(await page.evaluate(() => window.__uploads[0].prefix)).toBe('32')
  expect(await page.evaluate(() => window.__uploads[0].indices)).toEqual([3])
})

test('the dialog shows live progress, supports cancellation, and prevents invalid filenames', async ({
  page,
}, testInfo) => {
  await open(page)
  await page.locator('.upload-button').click()
  const dialog = page.getByRole('dialog', { name: '上传到 A1' })
  await dialog.getByLabel('文件名前缀').fill('../31')
  await expect(dialog.getByRole('button', { name: '上传全部', exact: true })).toBeDisabled()
  await expect(dialog.getByRole('button', { name: '上传特定页', exact: true })).toBeDisabled()
  await dialog.getByLabel('文件名前缀').fill('31')
  await page.screenshot({ path: testInfo.outputPath('upload-dialog.png') })
  await page.evaluate(() =>
    window.__emitPrinter({
      phase: 'uploading',
      page: 2,
      pageOrdinal: 2,
      totalPages: 5,
      uploadedPages: 1,
      progress: 35,
      jobId: 'batch',
    }),
  )
  await expect(dialog.getByRole('progressbar', { name: '全部页面上传进度' })).toHaveAttribute(
    'value',
    '35',
  )
  await expect(dialog.locator('.upload-progress-label')).toContainText('35%')
  await expect(dialog.locator('.upload-progress-label')).toContainText('第 2 页')
  await expect(dialog.getByLabel('文件名前缀')).toBeDisabled()
  await page.evaluate(() =>
    window.__emitPrinter({ page: 3, pageOrdinal: 3, uploadedPages: 2, progress: 62 }),
  )
  await expect(dialog.getByRole('progressbar')).toHaveAttribute('value', '62')
  await page.screenshot({ path: testInfo.outputPath('upload-progress.png') })
  await dialog.getByRole('button', { name: '取消上传', exact: true }).click()
  await expect(dialog.getByRole('progressbar')).toHaveCount(0)
  await expect(dialog.getByRole('button', { name: '上传全部', exact: true })).toBeEnabled()
  await page.keyboard.press('Escape')
  await expect(dialog).not.toBeVisible()
  expect(await page.evaluate(() => window.__cancels)).toBe(1)
})

test('upload errors stay visible and allow retry without reporting success', async ({ page }) => {
  await open(page)
  await page.evaluate(() => {
    window.go.application.App.UploadDevicePages = async () => {
      throw new Error('第 3 页上传失败，请重试')
    }
  })
  await page.locator('.upload-button').click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('button', { name: '上传全部', exact: true }).click()
  await expect(dialog.getByRole('alert')).toHaveText('第 3 页上传失败，请重试')
  await expect(dialog.getByRole('button', { name: '完成', exact: true })).toHaveCount(0)
  await expect(dialog.getByRole('button', { name: '上传全部', exact: true })).toBeEnabled()
})
