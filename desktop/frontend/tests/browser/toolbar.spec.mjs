import { test, expect } from '@playwright/test'

// Tools occupy the first row; options and view controls have fixed slots below.
// These drive the real editor against a stubbed bridge to pin the two things
// that are easy to break again: nothing may move when a tool changes, and the
// selected state must stay visible under the pointer.
async function stubBridge(page, withImage) {
  await page.addInitScript((withImage) => {
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
    }
    const buildDocument = (text) => ({
      id: `doc-${text.length}`,
      settings: layout,
      missing: [],
      characterCount: text.length,
      fontName: '测试字库',
      pages: [1, 2, 3, 4, 5].map((number) => ({
        number,
        lineHeight: layout.lineHeight,
        lineCount: layout.lineCount,
        strokes: [],
        textStrokes: [],
        images: [],
        missing: [],
        text: number === 1 ? text : '',
        cells: [],
        carets: [],
      })),
    })
    const images = withImage
      ? [
          {
            id: 'image-1',
            page: 0,
            data: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
            x: 20,
            y: 20,
            width: 80,
            height: 80,
            rotation: 0,
            mode: 'line',
          },
        ]
      : []
    const menuHandlers = []
    window.__menu = (action) => menuHandlers.forEach((h) => h(action))
    window.__title = ''
    window.runtime = {
      EventsOn: (name, callback) => {
        if (name === 'menu:action') menuHandlers.push(callback)
        return () => {}
      },
      WindowSetTitle: (title) => {
        window.__title = title
      },
    }
    const none = async () => ''
    window.go = {
      application: {
        App: {
          Bootstrap: async () => ({
            preferences: {
              theme: 'light',
              text: '工具栏回归样例',
              layout,
              writing: {
                offsetX: 0,
                offsetY: 0,
                penUp: 7,
                penDown: 3.4,
                originMode: 'auto',
                sound: true,
              },
              images,
            },
            printer: { ip: '', serial: '', hasAccessCode: false },
            state: {
              connected: false,
              phase: 'disconnected',
              message: '',
              progress: 0,
              jobId: '',
              documentId: '',
              page: 0,
              deviceState: '',
            },
            warning: '',
          }),
          Preview: async (text) => buildDocument(text),
          PreviewWithImages: async (text) => buildDocument(text),
          SavePreferences: async () => {},
          SaveProject: none,
          SaveProjectAs: none,
          OpenProject: async () => null,
          RevealProject: async () => {},
          SetLanguage: async () => {},
          SetMenuState: async () => {},
          ConfirmDiscardProject: async () => 'discard',
          ExportPage: none,
          ExportAll: none,
          DiscoverPrinter: none,
          ConnectPrinter: async () => {},
          DisconnectPrinter: async () => {},
          PrinterState: async () => null,
          UploadPages: none,
          CancelUpload: async () => {},
        },
      },
    }
  }, withImage)
}

async function open(page, withImage = true) {
  const errors = []
  page.on('pageerror', (error) => errors.push(error.message))
  await stubBridge(page, withImage)
  await page.goto('/')
  await expect(page.locator('.document-viewport')).toBeVisible()
  await expect(page.locator('.editor-toolbar')).toBeVisible()
  return errors
}

const box = (page, selector) =>
  page.locator(selector).evaluate((el) => {
    const r = el.getBoundingClientRect()
    return { x: Math.round(r.x), width: Math.round(r.width) }
  })
const tool = (page, name) => page.getByRole('button', { name, exact: true })

for (const [width, height] of [
  [1360, 900],
  [1100, 720],
]) {
  test(`switching tools never moves the view controls or the layout panel at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height })
    const errors = await open(page)
    const view = '.canvas-controls'
    const panel = '.editor-toolbar > button:has-text("排版设置")'
    const anchor = await Promise.all([box(page, view), box(page, panel)])
    for (const name of ['文本框', '擦除', '选择']) {
      await tool(page, name).click()
      expect(await box(page, view)).toEqual(anchor[0])
      expect(await box(page, panel)).toEqual(anchor[1])
      if (name !== '擦除') continue
      // The erase options are the widest the row ever gets, and the brush size
      // only exists in free mode.
      for (const mode of ['自由', '框选']) {
        await tool(page, mode).click()
        expect(await box(page, view)).toEqual(anchor[0])
        expect(await box(page, panel)).toEqual(anchor[1])
      }
    }
    expect(errors).toEqual([])
  })
}

test('a page without an image allows every paper tool', async ({ page }) => {
  const errors = await open(page, false)
  await expect(tool(page, '选择')).toBeEnabled()
  await expect(tool(page, '文本框')).toBeEnabled()
  await expect(tool(page, '擦除')).toBeEnabled()
  await expect(page.locator('.tool-options > *')).toHaveCount(0)
  // The paper eraser is also available through its shortcut.
  await page.keyboard.press('Control+3')
  await expect(tool(page, '擦除')).toHaveAttribute('aria-pressed', 'true')
  await page.keyboard.press('Control+2')
  await expect(tool(page, '文本框')).toHaveAttribute('aria-pressed', 'true')
  expect(errors).toEqual([])
})

test('an image page offers the erase modes and the brush sizes', async ({ page }) => {
  const errors = await open(page)
  await expect(page.locator('.tool-options > *')).toHaveCount(0)
  await tool(page, '擦除').click()
  await expect(tool(page, '框选')).toHaveAttribute('aria-pressed', 'true')
  await expect(page.locator('[aria-label="自由擦除笔刷粗细"]')).toHaveCount(0)
  await tool(page, '自由').click()
  await expect(page.locator('[aria-label="自由擦除笔刷粗细"]')).toHaveValue('0.018')
  await tool(page, '框选').click()
  await expect(page.locator('[aria-label="自由擦除笔刷粗细"]')).toHaveCount(0)
  // Leaving the erase tool takes its options away again.
  await tool(page, '选择').click()
  await expect(page.locator('.tool-options > *')).toHaveCount(0)
  expect(errors).toEqual([])
})

test('the selected tool and erase mode keep their fill under the pointer', async ({ page }) => {
  const errors = await open(page)
  const fill = (locator) => locator.evaluate((el) => getComputedStyle(el).backgroundColor)
  const token = (name) =>
    page.evaluate((name) => {
      const probe = document.createElement('span')
      probe.style.background = `var(${name})`
      document.body.append(probe)
      const value = getComputedStyle(probe).backgroundColor
      probe.remove()
      return value
    }, name)
  const accent = await token('--surface-selected')
  const hover = await token('--surface-hover')
  await tool(page, '擦除').click()
  for (const mode of ['自由', '框选']) {
    await tool(page, mode).click()
    await tool(page, mode).hover()
    expect(await fill(tool(page, mode))).toBe(accent)
  }
  await tool(page, '擦除').hover()
  expect(await fill(tool(page, '擦除'))).toBe(accent)
  await tool(page, '自由').hover()
  expect(await fill(tool(page, '自由'))).toBe(hover)
  expect(errors).toEqual([])
})

test('the offline job bar states the connection once', async ({ page }) => {
  const errors = await open(page, false)
  const status = page.locator('.connection-status')
  await expect(status.locator('.status-message')).toHaveCount(0)
  await expect(status.locator('.status-divider')).toHaveCount(0)
  await expect(status).toHaveText('A1 · 未连接')
  expect(errors).toEqual([])
})

test('tool shortcuts work outside the paper and stay out of the text', async ({ page }) => {
  const errors = await open(page)
  const paper = page.locator('textarea.paper-input')
  await paper.fill('')
  await paper.focus()
  // Bare letters belong to the document while the paper has focus.
  await page.keyboard.type('vet')
  await expect(paper).toHaveValue('vet')
  await expect(tool(page, '选择')).toHaveAttribute('aria-pressed', 'true')
  // The modifier form works while typing.
  await page.keyboard.press('Control+2')
  await expect(tool(page, '文本框')).toHaveAttribute('aria-pressed', 'true')
  await page.keyboard.press('Escape')
  await expect(tool(page, '选择')).toHaveAttribute('aria-pressed', 'true')
  expect(errors).toEqual([])
})
