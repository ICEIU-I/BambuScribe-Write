import { test, expect } from '@playwright/test'

// The Go side of the bridge is covered by internal/project tests; these drive
// the real editor against a stubbed bridge to check the wiring in between.
async function stubBridge(page, restored = null) {
  await page.addInitScript((restored) => {
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
    // Not named `document`: that binding already exists on window and a
    // redeclaration would abort this whole init script.
    const buildDocument = (text) => ({
      id: `doc-${text.length}-${text}`,
      settings: layout,
      missing: [],
      characterCount: text.length,
      fontName: '测试字库',
      pages: [
        {
          number: 1,
          lineHeight: layout.lineHeight,
          lineCount: layout.lineCount,
          strokes: [
            [
              { x: 5, y: 5 },
              { x: 40, y: 5 },
            ],
          ],
          textStrokes: [],
          images: [],
          missing: [],
          text,
          cells: [],
          carets: [],
        },
      ],
    })
    const calls = []
    let opened = null
    let confirm = 'discard'
    let currentPath = restored?.projectPath || ''
    window.__saveMode = 'ok'
    window.__saveAsMode = 'ok'
    window.__saveAsPath = '/tmp/另存项目.ccf'
    window.__autosaveMode = 'ok'
    window.__pauseAutosave = false
    window.__projects = {}
    window.__events = {}
    window.__newError = false
    window.__pauseConfirm = false
    window.__preferences = null
    window.__calls = calls
    window.__setOpenResult = (value) => {
      opened = value
    }
    window.__setConfirm = (value) => {
      confirm = value
    }
    const menuHandlers = []
    window.__menu = (action) => menuHandlers.forEach((h) => h(action))
    window.__title = ''
    window.runtime = {
      EventsOn: (name, callback) => {
        window.__events[name] = callback
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
            projectPath: restored?.projectPath || '',
            preferences: restored?.preferences || {
              theme: 'light',
              text: '原始正文',
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
          Preview: async (text) => buildDocument(text),
          PreviewWithImages: async (text) => buildDocument(text),
          SavePreferences: async (value, path) => {
            calls.push({ kind: 'autosave', preferences: value, path })
            if (window.__pauseAutosave)
              await new Promise((resolve) => {
                window.__resolveAutosave = resolve
              })
            if (window.__autosaveMode === 'fail') throw new Error('自动保存失败：磁盘不可写')
            if (path !== currentPath) throw new Error('项目已切换')
            window.__preferences = path
              ? { ...value, text: '', images: [], tables: [], erasures: [] }
              : value
            if (path) window.__projects[path] = value
          },
          SaveProject: async (project) => {
            calls.push({ kind: 'save', project, path: currentPath })
            if (window.__saveMode === 'cancel') return ''
            if (window.__saveMode === 'fail') throw new Error('保存失败：磁盘不可写')
            currentPath ||= '/tmp/书写项目.ccf'
            window.__projects[currentPath] = project
            return currentPath
          },
          SaveProjectAs: async (project) => {
            calls.push({ kind: 'saveAs', project })
            if (window.__saveAsMode === 'cancel') return ''
            if (window.__saveAsMode === 'fail') throw new Error('另存为失败')
            currentPath = window.__saveAsPath
            window.__projects[currentPath] = project
            return currentPath
          },
          OpenProject: async () => {
            calls.push({ kind: 'open' })
            if (opened) currentPath = opened.path
            return opened
          },
          NewProject: async () => {
            calls.push({ kind: 'new' })
            if (window.__newError) throw new Error('新建失败')
            currentPath = ''
          },
          RevealProject: async (path) => {
            calls.push({ kind: 'reveal', path })
          },
          SetLanguage: async () => {},
          SetMenuState: async () => {},
          CompleteClose: async () => {
            calls.push({ kind: 'close' })
          },
          ConfirmDiscardProject: async () => {
            calls.push({ kind: 'confirm' })
            if (window.__pauseConfirm)
              return new Promise((resolve) => {
                window.__resolveConfirm = resolve
              })
            return confirm
          },
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
  }, restored)
}

// Save, open and export live in the native menu bar now, which Playwright
// cannot click; the stub records the menu:action channel so these drive the
// same path the menu does.
const menu = (page, action) => page.evaluate((a) => window.__menu(a), action)
const title = (page) => page.evaluate(() => window.__title)
const lastCall = (page, kind) =>
  page.evaluate((kind) => window.__calls.filter((call) => call.kind === kind).at(-1) ?? null, kind)

async function open(page, restored = null) {
  const errors = []
  page.on('pageerror', (error) => errors.push(error.message))
  await stubBridge(page, restored)
  await page.goto('/')
  await expect(page.locator('.document-viewport')).toBeVisible()
  await expect(page.locator('textarea.paper-input')).toBeEnabled()
  return errors
}

test('saving a project sends the document without the workspace theme', async ({ page }) => {
  const errors = await open(page)
  await expect.poll(() => title(page)).toBe('原始正文')
  await menu(page, 'saveAs')
  const call = await lastCall(page, 'saveAs')
  expect(call.project.version).toBe(6)
  expect(call.project.text).toBe('原始正文')
  expect(call.project.theme).toBeUndefined()
  expect(call.project.writing.penDown).toBe(3.4)
  // The notice names the written file and the window title carries its name.
  await expect(page.locator('.notice')).toContainText('已保存到')
  await expect(page.locator('.notice .path-link')).toHaveText('/tmp/另存项目.ccf')
  await page.locator('.notice .path-link').click()
  expect((await lastCall(page, 'reveal')).path).toBe('/tmp/另存项目.ccf')
  await expect.poll(() => title(page)).toBe('另存项目')
  expect(await title(page)).not.toContain('已修改')
  expect(errors).toEqual([])
})

test('opening a project warns about unsaved edits and then replaces the document', async ({
  page,
}) => {
  const errors = await open(page)
  await page.locator('textarea.paper-input').fill('未保存的新内容')
  await expect.poll(() => title(page)).toContain('已修改')
  await page.evaluate(() =>
    window.__setOpenResult({
      path: '/tmp/会议记录.ccf',
      project: { version: 1, text: '被打开的项目', layout: null, writing: null, images: [] },
    }),
  )
  await menu(page, 'open')
  await expect(page.locator('.notice')).toContainText('已打开')
  await expect(page.locator('.notice .path-link')).toHaveText('/tmp/会议记录.ccf')
  expect((await lastCall(page, 'confirm')).kind).toBe('confirm')
  expect((await lastCall(page, 'open')).kind).toBe('open')
  await expect(page.locator('textarea.paper-input')).toHaveValue('被打开的项目')
  await expect.poll(() => title(page)).not.toContain('已修改')
  expect(errors).toEqual([])
})

test('cancelling the warning keeps the edited document', async ({ page }) => {
  const errors = await open(page)
  await page.locator('textarea.paper-input').fill('未保存的新内容')
  await expect.poll(() => title(page)).toContain('已修改')
  await page.evaluate(() => window.__setConfirm('cancel'))
  await page.evaluate(() =>
    window.__setOpenResult({
      path: '/tmp/会议记录.ccf',
      project: { version: 1, text: '被打开的项目', layout: null, writing: null, images: [] },
    }),
  )
  await menu(page, 'open')
  expect((await lastCall(page, 'confirm')).kind).toBe('confirm')
  expect(await lastCall(page, 'open')).toBeNull()
  await expect(page.locator('textarea.paper-input')).toHaveValue('未保存的新内容')
  expect(errors).toEqual([])
})

test('new button protects a restored draft, creates one blank page and preserves preferences', async ({
  page,
}) => {
  const errors = await open(page)
  await page.getByRole('button', { name: '深色主题' }).click()
  await page.getByRole('button', { name: '排版设置', exact: true }).click()
  await expect(page.getByLabel('模板名称', { exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: '关闭排版设置' }).click()
  await page.getByRole('tab', { name: '设备台' }).click()
  await page.evaluate(() => window.__setConfirm('cancel'))
  await page.getByRole('button', { name: '新建空项目', exact: true }).click()
  expect(await lastCall(page, 'confirm')).not.toBeNull()
  expect(await lastCall(page, 'new')).toBeNull()
  await expect(page.locator('textarea.paper-input')).toHaveValue('原始正文')
  await page.evaluate(() => window.__setConfirm('discard'))
  await page.getByRole('button', { name: '新建空项目', exact: true }).click()
  await expect(page.locator('textarea.paper-input')).toHaveValue('')
  await expect(page.locator('textarea.paper-input')).toBeFocused()
  await expect(page.getByRole('tab', { name: '书写台' })).toHaveAttribute('aria-selected', 'true')
  await expect(page.locator('.page-rail button')).toHaveCount(1)
  await expect(page.locator('.document-breadcrumb strong')).toHaveText('未命名')
  await expect.poll(() => title(page)).toBe('未命名')
  await expect.poll(() => page.evaluate(() => window.__preferences?.text)).toBe('')
  const saved = await page.evaluate(() => window.__preferences)
  expect(saved.images).toEqual([])
  expect(saved.tables).toEqual([])
  expect(saved.erasures).toEqual([])
  expect(saved.theme).toBe('dark')
  expect(saved.layout.paperWidth).toBe(163)
  expect(saved.layout.fontSize).toBe(6)
  expect(saved.writing.penDown).toBe(3.4)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('textarea.paper-input')).toHaveValue('')
  expect(errors).toEqual([])
})

test('new via menu or shortcut waits for autosave and detaches the old file', async ({ page }) => {
  const errors = await open(page)
  await menu(page, 'saveAs')
  await expect.poll(() => title(page)).toBe('另存项目')
  await page.locator('textarea.paper-input').fill('需要保留的改动')
  await page.evaluate(() => window.__setConfirm('save'))
  await page.evaluate(() => {
    window.__autosaveMode = 'fail'
  })
  await menu(page, 'new')
  await expect(page.getByRole('button', { name: '新建空项目', exact: true })).toBeEnabled()
  await expect(page.locator('textarea.paper-input')).toHaveValue('需要保留的改动')
  expect(await lastCall(page, 'new')).toBeNull()
  expect((await lastCall(page, 'autosave')).path).toBe('/tmp/另存项目.ccf')
  await expect(page.locator('.notice[role="alert"]')).toContainText('保存失败')
  await page.evaluate(() => {
    window.__autosaveMode = 'ok'
  })
  await page.keyboard.press('Meta+n')
  await expect(page.locator('textarea.paper-input')).toHaveValue('')
  expect((await lastCall(page, 'autosave')).preferences.text).toBe('需要保留的改动')
  await expect.poll(() => title(page)).toBe('未命名')
  await page.locator('textarea.paper-input').fill('新的未命名内容')
  await page.keyboard.press('Control+n')
  await expect(page.locator('textarea.paper-input')).toHaveValue('')
  expect((await lastCall(page, 'save')).path).toBe('')
  expect((await lastCall(page, 'save')).project.text).toBe('新的未命名内容')
  expect(errors).toEqual([])
})

test('only one replacement can run and a failed new request leaves the document intact', async ({
  page,
}) => {
  const errors = await open(page)
  await page.evaluate(() => {
    window.__pauseConfirm = true
  })
  await menu(page, 'new')
  await expect(page.getByRole('button', { name: '新建空项目', exact: true })).toBeDisabled()
  await menu(page, 'new')
  await menu(page, 'open')
  expect(await page.evaluate(() => window.__calls.filter((c) => c.kind === 'confirm').length)).toBe(
    1,
  )
  await page.evaluate(() => {
    window.__newError = true
    window.__resolveConfirm('discard')
  })
  await expect(page.locator('.notice[role="alert"]')).toContainText('新建失败')
  await expect(page.locator('textarea.paper-input')).toHaveValue('原始正文')
  expect(await lastCall(page, 'open')).toBeNull()
  expect(errors).toEqual([])
})

test('named documents autosave to the latest successful save-as and clear their dirty indicators', async ({
  page,
}) => {
  const errors = await open(page)
  await menu(page, 'saveAs')
  await page.locator('textarea.paper-input').fill('第一次自动保存')
  await expect(page.locator('.unsaved-mark')).toHaveCount(1)
  await expect
    .poll(() => page.evaluate(() => window.__projects['/tmp/另存项目.ccf']?.text))
    .toBe('第一次自动保存')
  await expect(page.locator('.unsaved-mark')).toHaveCount(0)
  await expect(page.locator('.save-status')).toHaveText('已保存')
  await page.evaluate(() => {
    window.__saveAsPath = '/tmp/最近项目.ccf'
  })
  await page.keyboard.press('Meta+Shift+s')
  await expect.poll(() => title(page)).toBe('最近项目')
  await page.locator('textarea.paper-input').fill('只更新最近文件')
  await expect
    .poll(() => page.evaluate(() => window.__projects['/tmp/最近项目.ccf']?.text))
    .toBe('只更新最近文件')
  expect(await page.evaluate(() => window.__projects['/tmp/另存项目.ccf'].text)).toBe(
    '第一次自动保存',
  )
  expect(await page.evaluate(() => window.__preferences.text)).toBe('')
  for (const mode of ['cancel', 'fail']) {
    await page.evaluate((mode) => {
      window.__saveAsMode = mode
      window.__saveAsPath = '/tmp/不应成为目标.ccf'
    }, mode)
    await menu(page, 'saveAs')
    await expect(page.getByRole('button', { name: '另存为', exact: true })).toBeEnabled()
    await page.locator('textarea.paper-input').fill(`另存为${mode}后继续`)
    await expect
      .poll(() => page.evaluate(() => window.__projects['/tmp/最近项目.ccf']?.text))
      .toBe(`另存为${mode}后继续`)
    expect(await page.evaluate(() => window.__projects['/tmp/不应成为目标.ccf'])).toBeUndefined()
    await expect.poll(() => title(page)).toBe('最近项目')
  }
  const saveAsCalls = await page.evaluate(
    () => window.__calls.filter((c) => c.kind === 'saveAs').length,
  )
  await page.keyboard.press('Meta+s')
  await expect.poll(() => lastCall(page, 'save')).toMatchObject({ path: '/tmp/最近项目.ccf' })
  expect(await page.evaluate(() => window.__calls.filter((c) => c.kind === 'saveAs').length)).toBe(
    saveAsCalls,
  )
  expect(errors).toEqual([])
})

test('autosave serializes edits and save-as waits for the previous file write', async ({
  page,
}) => {
  const errors = await open(page)
  await menu(page, 'saveAs')
  await page.evaluate(() => {
    window.__pauseAutosave = true
  })
  await page.locator('textarea.paper-input').fill('旧文件保存中')
  await expect
    .poll(() => lastCall(page, 'autosave'))
    .toMatchObject({ path: '/tmp/另存项目.ccf', preferences: { text: '旧文件保存中' } })
  await page.locator('textarea.paper-input').fill('保存期间的新编辑')
  await page.evaluate(() => {
    window.__saveAsPath = '/tmp/新位置.ccf'
  })
  await menu(page, 'saveAs')
  expect(await page.evaluate(() => window.__calls.filter((c) => c.kind === 'saveAs').length)).toBe(
    1,
  )
  await page.evaluate(() => {
    window.__pauseAutosave = false
    window.__resolveAutosave()
  })
  await expect.poll(() => title(page)).toBe('新位置')
  await expect
    .poll(() => page.evaluate(() => window.__projects['/tmp/新位置.ccf']?.text))
    .toBe('保存期间的新编辑')
  expect(await page.evaluate(() => window.__projects['/tmp/另存项目.ccf'].text)).toBe(
    '旧文件保存中',
  )
  await expect(page.locator('.unsaved-mark')).toHaveCount(0)
  expect(errors).toEqual([])
})

test('close flushes pending edits and a save failure keeps the document open', async ({ page }) => {
  const errors = await open(page)
  await menu(page, 'saveAs')
  await page.locator('textarea.paper-input').fill('关闭前最后一次编辑')
  await page.evaluate(() => {
    window.__autosaveMode = 'fail'
    window.__events['workspace:close']()
  })
  await expect(page.locator('.notice')).toContainText('自动保存失败')
  expect(await lastCall(page, 'close')).toBeNull()
  await expect(page.locator('.unsaved-mark')).toHaveCount(1)
  await expect(page.locator('.save-status')).toHaveText('未保存')
  await page.evaluate(() => {
    window.__autosaveMode = 'ok'
    window.__events['workspace:close']()
  })
  await expect.poll(() => lastCall(page, 'close')).not.toBeNull()
  expect(await page.evaluate(() => window.__projects['/tmp/另存项目.ccf'].text)).toBe(
    '关闭前最后一次编辑',
  )
  expect(await page.evaluate(() => window.__preferences.text)).toBe('')
  expect(errors).toEqual([])
})

test('restart restores the named file and subsequent edits save to it', async ({ page }) => {
  const errors = await open(page, {
    projectPath: '/tmp/恢复项目.ccf',
    preferences: {
      theme: 'dark',
      language: 'zh',
      text: '文件中的最新正文',
      images: [],
      tables: [],
      erasures: [],
    },
  })
  await expect(page.locator('.document-breadcrumb strong')).toHaveText('恢复项目')
  await expect.poll(() => title(page)).toBe('恢复项目')
  await expect(page.locator('.unsaved-mark')).toHaveCount(0)
  await page.locator('textarea.paper-input').fill('重启后继续编辑')
  await expect
    .poll(() => page.evaluate(() => window.__projects['/tmp/恢复项目.ccf']?.text))
    .toBe('重启后继续编辑')
  await expect.poll(() => title(page)).toBe('恢复项目')
  expect(await lastCall(page, 'saveAs')).toBeNull()
  expect(errors).toEqual([])
})

test('unnamed draft status is explicit and titles follow the first line and language', async ({
  page,
}) => {
  const errors = await open(page)
  await page.locator('textarea.paper-input').fill('实验二十三霍尔效应\n正文')
  await expect.poll(() => title(page)).toBe('实验二十三霍尔效应 — 已修改')
  await expect(page.locator('.document-breadcrumb strong')).toHaveText('实验二十三霍尔效应')
  await expect(page.locator('.save-status')).toHaveText('草稿已保存')
  await menu(page, 'lang:en')
  await expect.poll(() => title(page)).toBe('实验二十三霍尔效应 — Edited')
  await expect(page.locator('.save-status')).toHaveText('Draft saved')
  expect(await page.evaluate(() => window.__preferences.text)).toBe('实验二十三霍尔效应\n正文')
  expect(errors).toEqual([])
})
