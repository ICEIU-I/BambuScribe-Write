import { rebaseBodyCuts } from './inkCutHistory'
import type { Document, ImageAsset, PageErasure, TableAsset } from '../types'
import { placedObjects } from './objectLayout'

interface PageContents {
  text: string
  images: ImageAsset[]
  tables: TableAsset[]
  erasures: PageErasure[]
}

// Page.text contains source text, including explicit newlines but not visual
// wrapping. Its UTF-16 length therefore partitions the textarea exactly.
export function deleteDocumentPage(contents: PageContents, doc: Document, index: number) {
  const page = doc.pages[index]
  if (!page || !Number.isInteger(index)) return null
  const normalized = contents.text.replace(/\r\n?/g, '\n')
  if (doc.pages.map((p) => p.text).join('') !== normalized) return null
  let start = doc.pages.slice(0, index).reduce((n, p) => n + p.text.length, 0)
  const end = start + page.text.length
  // A final text page would otherwise return as an empty page: the newline
  // ending the preceding full page reserves a fresh line on the next sheet.
  if (
    end === normalized.length &&
    page.carets.length &&
    ['\n', '\f'].includes(normalized[start - 1])
  ) {
    const explicit = normalized[start - 1] === '\f'
    start--
    if (explicit && normalized[start - 1] === '\n') start--
  }
  const from = sourceOffset(contents.text, start),
    to = sourceOffset(contents.text, end)
  function remaining<T extends ImageAsset | TableAsset>(assets: T[]): T[] {
    return placedObjects(assets, doc).flatMap((shown, i) =>
      shown.page === index
        ? []
        : [shown.page > index ? { ...shown, page: shown.page - 1 } : assets[i]],
    )
  }
  return {
    pageLayouts: doc.settings?.pageLayouts?.filter((_, i) => i !== index),
    text: contents.text.slice(0, from) + contents.text.slice(to),
    images: remaining(contents.images),
    tables: remaining(contents.tables),
    erasures: rebaseBodyCuts(
      contents.erasures,
      contents.text,
      contents.text.slice(0, from) + contents.text.slice(to),
    )
      .filter((e) => e.cuts || e.page !== index)
      .map((e) => ({
        ...e,
        ...(e.cuts
          ? {
              cuts: e.cuts.filter(
                (c) =>
                  !c.objectId ||
                  [...remaining(contents.images), ...remaining(contents.tables)].some(
                    (o) => o.id === c.objectId,
                  ),
              ),
            }
          : {}),
      }))
      .map((e) => (e.page > index ? { ...e, page: e.page - 1 } : e)),
    start: from,
    end: from,
  }
}

// Keep untouched CRLF/CR bytes in older imported projects.
function sourceOffset(text: string, normalizedOffset: number) {
  let source = 0
  for (let offset = 0; offset < normalizedOffset; offset++, source++)
    if (text[source] === '\r' && text[source + 1] === '\n') source++
  return source
}
