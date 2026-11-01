import { test, expect } from '@playwright/test'

// Clipboard text from a rich-text view (Gemini's answers, for one) arrives with
// its block boundaries flattened into runs of spaces. The editor puts the line
// breaks back, and only the browser can tell it a paste happened, so these
// drive the real pasteboard. Playwright can fill it in Chromium only.
const FLATTENED =
  '实验三十一 普朗克常量的测定   一、实验目的   1.通过实验了解光的量子性.\n2.掌握用光电效应方程测定普朗克常量的方法.'
const TIDIED =
  '实验三十一 普朗克常量的测定\n一、实验目的\n1.通过实验了解光的量子性.\n2.掌握用光电效应方程测定普朗克常量的方法.'

async function stubBridge(page) {
  await page.addInitScript(() => {
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
      pages: [
        {
          number: 1,
          lineHeight: layout.lineHeight,
          lineCount: layout.lineCount,
          strokes: [],
          textStrokes: [],
          images: [],
          missing: [],
          text,
          cells: [],
          carets: [],
        },
      ],
    })
    window.__previewed = []
    window.__saved = []
    const none = async () => ''
    window.runtime = { EventsOn: () => () => {}, WindowSetTitle: () => {} }
    window.go = {
      application: {
        App: {
          Bootstrap: async () => ({
            preferences: {
              theme: 'light',
              text: '',
              layout,
              writing: {
                offsetX: 0,
                offsetY: 0,
                penUp: 7,
                penDown: 3.4,
                originMode: 'auto',
                sound: true,
              },
              images: [],
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
          Preview: async (text) => {
            window.__previewed.push(text)
            return buildDocument(text)
          },
          PreviewWithImages: async (text) => {
            window.__previewed.push(text)
            return buildDocument(text)
          },
          SavePreferences: async (preferences) => {
            window.__saved.push(preferences.text)
          },
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
  })
}

// A paste is only a paste to the editor when the browser says so, so the
// clipboard has to be the real one.
async function paste(page, text) {
  await page.evaluate((value) => navigator.clipboard.writeText(value), text)
  const input = page.locator('textarea.paper-input')
  await input.focus()
  await page.keyboard.press('Meta+V')
  await page.waitForTimeout(200)
  return input
}

test('a flattened paste gets its line breaks back, and one undo restores it', async ({
  page,
  context,
  browserName,
}) => {
  test.skip(browserName !== 'chromium', 'the pasteboard is only reachable in Chromium here')
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await stubBridge(page)
  await page.goto('/')
  await page.waitForSelector('textarea.paper-input')

  const input = await paste(page, FLATTENED)
  await expect(input).toHaveValue(TIDIED)
  // The caret follows the repaired text instead of the longer clipboard text.
  expect(await input.evaluate((el) => el.selectionStart)).toBe(TIDIED.length)
  await expect(page.locator('.notice')).toContainText('2 处连续空格')
  // The paper is typeset from the repaired text, not the clipboard's spacing.
  await expect.poll(() => page.evaluate(() => window.__previewed.at(-1))).toBe(TIDIED)

  // One undo brings the clipboard text back untouched, a second removes the
  // paste altogether: the repair is its own step, never fused with the paste.
  await page.keyboard.press('Meta+Z')
  await expect(input).toHaveValue(FLATTENED)
  await page.keyboard.press('Meta+Z')
  await expect(input).toHaveValue('')
})

// WKWebView is not Chromium: the repair is wired to the paste event as well,
// because that is the signal every engine reports. This drives that path alone.
test('the paste event itself is enough to trigger the repair', async ({ page }) => {
  await stubBridge(page)
  await page.goto('/')
  const input = page.locator('textarea.paper-input')
  await input.focus()
  await input.evaluate((el) => {
    el.dispatchEvent(new ClipboardEvent('paste', { bubbles: true, cancelable: true }))
    el.value = '普朗克常量的测定   一、实验目的'
    el.setSelectionRange(el.value.length, el.value.length)
    el.dispatchEvent(new Event('input', { bubbles: true }))
  })
  await expect(input).toHaveValue('普朗克常量的测定\n一、实验目的')
  await expect(page.locator('.notice')).toContainText('1 处连续空格')
})

// The paste signals are engine-dependent, so a block of text that arrives
// without one is repaired on its size alone.
test('a block of text inserted without a paste event is repaired too', async ({ page }) => {
  await stubBridge(page)
  await page.goto('/')
  const input = page.locator('textarea.paper-input')
  await input.focus()
  await input.evaluate((el) => {
    el.value = '汞灯及电源、滤色片   光电管'
    el.setSelectionRange(el.value.length, el.value.length)
    el.dispatchEvent(new Event('input', { bubbles: true }))
  })
  await expect(input).toHaveValue('汞灯及电源、滤色片\n光电管')
  // Typing is not a block: two spaces pressed by hand stay two spaces.
  await input.evaluate((el) => {
    el.value = '两个  空格'
    el.dispatchEvent(new Event('input', { bubbles: true }))
  })
  await expect(input).toHaveValue('两个  空格')
})

test('a paste that already has line breaks is left alone', async ({
  page,
  context,
  browserName,
}) => {
  test.skip(browserName !== 'chromium', 'the pasteboard is only reachable in Chromium here')
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await stubBridge(page)
  await page.goto('/')
  await page.waitForSelector('textarea.paper-input')

  const input = await paste(page, '第一行\n第二行 第三行')
  await expect(input).toHaveValue('第一行\n第二行 第三行')
  await expect(page.locator('.notice')).toHaveCount(0)
})
