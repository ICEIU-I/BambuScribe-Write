import type { Document, ImageAsset, PageErasure, PageLayout, TableAsset } from '../types'

export type PageAction = 'copy' | 'blank' | 'delete' | 'move'
export interface PageContents {
  text: string
  images: ImageAsset[]
  tables: TableAsset[]
  erasures: PageErasure[]
}

// Freeze the current physical sheets before changing their order or count.
export function documentPages(contents: PageContents, doc: Document) {
  const original = contents.text.replace(/\r\n?/g, '\n')
  if (doc.pages.map((p) => p.text).join('') !== original) return null
  const sections: string[] = []
  const pageLayouts: PageLayout[] = []
  let offset = 0
  for (const [i, page] of doc.pages.entries()) {
    sections.push(page.text.replace(/\f$/, ''))
    pageLayouts.push({
      lineCount: Math.round(
        (((doc.settings.paperHeight - doc.settings.marginTop - doc.settings.marginBottom) /
          page.lineHeight) *
          doc.settings.lineSpacing) /
          100,
      ),
      continuation:
        offset === 0 || original[offset - 1] === '\f'
          ? !!doc.settings.pageLayouts?.[i]?.continuation
          : original[offset - 1] !== '\n',
    })
    offset += page.text.length
  }
  return { sections, pageLayouts }
}

export function checkPageText(text: string, tables: TableAsset[]) {
  const characters =
    Array.from(text).length +
    tables.reduce((n, t) => n + t.cells.reduce((m, c) => m + Array.from(c.text).length, 0), 0)
  if (characters > 20000) throw new Error('分页后文字超过 20,000 字，请先拆分项目。')
}
