import { expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { test, open } from './tableHarness.mjs'
const scene = JSON.parse(
  readFileSync(new URL('../../../docs/diagram-fit-2026-10-09/first.json', import.meta.url)),
)
const png = readFileSync(
  new URL('../../../internal/engine/testdata/hall-switch.png', import.meta.url),
)

async function setup(page, delayed = false) {
  const errors = await open(page)
  await page.evaluate(
    ({ scene, delayed }) => {
      window.__diagram = { calls: 0, cancels: 0, active: 0, peak: 0, requests: [], delayed, scene }
      Object.assign(window.go.application.App, {
        AgentConfig: async () => ({
          baseURL: 'https://example.invalid/v1',
          model: 'gpt-6.1-sol',
          protocol: 'responses',
          hasKey: true,
        }),
        SaveAgentConfig: async () => {},
        RecognizeDiagram: async (source, instruction) => {
          const state = window.__diagram
          state.calls++
          state.active++
          state.peak = Math.max(state.peak, state.active)
          const request = { source, instruction }
          state.requests.push(request)
          try {
            const result = state.delayed
              ? await new Promise((resolve, reject) => {
                  Object.assign(request, { resolve, reject })
                  Object.assign(state, { resolve, reject })
                })
              : null
            return result ?? state.scene
          } finally {
            state.active--
          }
        },
        CancelDiagramRecognition: async () => {
          window.__diagram.cancels++
        },
      })
    },
    { scene, delayed },
  )
  return errors
}
async function menu(page) {
  await page.getByRole('button', { name: '智能导入', exact: true }).click()
  await page.getByRole('button', { name: '导入图像', exact: true }).click()
}
async function start(page) {
  await menu(page)
  await page
    .getByLabel('图像文件', { exact: true })
    .setInputFiles({ name: 'diagram.png', mimeType: 'image/png', buffer: png })
  await page.getByRole('button', { name: '识别并插入', exact: true }).click()
}
async function settled(page) {
  await expect(page.locator('[data-reorder-enabled]').first()).toHaveAttribute(
    'data-reorder-enabled',
    'true',
  )
}
test('inserting at a blank line keeps the image there and pushes following body below it', async ({
  page,
}) => {
  const errors = await setup(page)
  const text = '上方正文\n\n\n四、实验内容及步骤\n' + '下方正文保持完整。\n'.repeat(20)
  await page.evaluate((text) => {
    window.__tables.saved = { ...window.__tables.preferences, text }
    window.__tables.menu['menu:action']('open')
    window.__diagram.scene = {
      width: 100,
      height: 60,
      strokes: [
        {
          id: 'line',
          points: [
            { x: 10, y: 10 },
            { x: 90, y: 10 },
          ],
        },
      ],
      circles: [],
      labels: [],
      uncertainties: [],
    }
  }, text)
  const input = page.getByRole('textbox', { name: '书写内容', exact: true })
  await expect(input).toHaveValue(text)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.pages[0]?.text))
    .toContain('上方正文')
  await settled(page)
  const before = await page.evaluate(() => window.__tables.document)
  const paper = await page.locator('[data-paper="0"]').boundingBox()
  await page.mouse.click(paper.x + (100 * paper.width) / 163, paper.y + (19 * paper.width) / 163)
  await start(page)
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await settled(page)
  const after = await page.evaluate(() => window.__tables.document)
  const image = after.pages[0].images[0]
  expect(image.y).toBe(15)
  expect(after.pages.flatMap((p) => p.cells).filter((c) => c.start < 4)).toEqual(
    before.pages.flatMap((p) => p.cells).filter((c) => c.start < 4),
  )
  expect(after.pages[0].cells.find((c) => c.start === text.indexOf('四')).y).toBeGreaterThan(
    image.y + image.height,
  )
  expect(after.pages.map((p) => p.text).join('')).toBe(text)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  await settled(page)
  await expect.poll(() => page.evaluate(() => window.__tables.document?.id)).toBe(before.id)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await settled(page)
  await expect.poll(() => page.evaluate(() => window.__tables.document?.id)).toBe(after.id)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.images.length)).toBe(1)
  await page.evaluate(() => {
    window.__tables.document = null
    window.__tables.menu['menu:action']('open')
  })
  await expect.poll(() => page.evaluate(() => window.__tables.document?.id)).toBe(after.id)
  await settled(page)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => !!window.__tables.exported?.gcode)).toBe(true)
  await page.screenshot({
    path: `/tmp/bambuscribe-diagram-insertion-${test.info().project.name}.png`,
  })
  expect(errors).toEqual([])
})
test('two imported figures keep their order before following body and use the available paper', async ({
  page,
}) => {
  const errors = await setup(page, true)
  const text =
    '上方正文\n' + '\n'.repeat(16) + '四、实验内容及步骤\n' + '下方正文保持完整。\n'.repeat(20)
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
        fontSize: 6,
        lineCount: 30,
        lineSpacing: 85,
      },
    }
    window.__tables.menu['menu:action']('open')
  }, text)
  await expect(page.getByRole('textbox', { name: '书写内容', exact: true })).toHaveValue(text)
  await expect
    .poll(() => page.evaluate(() => window.__tables.document?.settings.paperWidth))
    .toBe(210)
  await settled(page)
  const paper = await page.locator('[data-paper="0"]').boundingBox()
  const scale = paper.width / 210
  await page.mouse.click(paper.x + 130 * scale, paper.y + 133 * scale)
  await startBatch(page, 2)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(2)
  await page.evaluate(() => {
    for (const [i, width, height] of [
      [0, 704, 384],
      [1, 632, 240],
    ]) {
      window.__diagram.requests[i].resolve({
        width,
        height,
        strokes: [
          {
            id: `line-${i}`,
            points: [
              { x: 10, y: 10 },
              { x: width - 10, y: 10 },
            ],
          },
        ],
        circles: [],
        labels: [],
        uncertainties: [],
      })
    }
  })
  await expect(page.locator('[data-image-id]')).toHaveCount(2)
  await settled(page)
  const doc = await page.evaluate(() => window.__tables.document)
  expect(doc.pages[0].images).toHaveLength(2)
  const [first, second] = doc.pages[0].images
  expect(first.y).toBeCloseTo(130.035, 5)
  expect(second.y).toBeCloseTo(first.y + first.height + 4, 5)
  expect(second.y + second.height).toBeGreaterThan(271.6)
  expect(second.y + second.height).toBeLessThanOrEqual(297)
  expect(doc.pages[0].text).not.toContain('四、实验内容及步骤')
  expect(doc.pages[1].text).toContain('四、实验内容及步骤')
  expect(doc.pages.map((p) => p.text).join('')).toBe(text)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  await settled(page)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(2)
  await settled(page)
  await expect.poll(() => page.evaluate(() => window.__tables.document?.id)).toBe(doc.id)
  await page.screenshot({
    path: `/tmp/bambuscribe-diagram-two-insertion-${test.info().project.name}.png`,
  })
  expect(errors).toEqual([])
})
test('one call inserts vector diagram directly, preserves resize/save/export and undo', async ({
  page,
}) => {
  const errors = await setup(page)
  await start(page)
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(1)
  expect(await page.evaluate(() => window.__tables.payload.images[0].diagram)).toEqual(scene)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await expect.poll(() => page.evaluate(() => window.__tables.payload.images[0]?.width)).toBe(150)
  const handle = page.locator('.image-resize-handle')
  const box = await handle.boundingBox()
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await page.mouse.down()
  await page.mouse.move(box.x - 100, box.y - 80, { steps: 8 })
  await page.mouse.up()
  await expect
    .poll(() => page.evaluate(() => window.__tables.payload.images[0].width))
    .toBeLessThan(150)
  const size = await page.evaluate(() => window.__tables.payload.images[0])
  expect(size.height / size.width).toBeCloseTo(scene.height / scene.width, 6)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect
    .poll(() => page.evaluate(() => window.__tables.saved?.images[0]?.diagram?.width))
    .toBe(1354)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => !!window.__tables.exported)).toBe(true)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.screenshot({ path: '/tmp/bambuscribe-diagram-agent-ui.png' })
  expect(errors).toEqual([])
})
test('explicit recognition cancellation prevents a late insert', async ({ page }) => {
  const errors = await setup(page, true)
  await start(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.getByRole('button', { name: '取消识别', exact: true }).click()
  await page.evaluate(() => window.__diagram.resolve())
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  expect(await page.evaluate(() => window.__diagram.cancels)).toBeGreaterThan(0)
  expect(errors).toEqual([])
})
test('project switch rejects an in-flight diagram result', async ({ page }) => {
  const errors = await setup(page, true)
  await start(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.getByRole('button', { name: '关闭图像导入', exact: true }).click()
  await expect(page.getByLabel('后台图像导入', { exact: true })).toBeVisible()
  await page.evaluate(() => window.__tables.menu['menu:action']('new'))
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  await page.evaluate(() => window.__diagram.resolve())
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  expect(await page.evaluate(() => window.__diagram.cancels)).toBe(1)
  await expect(page.getByLabel('后台图像导入', { exact: true })).not.toBeVisible()
  expect(errors).toEqual([])
})

test('closing and reopening keeps one background request and preserves ongoing edits', async ({
  page,
}) => {
  const errors = await setup(page, true)
  await page.clock.install()
  await start(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(1)
  const source = await page.getByAltText('待识别原图').getAttribute('src')
  await page.getByRole('button', { name: '关闭图像导入', exact: true }).click()
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  expect(await page.evaluate(() => window.__diagram.cancels)).toBe(0)
  await page.clock.fastForward(65000)
  await expect(page.getByRole('button', { name: '查看图像导入任务' })).toContainText('1:05')
  await page.locator('textarea.paper-input').fill('继续编辑')
  await menu(page)
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).toBeVisible()
  await expect(page.getByAltText('待识别原图')).toHaveAttribute('src', source)
  await expect(page.locator('.diagram-progress')).toContainText('1:05')
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  await page.screenshot({
    path: `/tmp/bambuscribe-diagram-background-${test.info().project.name}.png`,
  })
  await page.evaluate(() => window.__diagram.resolve())
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  await expect(page.getByLabel('后台图像导入', { exact: true })).not.toBeVisible()
  await expect(page.locator('textarea.paper-input')).toHaveValue('继续编辑')
  expect(await page.evaluate(() => window.__diagram.cancels)).toBe(0)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  await expect(page.locator('textarea.paper-input')).toHaveValue('继续编辑')
  expect(errors).toEqual([])
})

test('explicit background cancellation drops a late result', async ({ page }) => {
  const errors = await setup(page, true)
  await start(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.getByRole('button', { name: '关闭图像导入', exact: true }).click()
  await page.getByRole('button', { name: '取消识别', exact: true }).click()
  await page.evaluate(() => window.__diagram.resolve())
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  expect(await page.evaluate(() => window.__diagram.cancels)).toBe(1)
  await expect(page.getByLabel('后台图像导入', { exact: true })).not.toBeVisible()
  expect(errors).toEqual([])
})

test('background failures stay visible and can be reopened and retried', async ({ page }) => {
  const errors = await setup(page, true)
  await start(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(1)
  await page.getByRole('button', { name: '关闭图像导入', exact: true }).click()
  await page.evaluate(() => window.__diagram.reject(new Error('连接超时，请重试')))
  await expect(page.getByRole('alert')).toContainText('连接超时')
  await expect(page.getByRole('button', { name: '查看图像导入任务' })).toContainText('图像导入失败')
  await page.getByRole('button', { name: '查看图像导入任务' }).click()
  await expect(page.getByRole('dialog', { name: '智能导入图像' }).getByRole('alert')).toContainText(
    '连接超时',
  )
  await page.evaluate(() => {
    window.__diagram.delayed = false
  })
  await page.getByRole('button', { name: '识别并插入', exact: true }).click()
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(2)
  expect(errors).toEqual([])
})
test('invalid model geometry is blocked before document mutation', async ({ page }) => {
  const errors = await setup(page)
  await page.evaluate(() => {
    window.__diagram.scene.circles[0].r = 10000
  })
  await start(page)
  await expect(page.getByRole('alert').filter({ hasText: '图像圆形坐标无效' })).toBeVisible()
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(1)
  expect(errors).toEqual([])
})

async function startBatch(page, count = 5) {
  await menu(page)
  await page.getByLabel('图像文件', { exact: true }).setInputFiles(
    Array.from({ length: count }, (_, i) => ({
      name: `diagram-${i + 1}.png`,
      mimeType: 'image/png',
      buffer: png,
    })),
  )
  await expect(page.locator('[data-diagram-source]')).toHaveCount(count)
  await page.getByRole('button', { name: '识别并插入', exact: true }).click()
}
async function resolveSource(page, index) {
  await page.evaluate((i) => {
    const result = structuredClone(window.__diagram.scene)
    result.strokes[0].id = `source-${i + 1}`
    window.__diagram.requests[i].resolve(result)
  }, index)
}

test('batch runs three at a time, queues the rest and inserts in source order as one undo', async ({
  page,
}) => {
  const errors = await setup(page, true)
  await startBatch(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(3)
  await expect(page.locator('[data-diagram-source]').filter({ hasText: '识别中' })).toHaveCount(3)
  await expect(page.locator('[data-diagram-source]').filter({ hasText: '待识别' })).toHaveCount(2)
  await page.screenshot({ path: `/tmp/bambuscribe-diagram-batch-${test.info().project.name}.png` })
  await resolveSource(page, 1)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(4)
  await resolveSource(page, 0)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(5)
  for (const i of [4, 3, 2]) await resolveSource(page, i)
  await expect(page.getByRole('dialog', { name: '智能导入图像' })).not.toBeVisible()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.images.length)).toBe(5)
  const images = await page.evaluate(() => window.__tables.payload.images)
  expect(images.map((i) => i.diagram.strokes[0].id)).toEqual(
    Array.from({ length: 5 }, (_, i) => `source-${i + 1}`),
  )
  expect(new Set(images.map((i) => i.id)).size).toBe(5)
  expect(images.at(-1).page).toBeGreaterThan(0)
  const positions = await page.evaluate(() => window.__tables.document.objectLayout)
  const placed = images.map((im) => ({ ...im, ...positions.find((p) => p.id === im.id) }))
  for (let i = 1; i < placed.length; i++) {
    const a = placed[i - 1],
      b = placed[i]
    expect(b.page > a.page || b.y >= a.y + a.height + 3.99).toBe(true)
    expect(b.y + b.height).toBeLessThanOrEqual(240.001)
  }
  expect(await page.evaluate(() => window.__diagram.peak)).toBe(3)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.images.length)).toBe(0)
  await page.getByRole('button', { name: '重做', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.images.length)).toBe(5)
  await page.evaluate(() => window.__tables.menu['menu:action']('saveAs'))
  await expect.poll(() => page.evaluate(() => window.__tables.saved?.images.length)).toBe(5)
  await page.evaluate(() => window.__tables.menu['menu:action']('exportPage'))
  await expect.poll(() => page.evaluate(() => !!window.__tables.exported)).toBe(true)
  await page.evaluate(() => window.__tables.menu['menu:action']('open'))
  await expect.poll(() => page.evaluate(() => window.__tables.payload.images.length)).toBe(5)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(5)
  expect(errors).toEqual([])
})

test('batch failure preserves successful results and retry only recognizes the failed image', async ({
  page,
}) => {
  const errors = await setup(page, true)
  await startBatch(page, 3)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(3)
  await resolveSource(page, 0)
  await page.evaluate(() => window.__diagram.requests[1].reject(new Error('连接超时，请重试')))
  await resolveSource(page, 2)
  await expect(page.getByRole('alert')).toContainText('部分图片识别失败')
  await expect(page.locator('[data-diagram-source]').filter({ hasText: '已识别' })).toHaveCount(2)
  await expect(page.locator('[data-diagram-source="1"]')).toContainText('连接超时')
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  await page.getByRole('button', { name: '重试失败图片', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(4)
  await page.evaluate(() => {
    const result = structuredClone(window.__diagram.scene)
    result.strokes[0].id = 'retry-second'
    window.__diagram.requests[3].resolve(result)
  })
  await expect.poll(() => page.evaluate(() => window.__tables.payload.images.length)).toBe(3)
  expect(
    await page.evaluate(() => window.__tables.payload.images.map((i) => i.diagram.strokes[0].id)),
  ).toEqual(['source-1', 'retry-second', 'source-3'])
  expect(errors).toEqual([])
})

test('removing a failed image reuses the other results without another recognition', async ({
  page,
}) => {
  const errors = await setup(page, true)
  await startBatch(page, 3)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(3)
  await resolveSource(page, 0)
  await resolveSource(page, 2)
  await page.evaluate(() => window.__diagram.requests[1].reject(new Error('图片无效，请更换')))
  await expect(page.getByRole('alert')).toContainText('部分图片识别失败')
  await page.getByRole('button', { name: '移除图像 2', exact: true }).click()
  await page.getByRole('button', { name: '识别并插入', exact: true }).click()
  await expect.poll(() => page.evaluate(() => window.__tables.payload.images.length)).toBe(2)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(3)
  expect(errors).toEqual([])
})

test('batch cancellation stops queued images, drops all late results and allows a new batch', async ({
  page,
}) => {
  const errors = await setup(page, true)
  await startBatch(page)
  await expect.poll(() => page.evaluate(() => window.__diagram.calls)).toBe(3)
  await page.getByRole('button', { name: '关闭图像导入', exact: true }).click()
  await expect(page.getByRole('button', { name: '查看图像导入任务' })).toContainText('0/5 张')
  await page.getByRole('button', { name: '取消识别', exact: true }).click()
  for (const i of [2, 1, 0]) await resolveSource(page, i)
  await expect(page.locator('[data-image-id]')).toHaveCount(0)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(3)
  expect(await page.evaluate(() => window.__diagram.cancels)).toBe(1)
  await page.evaluate(() => {
    window.__diagram.delayed = false
  })
  await startBatch(page, 3)
  await expect.poll(() => page.evaluate(() => window.__tables.payload.images.length)).toBe(3)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(6)
  expect(errors).toEqual([])
})

test('multi-select, paste, drop, replacement and limits preserve the batch', async ({ page }) => {
  const errors = await setup(page)
  await menu(page)
  const file = { name: 'first.png', mimeType: 'image/png', buffer: png }
  await page.getByLabel('图像文件', { exact: true }).setInputFiles(Array(11).fill(file))
  await expect(page.getByRole('alert')).toContainText('最多 10 张')
  await expect(page.locator('[data-diagram-source]')).toHaveCount(0)
  await page.getByLabel('图像文件', { exact: true }).setInputFiles(file)
  for (const type of ['paste', 'drop']) {
    await page.evaluate(
      ({ type, base64 }) => {
        const data = new DataTransfer()
        const bytes = Uint8Array.from(atob(base64), (c) => c.charCodeAt(0))
        data.items.add(new File([bytes], `${type}.png`, { type: 'image/png' }))
        const source = document.querySelector('.diagram-source')
        const e =
          type === 'paste'
            ? new ClipboardEvent('paste', { bubbles: true, cancelable: true, clipboardData: data })
            : new DragEvent('drop', { bubbles: true, cancelable: true, dataTransfer: data })
        source.dispatchEvent(e)
      },
      { type, base64: png.toString('base64') },
    )
    await expect(page.locator('[data-diagram-source]')).toHaveCount(type === 'paste' ? 2 : 3)
  }
  await page
    .getByLabel('更换图像 2', { exact: true })
    .setInputFiles({ ...file, name: 'replaced.png' })
  await expect(page.locator('[data-diagram-source="1"]')).toContainText('replaced.png')
  await page.getByLabel('图像文件', { exact: true }).setInputFiles({
    name: 'bad.gif',
    mimeType: 'image/gif',
    buffer: png,
  })
  await expect(page.getByRole('alert')).toContainText('PNG 或 JPEG')
  await expect(page.locator('[data-diagram-source]')).toHaveCount(3)
  await page.getByRole('button', { name: '移除图像 1', exact: true }).click()
  await expect(page.locator('[data-diagram-source]')).toHaveCount(2)
  expect(await page.evaluate(() => window.__diagram.calls)).toBe(0)
  expect(errors).toEqual([])
})

test('re-recognizing a selected image preserves its identity and can undo the replacement', async ({
  page,
}) => {
  const errors = await setup(page)
  await start(page)
  await expect(page.locator('[data-image-id]')).toHaveCount(1)
  const original = await page.evaluate(() => window.__tables.payload.images[0])
  await menu(page)
  await expect(page.getByRole('heading', { name: '重新识别图像', exact: true })).toBeVisible()
  await page.evaluate(() => {
    window.__diagram.scene = structuredClone(window.__diagram.scene)
    window.__diagram.scene.strokes[0].id = 'rebuilt'
  })
  await page.getByRole('button', { name: '重新识别', exact: true }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.payload.images[0].diagram.strokes[0].id))
    .toBe('rebuilt')
  const rebuilt = await page.evaluate(() => window.__tables.payload.images)
  expect(rebuilt).toHaveLength(1)
  expect({ ...rebuilt[0], diagram: original.diagram }).toEqual(original)
  await page.getByRole('button', { name: '撤销', exact: true }).click()
  await expect
    .poll(() => page.evaluate(() => window.__tables.payload.images[0].diagram.strokes[0].id))
    .toBe(original.diagram.strokes[0].id)
  expect(errors).toEqual([])
})
