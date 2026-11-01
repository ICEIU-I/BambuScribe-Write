import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'
async function moduleURL(name, replacements = {}) {
  let source = ts.transpileModule(
    await readFile(new URL(`../src/state/${name}.ts`, import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
  ).outputText
  for (const [name, url] of Object.entries(replacements))
    source = source.replace(`'${name}'`, JSON.stringify(url))
  return `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
}
const fontSizing = await moduleURL('fontSizing')
const { defaultFontPercent, fontPercent, fontMillimeters, updateLayoutSettings } = await import(
  fontSizing
)
const defaults = await moduleURL('defaults', { './fontSizing': fontSizing })
const { defaultLayout, defaultPreferences } = await import(defaults)
const { normalizePreferences } = await import(
  await moduleURL('normalizePreferences', {
    './defaults': defaults,
    './fontSizing': fontSizing,
    './inkCuts': await moduleURL('inkCuts'),
  })
)
test('paper erasures survive draft/project normalisation while legacy drafts default to none', () => {
  const erasure = {
    id: 'e',
    page: 0,
    x: 10,
    y: 20,
    width: 1,
    height: 2,
    path: [
      { x: 10, y: 20 },
      { x: 11, y: 22 },
    ],
    radius: 1.8,
  }
  assert.deepEqual(normalizePreferences({ erasures: [erasure] }).erasures, [erasure])
  assert.deepEqual(normalizePreferences({}).erasures, [])
  assert.deepEqual(
    normalizePreferences({ erasures: [{ ...erasure, path: [{ x: '<svg>', y: 1 }] }] }).erasures,
    [],
  )
})
const { wheelZoom, clampZoom } = await import(await moduleURL('canvasZoom'))
const { displayRegions } = await import(await moduleURL('imageEditing'))
const { writingOriginRange } = await import(await moduleURL('writingOrigin'))
const { hitCaret, caretAt, rowsFor } = await import(await moduleURL('caretGeometry'))
test('clicking either side of a wrapped fragment keeps the chosen caret position', () => {
  const document = {
    settings: { fontSize: 6, marginLeft: 0, marginTop: 0 },
    pages: [
      {
        lineHeight: 15,
        carets: [
          { index: 0, x: 0, y: 0 },
          { index: 1, x: 6, y: 0 },
          { index: 1, x: 50, y: 0 },
          { index: 2, x: 56, y: 0 },
        ],
      },
    ],
  }
  for (const x of [6, 50]) {
    const hit = hitCaret(document, 0, x, 3)
    assert.equal(hit.index, 1)
    assert.equal(caretAt(document, hit.index, hit).x, x)
  }
  assert.equal(rowsFor(document).length, 1)
})
test('110 percent type selects the clicked row when its em is larger than row pitch', () => {
  const document = {
    settings: { fontSize: 16.5, marginLeft: 0, marginTop: 0 },
    pages: [
      {
        lineHeight: 15,
        carets: [
          { index: 0, x: 0, y: 0 },
          { index: 1, x: 0, y: 15 },
        ],
      },
    ],
  }
  assert.equal(hitCaret(document, 0, 0, 15.5).index, 1)
})
test('physical start range covers all pages and prevents the Y240 bottom clipping', () => {
  const document = {
    settings: { marginLeft: 25.4, marginTop: 25.4 },
    pages: [
      {
        strokes: [
          [
            { x: 25.4, y: 25.4 },
            { x: 125.4, y: 269.078782 },
          ],
        ],
      },
      { strokes: [[{ x: 135.4, y: 270.078782 }]] },
    ],
  }
  const writing = defaultPreferences().writing
  const range = writingOriginRange(document, writing)
  assert.deepEqual(range, { x: [0, 146], y: [244.7, 256] })
  assert.ok(240 < range.y[0])
  assert.ok(250 >= range.y[0] && 250 <= range.y[1])
  assert.deepEqual(document.settings, { marginLeft: 25.4, marginTop: 25.4 })
  assert.deepEqual(writing, defaultPreferences().writing)
})
test('physical start range includes offsets, reverse travel and the start/park', () => {
  const document = {
    settings: { marginLeft: 10, marginTop: 20 },
    pages: [
      {
        strokes: [
          [
            { x: 5, y: 15 },
            { x: 150, y: 220 },
          ],
        ],
      },
    ],
  }
  const writing = { ...defaultPreferences().writing, offsetX: 2, offsetY: 3 }
  assert.deepEqual(writingOriginRange(document, writing), {
    x: [3, 114],
    y: [203, 254],
  })
  assert.equal(writingOriginRange(document, { ...writing, originMode: 'manual' }), null)
  assert.equal(writingOriginRange(null, writing), null)
  assert.equal(writingOriginRange({ ...document, pages: [{ strokes: [] }] }, writing), null)
  assert.equal(writingOriginRange(document, { ...writing, offsetY: NaN }), null)
  assert.equal(writingOriginRange(document, { ...writing, offsetY: 100 }), null)
})
test('default font size is 110 percent and writing calibration is unchanged', () => {
  const layout = defaultLayout()
  assert.equal(defaultFontPercent, 110)
  assert.equal(fontPercent(layout.fontSize, layout.lineHeight), 110)
  assert.equal(layout.fontSize, 16.5)
  assert.equal(layout.lineCount, 16)
  assert.deepEqual(defaultPreferences().writing, {
    offsetX: 0,
    offsetY: 0,
    penUp: 7,
    penDown: 3.4,
    originMode: 'auto',
    sound: true,
  })
})
test('percentage sizing retains scale across row and paper changes without changing row count', () => {
  const layout = defaultLayout()
  const rows = updateLayoutSettings(layout, { lineCount: 30 })
  assert.equal(rows.fontSize, 8.8)
  assert.equal(rows.lineCount, 30)
  const margins = updateLayoutSettings(rows, { marginTop: 10, marginBottom: 20 })
  assert.ok(Math.abs(margins.fontSize - 7.7) < 1e-10)
  assert.ok(Math.abs(fontPercent(margins.fontSize, margins.lineHeight) - 110) < 1e-10)
  const size = updateLayoutSettings(margins, { fontSize: fontMillimeters(80, margins.lineHeight) })
  assert.equal(size.lineCount, 30)
  assert.equal(size.fontSize, 5.6)
})
test('legacy millimeter font size remains unchanged while missing size uses 110 percent', () => {
  const saved = {
    lineMode: 'fixed',
    lineCount: 30,
    paperWidth: 210,
    paperHeight: 297,
    marginTop: 25.4,
    marginBottom: 25.4,
    fontSize: 7,
  }
  const restored = normalizePreferences({ layout: saved }).layout
  assert.equal(restored.fontSize, 7)
  assert.equal(saved.fontSize, 7)
  assert.ok(
    Math.abs(fontPercent(restored.fontSize, restored.lineHeight) - 85.29650690495532) < 1e-10,
  )
  const { fontSize, ...withoutSize } = saved
  const fresh = normalizePreferences({ layout: withoutSize }).layout
  assert.ok(Math.abs(fresh.fontSize - 9.027333333333333) < 1e-10)
})
test('spacing changes preserve glyph size and legacy projects keep their original spacing', () => {
  const layout = defaultLayout()
  const compact = updateLayoutSettings(layout, { letterSpacing: 85, lineSpacing: 85 })
  assert.equal(compact.fontSize, layout.fontSize)
  assert.equal(compact.lineHeight, layout.lineHeight)
  assert.equal(compact.lineCount, layout.lineCount)
  assert.equal(fontPercent(compact.fontSize, compact.lineHeight), 110)
  assert.equal(compact.letterSpacing, 85)
  assert.equal(compact.lineSpacing, 85)
  const { letterSpacing, lineSpacing, ...legacy } = compact
  const restored = normalizePreferences({ layout: legacy }).layout
  assert.equal(restored.letterSpacing, 100)
  assert.equal(restored.lineSpacing, 100)
  assert.equal(restored.fontSize, compact.fontSize)
  assert.equal(normalizePreferences({ layout: compact }).layout.lineSpacing, 85)
})
test('refreshed OCR geometry keeps current manual edits and excludes deleted objects', () => {
  const old = {
    id: 'old-ocr',
    source: 'ocr',
    mode: 'handwriting',
    x: 0.1,
    y: 0.1,
    width: 0.7,
    height: 0.2,
  }
  const fresh = { ...old, id: 'ocr-v1-1', mode: 'keep', ocrVersion: 1, width: 0.05 }
  const manual = { ...old, id: 'manual', source: 'manual', x: 0.4 }
  const erased = { ...manual, id: 'erased', mode: 'erase' }
  const added = { ...manual, id: 'added' }
  const regions = displayRegions(
    { textRegions: [old, { ...manual, x: 0.6 }, added] },
    { textRegions: [fresh, manual, erased] },
  )
  assert.deepEqual(
    regions.map((r) => r.id),
    ['ocr-v1-1', 'manual', 'added'],
  )
  assert.equal(regions[0].width, 0.05)
  assert.equal(regions[0].ocrVersion, 1)
  assert.equal(regions[1].x, 0.6)
  assert.equal(regions[1].sourceBounds, undefined)
})
test('legacy A4 draft keeps content, margins and inferred rows', () => {
  const p = normalizePreferences({
    text: '测试正文',
    layout: {
      paperWidth: 210,
      paperHeight: 297,
      marginTop: 25.4,
      marginBottom: 25.4,
      marginLeft: 25.4,
      marginRight: 25.4,
      lineHeight: 10,
    },
  })
  assert.equal(p.text, '测试正文')
  assert.equal(p.layout.paperWidth, 210)
  assert.equal(p.layout.lineCount, 24)
  assert.ok(Number.isFinite(p.layout.lineHeight))
  assert.equal(p.layout.templateName, '新建笔记')
})
test('explicit settings are preserved for backend validation', () => {
  const p = normalizePreferences({
    layout: { lineMode: 'random', lineCount: 12, paperWidth: 163, paperHeight: 240 },
    writing: { penUp: null, penDown: null },
  })
  assert.equal(p.layout.lineCount, 12)
  assert.equal(p.layout.lineMode, 'random')
  assert.equal(p.writing.penUp, null)
  assert.equal(
    normalizePreferences({ layout: { lineMode: 'fixed', lineCount: 0 } }).layout.lineCount,
    0,
  )
})
test('small trackpad deltas accumulate; zoom stays bounded in all wheel modes', () => {
  let zoom = 65
  for (let i = 0; i < 100; i++) zoom = wheelZoom(zoom, -0.1, 0, 600)
  assert.ok(zoom > 70)
  assert.equal(clampZoom(200), 150)
  assert.equal(clampZoom(10), 30)
  assert.equal(wheelZoom(65, 1, 1, 600), wheelZoom(65, 16, 0, 600))
  assert.equal(wheelZoom(65, 0.1, 2, 600), wheelZoom(65, 60, 0, 600))
})

const { outputAvailability } = await import(await moduleURL('outputAvailability'))
const { defaultUploadPrefix, validUploadPrefix, uploadPageSuffix } = await import(
  await moduleURL('uploadNaming')
)
test('upload filenames use the first-line number and preserve automatic page suffixes', () => {
  assert.equal(defaultUploadPrefix('实验三十一 普朗克常量\n1. 目的'), '31')
  assert.equal(defaultUploadPrefix('实验 31 普朗克常量\n99'), '31')
  assert.equal(defaultUploadPrefix('实验三十一 2026'), '31')
  assert.equal(defaultUploadPrefix('实验二〇二六'), '2026')
  assert.equal(defaultUploadPrefix('实验一百零一'), '101')
  assert.equal(defaultUploadPrefix('\n笔记标题\n31'), '笔记标题')
  assert.equal(defaultUploadPrefix('', '未命名'), '未命名')
  assert.equal(validUploadPrefix('../31'), false)
  assert.equal(validUploadPrefix(''), false)
  assert.equal(validUploadPrefix('a'.repeat(65)), false)
  assert.equal(validUploadPrefix('实验31'), true)
  assert.equal(uploadPageSuffix([1, 2, 3, 4]), '-page1~page4')
  assert.equal(uploadPageSuffix([3]), '-page3')
})
test('export tells the missing page and does not block a valid current page', () => {
  const pages = [
    {
      number: 1,
      strokes: [
        [
          { x: 1, y: 1 },
          { x: 2, y: 2 },
        ],
      ],
      missing: [],
    },
    { number: 2, strokes: [], missing: [{ char: '🙂' }] },
  ]
  const state = {
    document: { pages },
    index: 0,
    ready: true,
    loading: false,
    dirty: false,
    busy: false,
  }
  const result = outputAvailability(state)
  assert.equal(result.current, '')
  assert.match(result.all, /第 2 页缺字：🙂/)
  assert.match(outputAvailability({ ...state, index: 1 }).current, /第 2 页缺字/)
})
test('all-page export skips empty current page, but never stale or empty documents', () => {
  const pages = [
    { number: 1, strokes: [], missing: [] },
    { number: 2, strokes: [[{}]], missing: [] },
  ]
  const state = {
    document: { pages },
    index: 0,
    ready: true,
    loading: false,
    dirty: false,
    busy: false,
  }
  assert.match(outputAvailability(state).current, /没有可书写内容/)
  assert.equal(outputAvailability(state).all, '')
  assert.match(outputAvailability({ ...state, dirty: true }).all, /排版尚未完成/)
  assert.match(outputAvailability({ ...state, document: { pages: [] } }).all, /没有可书写内容/)
})

const { copyTextBox, pasteTextBox, readTextBox } = await import(await moduleURL('textBoxClipboard'))
const { tidySpacing, insertedRange } = await import(await moduleURL('pasteTidy'))
test('textbox clipboard keeps physical size across images and clamps the paste inside the target', () => {
  const source = { width: 100, height: 50 }
  const region = { text: 'W/m²', x: 0.3, y: 0.2, width: 0.2, height: 0.1 }
  const copied = copyTextBox(source, region)
  const target = { width: 200, height: 100 }
  const pasted = pasteTextBox(copied, target, { x: 0.99, y: 0.99 })
  assert.equal(pasted.text, 'W/m²')
  assert.equal(pasted.width * target.width, 20)
  assert.equal(pasted.height * target.height, 5)
  assert.ok(pasted.x + pasted.width <= 1)
  assert.ok(pasted.y + pasted.height <= 1)
  assert.notEqual(pasted.id, pasteTextBox(copied, target).id)
  assert.equal(region.x, 0.3)
})
const { projectPayload, projectPreferences, projectVersion } = await import(
  await moduleURL('projectFile', {
    './normalizePreferences': await moduleURL('normalizePreferences', {
      './inkCuts': await moduleURL('inkCuts'),
      './defaults': defaults,
      './fontSizing': fontSizing,
    }),
  })
)
const workspace = normalizePreferences({ theme: 'dark' })
test('project file round-trips the document and leaves the workspace theme out of it', () => {
  const project = projectPayload({
    text: '第一行\n第二行',
    layout: { ...workspace.layout, templateName: '会议记录', lineCount: 12 },
    writing: { ...workspace.writing, sound: false, originMode: 'manual' },
    images: [
      {
        id: 'diagram',
        page: 1,
        data: 'data:image/png;base64,AAAA',
        x: 10,
        y: 20,
        width: 30,
        height: 40,
        rotation: 15,
        mode: 'line',
        textRegions: [
          {
            id: 'r1',
            text: '',
            x: 0.1,
            y: 0.1,
            width: 0.2,
            height: 0.2,
            mode: 'erase',
            source: 'manual',
            erasePath: [{ x: 0.1, y: 0.1 }],
            eraseRadius: 0.04,
          },
        ],
      },
    ],
  })
  assert.equal(project.version, projectVersion)
  assert.equal('theme' in project, false)
  const restored = projectPreferences(workspace, project)
  assert.equal(restored.theme, 'dark')
  assert.equal(restored.text, '第一行\n第二行')
  assert.equal(restored.layout.templateName, '会议记录')
  assert.equal(restored.layout.lineCount, 12)
  assert.equal(restored.writing.sound, false)
  assert.equal(restored.writing.originMode, 'manual')
  assert.deepEqual(restored.images, project.images)
})
test('a project missing newer fields still opens with editor defaults', () => {
  const restored = projectPreferences(workspace, {
    version: projectVersion,
    text: '旧文件',
    layout: { paperWidth: 163, paperHeight: 240, lineHeight: 15 },
    images: [],
  })
  assert.equal(restored.text, '旧文件')
  assert.equal(restored.layout.templateName, '新建笔记')
  assert.equal(restored.layout.lineCount, 16)
  assert.equal(restored.images.length, 0)
  assert.equal(restored.writing.originMode, 'auto')
})
test('flattened block boundaries become line breaks, single spaces stay', () => {
  const pasted = '实验三十一 普朗克常量的测定   一、实验目的   1.通过实验了解光的量子性.'
  const tidied = tidySpacing(pasted)
  assert.equal(tidied.text, '实验三十一 普朗克常量的测定\n一、实验目的\n1.通过实验了解光的量子性.')
  assert.equal(tidied.count, 2)
  // A run of tabs left by a word processor reads the same way.
  assert.equal(tidySpacing('图1\t\t图2').text, '图1\n图2')
  // Nothing to do: text the clipboard already delivered with line breaks.
  assert.deepEqual(tidySpacing('第一行\n第二行'), { text: '第一行\n第二行', count: 0 })
  // Indentation and trailing space are the writer's, not a lost boundary: a run
  // that opens or closes a line stays where it is.
  assert.deepEqual(tidySpacing('        实验三十一'), { text: '        实验三十一', count: 0 })
  assert.deepEqual(tidySpacing('行一\n            子行'), {
    text: '行一\n            子行',
    count: 0,
  })
  assert.deepEqual(tidySpacing('末尾   '), { text: '末尾   ', count: 0 })
})
test('the pasted range is what changed between the two texts', () => {
  assert.deepEqual(insertedRange('开头结尾', '开头中间结尾'), [2, 4])
  // Deleting a selection inserts nothing, so the range is empty rather than
  // the whole text — a replacement that shares characters is partitioned
  // between prefix and suffix, and only what is left is new.
  assert.deepEqual(insertedRange('开头旧文结尾', '开头结尾'), [2, 2])
  assert.deepEqual(insertedRange('', '整段新文字'), [0, 5])
  assert.deepEqual(insertedRange('没有变化', '没有变化'), [4, 4])
})
test('unrelated clipboard text cannot paste a previous copied textbox', () => {
  const copied = copyTextBox(
    { width: 100, height: 50 },
    { text: 'W/m²', x: 0.3, y: 0.2, width: 0.2, height: 0.1 },
  )
  assert.equal(
    readTextBox({ getData: (type) => (type === 'text/plain' ? 'other text' : '') }, copied),
    undefined,
  )
  assert.deepEqual(
    readTextBox({ getData: (type) => (type === 'text/plain' ? 'W/m²' : '') }, copied),
    copied,
  )
})
