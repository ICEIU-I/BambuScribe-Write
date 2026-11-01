// Render only the small Markdown subset used by our bundled rules. Content is
// text, never HTML; the app and downloadable document share the same source.
export interface RuleBlock {
  kind: 'heading' | 'paragraph' | 'list' | 'table' | 'code'
  text?: string
  level?: number
  lines?: string[]
  rows?: string[][]
}
export function ruleBlocks(markdown: string): RuleBlock[] {
  const lines = markdown.split('\n'),
    blocks: RuleBlock[] = []
  for (let i = 0; i < lines.length; ) {
    const line = lines[i].trim()
    if (!line) {
      i++
      continue
    }
    if (line.startsWith('```')) {
      const code: string[] = []
      for (i++; i < lines.length && !lines[i].startsWith('```'); i++) code.push(lines[i])
      blocks.push({ kind: 'code', text: code.join('\n') })
      i++
      continue
    }
    const heading = /^(#{1,3})\s+(.*)/.exec(line)
    if (heading) {
      blocks.push({ kind: 'heading', level: heading[1].length, text: heading[2] })
      i++
      continue
    }
    if (line.startsWith('|')) {
      const rows: string[][] = []
      while (i < lines.length && lines[i].trim().startsWith('|')) {
        const row = lines[i++]
          .trim()
          .slice(1, -1)
          .split('|')
          .map((v) => v.trim())
        if (!row.every((v) => /^:?-+:?$/.test(v))) rows.push(row)
      }
      blocks.push({ kind: 'table', rows })
      continue
    }
    if (/^(?:- |\d+\. )/.test(line)) {
      const items: string[] = []
      while (i < lines.length && /^(?:- |\d+\. )/.test(lines[i].trim()))
        items.push(lines[i++].trim().replace(/^(?:- |\d+\. )/, ''))
      blocks.push({ kind: 'list', lines: items })
      continue
    }
    blocks.push({ kind: 'paragraph', text: line })
    i++
  }
  return blocks
}
export function ruleText(text: string) {
  return text
    .split(/(`[^`]+`|\*\*[^*]+\*\*)/g)
    .filter(Boolean)
    .map((part) => ({
      kind: part.startsWith('`') ? 'code' : part.startsWith('**') ? 'strong' : 'span',
      text: part.startsWith('`')
        ? part.slice(1, -1)
        : part.startsWith('**')
          ? part.slice(2, -2)
          : part,
    }))
}

export function rulePages(markdown: string) {
  const pages: { title: string; blocks: RuleBlock[] }[] = []
  for (const block of ruleBlocks(markdown)) {
    if (block.kind === 'heading' && block.level === 1) continue
    if (block.kind === 'heading' && block.level === 2)
      pages.push({ title: block.text!, blocks: [] })
    else pages.at(-1)?.blocks.push(block)
  }
  return pages
}
