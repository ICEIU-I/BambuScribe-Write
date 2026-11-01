import type { Document, ImageAsset, TableAsset } from '../types'
import { mapBodyCuts, cutsOnPage } from './inkCutHistory'
import { placedObjects } from './objectLayout'
import { documentPages, checkPageText, type PageContents } from './documentPages'

export function insertDocumentPage(
  contents: PageContents,
  doc: Document,
  index: number,
  mode: 'copy' | 'blank',
) {
  if (!Number.isInteger(index) || !doc.pages[index]) return null
  const pages = documentPages(contents, doc)
  if (!pages) return null
  const { sections, pageLayouts } = pages
  const duplicate = mode === 'copy'
  sections.splice(index + 1, 0, duplicate ? sections[index] : '')
  pageLayouts.splice(index + 1, 0, {
    ...pageLayouts[index],
    continuation: duplicate ? pageLayouts[index].continuation : false,
  })
  const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value))
  const groups = new Map<string, string>()
  const objectIds = new Map<string, string>()
  const regionIds = new Map<string, string>()
  function copies<T extends ImageAsset | TableAsset>(assets: T[]) {
    const shown = placedObjects(assets, doc)
    return [
      ...shown.map((item) => ({ ...item, page: item.page > index ? item.page + 1 : item.page })),
      ...shown
        .filter((item) => duplicate && item.page === index)
        .map((item) => {
          const next = { ...clone(item), id: crypto.randomUUID(), page: index + 1 }
          objectIds.set(item.id, next.id)
          if ('groupId' in next && next.groupId) {
            if (!groups.has(next.groupId)) groups.set(next.groupId, crypto.randomUUID())
            next.groupId = groups.get(next.groupId)
          }
          if ('textRegions' in next)
            next.textRegions = next.textRegions?.map((region) => {
              const id = crypto.randomUUID()
              regionIds.set(`${item.id}:${region.id}`, id)
              return { ...region, id }
            })
          return next
        }),
    ]
  }
  const text = sections.join('\f')
  const images = copies(contents.images),
    tables = copies(contents.tables)
  if (tables.length > 50) throw new Error('新增后表格和文本框超过 50 个，请先拆分项目。')
  if (pageLayouts.length > 20000) throw new Error('新增后页面过多，请先拆分项目。')
  checkPageText(text, tables)
  const start = sections.slice(0, index + 1).reduce((n, s) => n + s.length + 1, 0)
  return {
    text,
    images,
    tables,
    pageLayouts,
    erasures: [
      ...mapBodyCuts(contents.erasures, (n) => {
        let p = 0,
          old = 0
        while (p < doc.pages.length - 1 && old + doc.pages[p]!.text.length <= n) {
          old += doc.pages[p]!.text.length
          p++
        }
        const dest = p > index ? p + 1 : p
        return sections.slice(0, dest).reduce((v, s) => v + s.length + 1, 0) + n - old
      }).map((e) => (e.page > index ? { ...e, page: e.page + 1 } : e)),
      ...contents.erasures
        .filter(
          (e) =>
            duplicate && (e.page === index || (e.cuts && cutsOnPage(e, doc, index)!.length > 0)),
        )
        .map((e) => ({
          ...clone(e),
          id: crypto.randomUUID(),
          page: index + 1,
          ...(e.cuts
            ? {
                cuts: cutsOnPage(e, doc, index)!.map((c) =>
                  c.objectId
                    ? {
                        ...c,
                        objectId: objectIds.get(c.objectId)!,
                        part: c.part.startsWith('region:')
                          ? `region:${regionIds.get(`${c.objectId}:${c.part.slice(7)}`) ?? c.part.slice(7)}`
                          : c.part,
                      }
                    : {
                        ...c,
                        part: String(
                          start +
                            Number(c.part) -
                            doc.pages.slice(0, index).reduce((n, p) => n + p.text.length, 0),
                        ),
                      },
                ),
              }
            : {}),
          ...(e.objectIds
            ? {
                objectIds: e.objectIds.flatMap((id) =>
                  objectIds.has(id) ? [objectIds.get(id)!] : [],
                ),
              }
            : {}),
        })),
    ],
    start,
    end: start,
  }
}
