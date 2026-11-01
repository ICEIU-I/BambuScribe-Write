import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import assert from 'node:assert/strict'
import ts from 'typescript'
const modules = new Map()
async function moduleURL(name) {
  if (modules.has(name)) return modules.get(name)
  let source = ts.transpileModule(
    await readFile(new URL(`../src/state/${name}.ts`, import.meta.url), 'utf8'),
    { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } },
  ).outputText
  for (const [, dependency] of source.matchAll(/from ['"]\.\/([^'"]+)['"]/g))
    source = source.replace(`'./${dependency}'`, `'${await moduleURL(dependency)}'`)
  const url = `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
  modules.set(name, url)
  return url
}
const { formulaRanges, calculationAt, insertCalculation } = await import(
  await moduleURL('calculationText')
)
const { formulaSelection, formulaKey } = await import(await moduleURL('formulaSelection'))
const { parseFormulaFields, formulaFieldText } = await import(await moduleURL('formulaFields'))
const { tidySpacing } = await import(await moduleURL('pasteTidy'))
const { rulePages } = await import(await moduleURL('writingRules'))
test('pasted math retains argument spaces while prose paragraph gaps are repaired', () => {
  const input = String.raw`说明  内容\frac{a  b}{c  d}，\sqrt[3]{x  y}与x_{长  下标}。`
  assert.deepEqual(tidySpacing(input), {
    text: input.replace('说明  内容', '说明\n内容'),
    count: 1,
  })
  for (const text of [
    String.raw`\[x =  \frac{1}{2}\]`,
    String.raw`\(x =  2\)`,
    '√(a  b)',
    String.raw`\sqrt{a  b`,
  ])
    assert.deepEqual(tidySpacing(text), { text, count: 0 })
})
test('the in-app guide exposes each document category as a separate page', async () => {
  const markdown = await readFile(new URL('../../docs/writing-syntax.md', import.meta.url), 'utf8')
  const pages = rulePages(markdown)
  assert.deepEqual(
    pages.map((p) => p.title),
    ['基本规则', '上下标', '分式', '根号', '上横线', '独立公式', '符号与空格', '错误与兼容'],
  )
  assert.ok(pages.every((p) => p.blocks.some((b) => b.kind === 'code')))
})
test('formula selection and deletion cannot split source delimiters', () => {
  const text = '数据\n\\(x=\\frac{1}{2}\\)\n结尾'
  const [range] = formulaRanges(text)
  assert.equal(range.start, 3)
  assert.deepEqual(formulaSelection(text, range.start + 2, range.end - 2), {
    start: range.start,
    end: range.end,
  })
  assert.deepEqual(formulaKey(text, range.end, range.end, 'Backspace'), {
    start: range.start,
    end: range.end,
    remove: true,
  })
  assert.equal(formulaKey(text, 0, 0, 'Delete'), undefined)
  assert.equal(insertCalculation('甲乙', 1, 1, '公式'), '\n公式\n')
})
test('editable formula fields round trip nested fractions, scripts and bars', () => {
  const source = 'U̅_{H}=\\frac{\\sqrt{1+x^{2}}}{\\bar{B}_{1}}'
  const fields = parseFormulaFields(source)
  assert.equal(formulaFieldText(fields), source)
  const frac = fields.find((f) => f.kind === 'fraction')
  frac.denominator = [{ kind: 'text', text: '242×5.00' }]
  assert.match(formulaFieldText(fields), /\{242×5\.00\}/)
  assert.throws(() => parseFormulaFields('\\frac{1}'))
})
test('new display delimiters and root degrees survive editing without source changes', () => {
  const source = String.raw`x_{i}^{2}=\sqrt[3]{\frac{a+\bar{b}}{c}}+\{x\}`
  assert.equal(formulaFieldText(parseFormulaFields(source)), source)
  const text = `说明\n\\[${source}\\]\n后续`
  const [range] = formulaRanges(text)
  assert.equal(range.source, source)
  assert.equal(text.slice(range.start, range.end), `\\[${source}\\]`)
  assert.deepEqual(formulaSelection(text, range.start + 3, range.end - 3), {
    start: range.start,
    end: range.end,
  })
  assert.equal(formulaRanges('\\[x=1\\)').length, 0)
})
test('reopening repeated calculation text edits the occurrence under the caret', () => {
  const records = [{ id: 'a', text: '组\n\\(x=1\\)', plan: {} }]
  const text = records[0].text + '\n其他\n' + records[0].text
  const occurrence = calculationAt(text, records, text.length - 1)
  assert.equal(occurrence.from, text.lastIndexOf(records[0].text))
  assert.equal(calculationAt('已经删除', records, 0), undefined)
})
