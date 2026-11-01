import { test as base, expect } from '@playwright/test'
import { execFileSync } from 'node:child_process'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'

let binary, binDir
const moduleRoot = resolve(process.cwd(), '..')
export const test = base.extend({
  tableBackend: [
    async ({}, use) => {
      binDir = mkdtempSync(`${tmpdir()}/bambuscribe-table-engine-`)
      binary = `${binDir}/preview`
      execFileSync('go', ['build', '-o', binary, './tests/table-preview'], { cwd: moduleRoot })
      try {
        await use(binary)
      } finally {
        rmSync(binDir, { recursive: true, force: true })
      }
    },
    { scope: 'worker', auto: true },
  ],
})
export async function open(page, update = {}, compact = false) {
  const errors = []
  page.on('pageerror', (e) => errors.push(e.message))
  await page.route('**/__table-preview', async (route) => {
    try {
      const result = execFileSync(binary, {
        input: route.request().postData(),
        maxBuffer: 20 * 1024 * 1024,
      }).toString()
      await route.fulfill({ status: 200, contentType: 'application/json', body: result })
    } catch (e) {
      await route.fulfill({ status: 400, body: e.stderr?.toString() ?? e.message })
    }
  })
  await page.addInitScript((enabled) => {
    window.__compactPreview = enabled
  }, compact)
  await page.addInitScript((update) => {
    const compact = window.__compactPreview
    const layout = {
      templateName: '表格测试',
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
      letterSpacing: 100,
      lineSpacing: 100,
      lineHeight: 15,
      paragraphGap: 0,
      indent: 0,
      align: 'left',
    }
    const preferences = {
      text: '',
      theme: 'light',
      language: 'zh',
      layout,
      images: [],
      tables: [],
      writing: {
        offsetX: 0,
        offsetY: 0,
        penUp: 7,
        penDown: 3.4,
        originMode: 'auto',
        sound: true,
      },
    }
    const state = {
      connected: false,
      phase: 'disconnected',
      message: '',
      progress: 0,
      jobId: '',
      documentId: '',
      page: 0,
      deviceState: '',
      totalPages: 0,
      pageOrdinal: 0,
      uploadedPages: 0,
    }
    window.__tables = { preferences, document: null, saved: null, exported: null, menu: {} }
    window.runtime = {
      EventsOn: (name, fn) => {
        window.__tables.menu[name] = fn
        return () => {}
      },
      WindowSetTitle: () => {},
    }
    const preview = async (text, settings, images, tables = [], erasures = [], packed = false) => {
      const payload = { text, settings, images, tables, erasures, compact: packed }
      window.__tables.payload = payload
      const r = await fetch('/__table-preview', { method: 'POST', body: JSON.stringify(payload) })
      if (!r.ok) throw new Error(await r.text())
      const doc = await r.json()
      window.__tables.document = packed
        ? (
            await import(`${window.location.origin}/src/services/editorPreview.ts`)
          ).unpackEditorPreview(structuredClone(doc))
        : doc
      return doc
    }
    const none = async () => ''
    window.go = {
      application: {
        App: {
          CheckUpdate: async () => {
            window.__tables.updateChecks = (window.__tables.updateChecks ?? 0) + 1
            if (update.fail) throw new Error('无法连接更新服务，请检查网络后重试。')
            const info = {
              current: '0.1.3',
              version: '0.1.3',
              available: false,
              notes: '',
              url: '',
              ...update,
            }
            if (update.defer)
              return new Promise((resolve) => {
                window.__tables.finishCheck = () => resolve(info)
              })
            return info
          },
          Bootstrap: async () => ({
            preferences,
            printer: { ip: '', serial: '', hasAccessCode: false },
            state,
            warning: '',
          }),
          Preview: preview,
          PreviewWithImages: preview,
          PreviewWithObjects: preview,
          PreviewWithErasures: preview,
          ...(compact
            ? {
                PreviewEditor: (text, settings, images, tables, erasures) =>
                  preview(text, settings, images, tables, erasures, true),
              }
            : {}),
          SavePreferences: async (p, path) => {
            window.__tables.preferences = p
            if (path) {
              const { text, layout, writing, images, tables, erasures, calculations } = p
              window.__tables.saved = {
                version: 7,
                calculations,
                text,
                layout,
                writing,
                images,
                tables,
                erasures,
              }
            }
          },
          SaveProjectAs: async (p) => {
            window.__tables.saved = p
            return '/tmp/table.ccf'
          },
          SaveProject: async (p) => {
            window.__tables.saved = p
            return '/tmp/table.ccf'
          },
          OpenProject: async () =>
            window.__tables.saved
              ? { path: '/tmp/table.ccf', project: window.__tables.saved }
              : null,
          NewProject: none,
          ConfirmDiscardProject: async () => 'discard',
          RevealProject: none,
          SetLanguage: none,
          SetMenuState: none,
          ExportPage: async (_id, index) => {
            const r = await fetch('/__table-preview', {
              method: 'POST',
              body: JSON.stringify({ ...window.__tables.payload, export: true, page: index }),
            })
            if (!r.ok) throw new Error(await r.text())
            window.__tables.exported = await r.json()
            return '/tmp/table.gcode'
          },
          ExportAll: none,
          PrinterState: async () => state,
          ConnectPrinter: none,
          DisconnectPrinter: none,
          DiscoverPrinter: none,
          UploadPages: none,
          CancelUpload: none,
        },
      },
    }
  }, update)
  await page.goto('/')
  await expect(page.getByRole('button', { name: '表格', exact: true })).toBeEnabled()
  return errors
}
export async function insert(page, options = {}) {
  await page.getByRole('button', { name: '表格', exact: true }).click()
  await page.getByLabel('表格行数', { exact: true }).fill(String(options.rows ?? 3))
  await page.getByLabel('表格列数', { exact: true }).fill(String(options.cols ?? 3))
  await page.getByLabel('表格宽度', { exact: true }).fill(String(options.width ?? 105))
  if (options.style) await page.getByLabel('表格样式', { exact: true }).selectOption(options.style)
  await page.getByRole('button', { name: '插入', exact: true }).click()
  await expect(page.locator('[data-table]')).toHaveCount(1)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.tables?.length))
    .toBe(1)
}
export async function table(page, index = 0) {
  return page.evaluate((i) => window.__tables.payload.tables[i], index)
}
export async function waitText(page, text, row = 0, col = 0) {
  await expect
    .poll(() =>
      page.evaluate(
        ({ row, col }) =>
          window.__tables.payload.tables[0]?.cells.find((c) => c.row === row && c.col === col)
            ?.text,
        { row, col },
      ),
    )
    .toBe(text)
}
export async function cell(page, row = 0, col = 0) {
  const el = page.locator(`[data-table-cell="${row}:${col}"]`)
  await el.scrollIntoViewIfNeeded()
  return el
}
export async function clipboard(page, type, text, target = '.document-viewport', mime) {
  await page.evaluate(
    ({ type, text, target, mime }) => {
      const data = new DataTransfer()
      data.setData('text/plain', text)
      if (mime) data.setData('application/x-bambuscribe-table-v1', mime)
      document
        .querySelector(target)
        .dispatchEvent(
          new ClipboardEvent(type, { bubbles: true, cancelable: true, clipboardData: data }),
        )
    },
    { type, text, target, mime },
  )
}
