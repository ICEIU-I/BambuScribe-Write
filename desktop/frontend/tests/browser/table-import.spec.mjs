import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { test, open, insert, cell } from './tableHarness.mjs'
const imagePath = new URL(
  '../../../internal/engine/testdata/ocr-formula-table.png',
  import.meta.url,
)
const extraction = {
  tables: [
    {
      title: '表1 U_{AK}关系',
      notes: [],
      rows: 2,
      columns: [2, 1, 1],
      style: 'grid',
      cells: [
        { row: 0, col: 0, rowSpan: 2, colSpan: 1, text: 'λ_{i}', uncertain: false, borders: null },
        {
          row: 0,
          col: 1,
          rowSpan: 1,
          colSpan: 2,
          text: '10¹⁴ Hz',
          uncertain: false,
          borders: null,
        },
        { row: 1, col: 1, rowSpan: 1, colSpan: 1, text: '', uncertain: false, borders: null },
        { row: 1, col: 2, rowSpan: 1, colSpan: 1, text: '', uncertain: false, borders: null },
      ],
    },
  ],
}
async function setup(page) {
  const errors = await open(page)
  await page.evaluate((extraction) => {
    const api = window.go.application.App
    window.__tables.replies = []
    window.__tables.plans = 0
    Object.assign(api, {
      AgentConfig: async () => ({
        baseURL: 'https://test.invalid/v1',
        model: 'test-vision',
        protocol: 'chat-completions',
        hasKey: true,
      }),
      SaveAgentConfig: async (c) => {
        window.__tables.config = c
      },
      RecognizeTables: async () => {
        await new Promise((r) => setTimeout(r, 30))
        return extraction
      },
      CancelTableRecognition: async () => {},
      SetEditorReady: async () => {},
      CompleteEditorRequest: async (id, reply) => {
        window.__tables.replies.push({ id, ...reply })
      },
      PlanTableImport: async (project, extraction, placement) => {
        window.__tables.plans++
        const r = await fetch('/__table-preview', {
          method: 'POST',
          body: JSON.stringify({ import: { project, extraction, placement } }),
        })
        if (!r.ok) throw new Error(await r.text())
        const plan = await r.json()
        if (window.__tables.delayPlan) await new Promise((resolve) => setTimeout(resolve, 400))
        return plan
      },
    })
  }, extraction)
  return errors
}
async function send(page, params, id = 'test-import') {
  await page.evaluate(
    ({ params, id }) => window.__tables.menu['editor:request']({ id, action: 'import', params }),
    { params, id },
  )
}
test('image import creates native merged cells and one undo restores all objects', async ({
  page,
}) => {
  const errors = await setup(page)
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入表格', exact: true }).click()
  await expect(page.getByRole('dialog', { name: '智能导入表格' })).toBeVisible()
  await page
    .getByLabel('表格图片文件')
    .setInputFiles({ name: 'table.png', mimeType: 'image/png', buffer: readFileSync(imagePath) })
  await page.getByRole('button', { name: '识别并预览' }).click()
  await expect(page.getByLabel('导入排版预览')).toBeVisible()
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  await page.getByRole('button', { name: '全部插入' }).click()
  await expect(page.getByRole('dialog')).not.toBeVisible()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(2)
  const state = await page.evaluate(() => window.__tables.payload)
  expect(state.images).toHaveLength(0)
  expect(state.tables[1].cells[0].rowSpan).toBe(2)
  expect(state.tables[1].fontSize).toBe(state.settings.fontSize)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(2)
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入表格', exact: true }).click()
  await page.getByRole('button', { name: /模型连接/ }).click()
  await expect(page.getByLabel('模型 API Key')).toHaveValue('')
  await page.screenshot({ path: '/tmp/bambuscribe-import-dialog.png' })
  expect(errors).toEqual([])
})
test('live import replans against inline edits and preserves the original table', async ({
  page,
}) => {
  const errors = await setup(page)
  await insert(page, { rows: 1, cols: 1, width: 80 })
  const original = await page.evaluate(() => window.__tables.payload.tables[0].id)
  await page.evaluate(() => {
    window.__tables.delayPlan = true
  })
  await send(page, { extraction, placement: { page: 1, y: 0, afterId: original } })
  await expect.poll(() => page.evaluate(() => window.__tables.plans)).toBe(1)
  const c = await cell(page)
  await c.dblclick()
  const input = page.locator('.table-cell-input')
  await input.fill('手动调整')
  await input.press('Enter')
  await expect.poll(() => page.evaluate(() => window.__tables.replies.length)).toBe(1)
  const reply = await page.evaluate(() => window.__tables.replies[0])
  expect(reply.error).toBeUndefined()
  expect(await page.evaluate(() => window.__tables.plans)).toBe(2)
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(3)
  expect(await page.evaluate(() => window.__tables.payload.tables[0].cells[0].text)).toBe(
    '手动调整',
  )
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(1)
  expect(await page.evaluate(() => window.__tables.payload.tables[0].cells[0].text)).toBe(
    '手动调整',
  )
  expect(errors).toEqual([])
})
test('live revision mismatch and invalid topology preserve the document; full page spills', async ({
  page,
}) => {
  const errors = await setup(page)
  await send(page, { extraction, expectedRevision: -1 }, 'stale')
  await expect.poll(() => page.evaluate(() => window.__tables.replies.length)).toBe(1)
  expect(await page.evaluate(() => window.__tables.replies[0].error)).toContain('文档已修改')
  const broken = structuredClone(extraction)
  broken.tables[0].cells[0].colSpan = 2
  await send(page, { extraction: broken }, 'broken')
  await expect.poll(() => page.evaluate(() => window.__tables.replies.length)).toBe(2)
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  await send(page, { extraction, placement: { page: 1, y: 238 } }, 'next-page')
  await expect.poll(() => page.evaluate(() => window.__tables.replies.length)).toBe(3)
  expect(await page.evaluate(() => window.__tables.replies[2].result.plan.pages)).toEqual([2])
  await expect.poll(() => page.evaluate(() => window.__tables.document.pages.length)).toBe(2)
  await page.screenshot({ path: '/tmp/bambuscribe-import-next-page.png' })
  expect(errors).toEqual([])
})

test('cancel during recognition does not insert objects', async ({ page }) => {
  const errors = await setup(page)
  await page.evaluate(() => {
    window.go.application.App.RecognizeTables = async () => {
      await new Promise((r) => setTimeout(r, 500))
      return window.__cancelExtraction
    }
  })
  await page.evaluate((e) => {
    window.__cancelExtraction = e
  }, extraction)
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入表格', exact: true }).click()
  await page
    .getByLabel('表格图片文件')
    .setInputFiles({ name: 'table.png', mimeType: 'image/png', buffer: readFileSync(imagePath) })
  await page.getByRole('button', { name: '识别并预览' }).click()
  await page.getByRole('button', { name: '取消识别' }).click()
  await expect(page.getByRole('button', { name: '识别并预览' })).toBeEnabled()
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  expect(errors).toEqual([])
})

const png = (name) => ({ name, mimeType: 'image/png', buffer: readFileSync(imagePath) })
async function readyPreview(page) {
  await expect(page.getByRole('button', { name: '全部插入' })).toBeEnabled()
}
async function field(page, name, value) {
  await page.getByLabel(name, { exact: true }).fill(String(value))
  await page.getByLabel(name, { exact: true }).press('Tab')
  await readyPreview(page)
}
test('multiple images preview side by side, edit, drag on paper and undo atomically', async ({
  page,
}) => {
  const errors = await setup(page)
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入表格', exact: true }).click()
  await page.getByLabel('表格图片文件').setInputFiles([png('first.png'), png('second.png')])
  await expect(page.locator('[data-source]')).toHaveCount(2)
  await page.getByRole('button', { name: '识别并预览' }).click()
  await readyPreview(page)
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  await field(page, '预览宽度', 75)
  await page.getByLabel('识别表名').fill('第一张')
  await expect.poll(() => page.locator('[data-import-group="0"] path').count()).toBeGreaterThan(0)
  await readyPreview(page)
  await page.getByLabel('核对表格').selectOption('1')
  await field(page, '预览宽度', 75)
  const paper = await page.locator('.import-paper').first().boundingBox()
  const group = await page.locator('[data-import-group="1"]').boundingBox()
  const first = await page.locator('[data-import-group="0"]').boundingBox()
  const k = paper.width / 163
  await page.mouse.move(group.x + 20, group.y + 4)
  await page.mouse.down()
  await page.mouse.move(group.x + 20 + 85 * k, group.y + 4 + (first.y - group.y), { steps: 12 })
  await page.mouse.up()
  await readyPreview(page)
  expect(Number(await page.getByLabel('预览X', { exact: true }).inputValue())).toBeCloseTo(85, 0)
  await page.screenshot({ path: `/tmp/bambuscribe-batch-preview-${test.info().project.name}.png` })
  await page.getByRole('button', { name: '全部插入' }).click()
  await expect(page.getByRole('dialog')).not.toBeVisible()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(4)
  const imported = await page.evaluate(() => window.__tables.payload.tables)
  expect(imported[0].cells[0].text).toBe('第一张')
  expect(imported[2].x).toBeCloseTo(85, 0)
  expect(imported[2].cells[0].text).toBe(extraction.tables[0].title)
  expect(imported[2].groupId).toBe(imported[3].groupId)
  const id = imported[3].id
  await page.locator(`[data-table="${id}"] [data-table-cell]`).first().click()
  const handle = page
    .locator(`[data-table="${id}"]`)
    .getByRole('button', { name: '整组移动', exact: true })
  const box = await handle.boundingBox()
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await page.mouse.down()
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2 + 70, { steps: 10 })
  await page.mouse.up()
  await expect
    .poll(() =>
      page.evaluate((id) => window.__tables.payload.tables.find((t) => t.id === id).y, id),
    )
    .toBeGreaterThan(imported[3].y + 5)
  const moved = await page.evaluate(() => window.__tables.payload.tables)
  expect(moved[2].y - imported[2].y).toBeCloseTo(moved[3].y - imported[3].y, 4)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect
    .poll(() =>
      page.evaluate((id) => window.__tables.payload.tables.find((t) => t.id === id).y, id),
    )
    .toBeCloseTo(imported[3].y, 4)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(4)
  expect(errors).toEqual([])
})
test('batch retries only failed image, permits overlap and still rejects page overflow', async ({
  page,
}) => {
  const errors = await setup(page)
  await page.evaluate((e) => {
    window.__tables.calls = 0
    window.go.application.App.RecognizeTables = async () => {
      window.__tables.calls++
      if (window.__tables.calls === 2) throw new Error('识别超时，请重试')
      return e
    }
  }, extraction)
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入表格', exact: true }).click()
  await page.getByLabel('表格图片文件').setInputFiles([png('one.png'), png('two.png')])
  await page.getByRole('button', { name: '识别并预览' }).click()
  await expect(page.getByRole('button', { name: '重试并预览' })).toBeEnabled()
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  await expect(page.locator('[data-source="0"]')).toContainText('已识别')
  await page.getByRole('button', { name: '重试并预览' }).click()
  await readyPreview(page)
  expect(await page.evaluate(() => window.__tables.calls)).toBe(3)
  await page.getByLabel('核对表格').selectOption('1')
  await page.getByLabel('预览Y', { exact: true }).fill('0')
  await page.getByLabel('预览Y', { exact: true }).press('Tab')
  await readyPreview(page)
  await expect(page.getByRole('button', { name: '全部插入' })).toBeEnabled()
  await page.getByLabel('预览Y', { exact: true }).fill('239')
  await page.getByLabel('预览Y', { exact: true }).press('Tab')
  await expect(page.getByRole('alert')).toContainText('超出')
  await expect(page.getByRole('button', { name: '全部插入' })).toBeDisabled()
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  await page.getByLabel('预览Y', { exact: true }).fill('0')
  await page.getByLabel('预览Y', { exact: true }).press('Tab')
  await readyPreview(page)
  await page.getByRole('button', { name: '全部插入' }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(4)
  expect(errors).toEqual([])
})
test('repeated multi-image paste and drop append files without changing the document', async ({
  page,
}) => {
  const errors = await setup(page)
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入表格', exact: true }).click()
  const base64 = readFileSync(imagePath).toString('base64')
  for (const kind of ['paste', 'drop']) {
    await page.evaluate(
      ({ base64, kind }) => {
        const bytes = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0)),
          data = new DataTransfer()
        for (let i = 0; i < 2; i++)
          data.items.add(new File([bytes], `${kind}-${i}.png`, { type: 'image/png' }))
        const el = document.querySelector('.import-image')
        el.dispatchEvent(
          kind === 'paste'
            ? new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: data })
            : new DragEvent('drop', { bubbles: true, cancelable: true, dataTransfer: data }),
        )
      },
      { base64, kind },
    )
    await expect(page.locator('[data-source]')).toHaveCount(kind === 'paste' ? 2 : 4)
  }
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  await page.getByRole('button', { name: '移除图片 2' }).click()
  await expect(page.locator('[data-source]')).toHaveCount(3)
  expect(errors).toEqual([])
})

test('cancel second image retains first result and stale previews cannot insert', async ({
  page,
}) => {
  const errors = await setup(page)
  await page.evaluate((e) => {
    window.__tables.calls = 0
    window.go.application.App.RecognizeTables = async () => {
      window.__tables.calls++
      if (window.__tables.calls === 2) await new Promise((r) => setTimeout(r, 700))
      return structuredClone(e)
    }
  }, extraction)
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入表格', exact: true }).click()
  await page.getByLabel('表格图片文件').setInputFiles([png('one.png'), png('two.png')])
  await page.getByRole('button', { name: '识别并预览' }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.calls)).toBe(2)
  await page.getByRole('button', { name: '取消识别' }).click()
  await expect(page.getByRole('button', { name: '识别并预览' })).toBeEnabled()
  await expect(page.locator('[data-source="0"]')).toContainText('已识别')
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  await page.getByRole('button', { name: '识别并预览' }).click()
  await readyPreview(page)
  expect(await page.evaluate(() => window.__tables.calls)).toBe(3)
  await send(page, { extraction, placement: { page: 2, y: 0 } }, 'concurrent-edit')
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(2)
  await page.getByRole('button', { name: '全部插入' }).click()
  await expect(page.getByRole('alert')).toContainText('文档已修改')
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(2)
  await page.getByRole('button', { name: '重新预览' }).click()
  await readyPreview(page)
  await page.getByRole('button', { name: '全部插入' }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(6)
  expect(errors).toEqual([])
})

test('images run three at a time and out-of-order completion retains source order', async ({
  page,
}) => {
  const errors = await setup(page)
  await page.evaluate((e) => {
    Object.assign(window.__tables, { calls: 0, active: 0, peak: 0, finish: [] })
    window.go.application.App.RecognizeTables = async () => {
      const s = window.__tables,
        i = s.calls++
      s.active++
      s.peak = Math.max(s.peak, s.active)
      await new Promise((resolve) => {
        s.finish[i] = resolve
      })
      s.active--
      const result = structuredClone(e)
      result.tables[0].title = `图片 ${i + 1}`
      return result
    }
  }, extraction)
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入表格', exact: true }).click()
  await page
    .getByLabel('表格图片文件')
    .setInputFiles(Array.from({ length: 5 }, (_, i) => png(`${i + 1}.png`)))
  await page.getByRole('button', { name: '识别并预览' }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.calls)).toBe(3)
  await expect(page.locator('[data-source] small').filter({ hasText: '识别中' })).toHaveCount(3)
  await expect(
    page.getByRole('dialog', { name: '智能导入表格' }).getByRole('status'),
  ).toContainText('正在识别 3 张')
  await page.screenshot({ path: `/tmp/bambuscribe-parallel-${test.info().project.name}.png` })
  await page.evaluate(() => window.__tables.finish[1]())
  await expect.poll(() => page.evaluate(() => window.__tables.calls)).toBe(4)
  await page.evaluate(() => window.__tables.finish[3]())
  await expect.poll(() => page.evaluate(() => window.__tables.calls)).toBe(5)
  await page.evaluate(() => {
    window.__tables.finish[4]()
    window.__tables.finish[2]()
    window.__tables.finish[0]()
  })
  await readyPreview(page)
  expect(await page.evaluate(() => window.__tables.peak)).toBe(3)
  expect(
    (await page.getByLabel('核对表格').locator('option').allTextContents()).map((s) => s.trim()),
  ).toEqual([
    '1.png · 图片 1',
    '2.png · 图片 2',
    '3.png · 图片 3',
    '4.png · 图片 4',
    '5.png · 图片 5',
  ])
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  await page.getByRole('button', { name: '全部插入' }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(10)
  expect(
    await page.evaluate(() =>
      window.__tables.payload.tables
        .filter((t) => t.kind === 'textbox')
        .map((t) => t.cells[0].text),
    ),
  ).toEqual(['图片1', '图片2', '图片3', '图片4', '图片5'])
  expect(errors).toEqual([])
})

test('batch cancellation stops all running images and never starts queued images', async ({
  page,
}) => {
  const errors = await setup(page)
  await page.evaluate(() => {
    Object.assign(window.__tables, { calls: 0, aborts: [], cancelledRequests: 0 })
    window.go.application.App.RecognizeTables = async () => {
      window.__tables.calls++
      return new Promise((_resolve, reject) =>
        window.__tables.aborts.push(() => {
          window.__tables.cancelledRequests++
          reject(new Error('已取消'))
        }),
      )
    }
    window.go.application.App.CancelTableRecognition = async () =>
      window.__tables.aborts.splice(0).forEach((fn) => fn())
  })
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入表格', exact: true }).click()
  await page
    .getByLabel('表格图片文件')
    .setInputFiles(Array.from({ length: 5 }, (_, i) => png(`${i}.png`)))
  await page.getByRole('button', { name: '识别并预览' }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.calls)).toBe(3)
  await page.getByRole('button', { name: '取消识别' }).click()
  await expect(page.getByRole('button', { name: '识别并预览' })).toBeEnabled()
  expect(
    await page.evaluate(() => ({
      calls: window.__tables.calls,
      cancelled: window.__tables.cancelledRequests,
    })),
  ).toEqual({ calls: 3, cancelled: 3 })
  await expect(page.locator('[data-source] small').filter({ hasText: '待识别' })).toHaveCount(5)
  expect(await page.evaluate(() => window.__tables.payload.tables.length)).toBe(0)
  expect(errors).toEqual([])
})

test('lens and slit tables retain answer blanks, one-line caption and 70 percent minimum', async ({
  page,
}) => {
  const errors = await setup(page)
  const recognized = JSON.parse(
    readFileSync(
      new URL('../../../internal/tableimport/testdata/lens-and-slit.json', import.meta.url),
      'utf8',
    ),
  )
  await page.evaluate((e) => {
    window.go.application.App.RecognizeTables = async () => e
  }, recognized)
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入表格', exact: true }).click()
  await page.getByLabel('表格图片文件').setInputFiles(png('lens-and-slit.png'))
  await page.getByRole('button', { name: '识别并预览' }).click()
  await readyPreview(page)
  await expect(page.getByLabel('识别表名')).toHaveValue('表2 Δs=0.00 mm, Δe=35.00 mm')
  await page.screenshot({
    path: `/tmp/bambuscribe-readable-tables-${test.info().project.name}.png`,
  })
  await page.getByRole('button', { name: '全部插入' }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.tables.length)).toBe(4)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document.pages.flatMap((p) => p.tables).length))
    .toBe(4)
  const result = await page.evaluate(() => ({
    body: window.__tables.payload.settings.fontSize,
    tables: window.__tables.document.pages.flatMap((p) => p.tables),
  }))
  const cells = result.tables.flatMap((t) => t.cells)
  expect(result.tables[0].cells[0].text).not.toContain('\n')
  expect(cells.filter((c) => c.text.includes('        mm'))).toHaveLength(3)
  for (const c of cells.filter((c) => c.text.trim()))
    expect(c.fontSize).toBeGreaterThanOrEqual(result.body * 0.7 - 1e-6)
  const root = cells.find((c) => c.text.includes('√'))
  expect(root.missing).toEqual([])
  expect(
    root.strokes.some(
      (s) => s.length === 2 && s[0].y === s[1].y && s[1].x - s[0].x > root.fontSize,
    ),
  ).toBe(true)
  await page.screenshot({
    path: `/tmp/bambuscribe-readable-tables-inserted-${test.info().project.name}.png`,
  })
  expect(errors).toEqual([])
})
