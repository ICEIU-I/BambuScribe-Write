export interface FormulaField {
  kind: 'text' | 'fraction' | 'sqrt' | 'bar' | 'sub' | 'super'
  text?: string
  children?: FormulaField[]
  denominator?: FormulaField[]
}
export function parseFormulaFields(source: string): FormulaField[] {
  let at = 0
  function group(depth: number): FormulaField[] {
    if (source[at++] !== '{') throw new Error('公式括号不完整')
    return sequence(depth + 1, true)
  }
  function sequence(depth: number, grouped = false): FormulaField[] {
    if (depth > 8) throw new Error('公式嵌套过多')
    const nodes: FormulaField[] = []
    while (at < source.length && source[at] !== '}') {
      if (source.startsWith('\\frac', at)) {
        at += 5
        nodes.push({ kind: 'fraction', children: group(depth), denominator: group(depth) })
        continue
      }
      if (source.startsWith('\\sqrt', at)) {
        at += 5
        nodes.push({ kind: 'sqrt', children: group(depth) })
        continue
      }
      if (source.startsWith('\\bar', at)) {
        at += 4
        nodes.push({ kind: 'bar', children: group(depth) })
        continue
      }
      if (source[at] === '_' || source[at] === '^') {
        const kind = source[at++] === '_' ? 'sub' : 'super'
        nodes.push({ kind, children: group(depth) })
        continue
      }
      if (source[at] === '\\' || source[at] === '{') throw new Error('暂不支持此公式写法')
      const start = at++
      while (at < source.length && !['\\', '{', '}', '_', '^'].includes(source[at])) at++
      nodes.push({ kind: 'text', text: source.slice(start, at) })
    }
    if (grouped && source[at++] !== '}') throw new Error('公式括号不完整')
    return nodes
  }
  const result = sequence(0)
  if (at !== source.length) throw new Error('公式括号不匹配')
  return result
}
export function formulaFieldText(nodes: FormulaField[]): string {
  return nodes
    .map((n) => {
      if (n.kind === 'text') return n.text ?? ''
      const a = formulaFieldText(n.children ?? [])
      if (n.kind === 'fraction') return `\\frac{${a}}{${formulaFieldText(n.denominator ?? [])}}`
      if (n.kind === 'sub' || n.kind === 'super') return `${n.kind === 'sub' ? '_' : '^'}{${a}}`
      return `\\${n.kind}{${a}}`
    })
    .join('')
}
