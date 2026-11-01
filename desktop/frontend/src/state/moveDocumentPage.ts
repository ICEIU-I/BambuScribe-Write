import { remapPageCuts } from './inkCutHistory'
import type { Document } from '../types'
import { documentPages, checkPageText, type PageContents } from './documentPages'
import { placedObjects } from './objectLayout'

export function moveDocumentPage(contents: PageContents, doc: Document, from: number, to?: number) {
  if (
    to === undefined ||
    !Number.isInteger(from) ||
    !Number.isInteger(to) ||
    !doc.pages[from] ||
    !doc.pages[to] ||
    from === to
  )
    return null
  const pages = documentPages(contents, doc)
  if (!pages) return null
  const order = doc.pages.map((_, i) => i)
  order.splice(to, 0, order.splice(from, 1)[0])
  const destinations = new Map(order.map((source, target) => [source, target]))
  const sections = order.map((i) => pages.sections[i])
  // A newline that led into another sheet must not create a new empty sheet
  // when that full page is moved to the end of the document.
  if (order.at(-1) !== doc.pages.length - 1)
    sections[sections.length - 1] = sections.at(-1)!.replace(/\n$/, '')
  const text = sections.join('\f')
  const remap = <T extends { page: number }>(item: T): T => ({
    ...item,
    page: destinations.get(item.page) ?? item.page,
  })
  const tables = placedObjects(contents.tables, doc).map(remap)
  checkPageText(text, tables)
  const start = sections.slice(0, to).reduce((n, section) => n + section.length + 1, 0)
  return {
    text,
    pageLayouts: order.map((i) => pages.pageLayouts[i]),
    images: placedObjects(contents.images, doc).map(remap),
    tables,
    erasures: remapPageCuts(contents.erasures, doc, order, sections).map(remap),
    start,
    end: start,
  }
}
