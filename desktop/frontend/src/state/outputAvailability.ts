import type { Document } from '../types'

type OutputState = {
  document: Document | null
  index: number
  ready: boolean
  loading: boolean
  dirty: boolean
  busy: boolean
}
export function outputAvailability(state: OutputState): { current: string; all: string } {
  const common = !state.ready
    ? '桌面服务尚未就绪，请重新打开应用。'
    : state.busy
      ? '正在处理操作，请稍候。'
      : state.loading
        ? '正在排版，请稍候。'
        : state.dirty
          ? '排版尚未完成，请检查文字和排版参数。'
          : !state.document
            ? '请先输入文字。'
            : ''
  if (common || !state.document) return { current: common, all: common }
  const pages = state.document.pages
  function missingReason(page: Document['pages'][number]) {
    const chars = [...new Set(page.missing.map((item) => item.char))]
    return `第 ${page.number} 页缺字：${chars.slice(0, 8).join(' ')}${chars.length > 8 ? '…' : ''}。请修改标记的文字后导出。`
  }
  const page = pages[state.index],
    missingPage = pages.find((item) => item.missing.length),
    issuePage = pages.find((item) => item.issues?.length)
  return {
    current: !page
      ? '当前页不存在，请重新选择页面。'
      : page.missing.length
        ? missingReason(page)
        : page.issues?.length
          ? `第 ${page.number} 页：${page.issues[0]}`
          : !page.strokes.length
            ? '当前页没有可书写内容，请输入文字。'
            : '',
    all: missingPage
      ? missingReason(missingPage)
      : issuePage
        ? `第 ${issuePage.number} 页：${issuePage.issues![0]}`
        : !pages.some((item) => item.strokes.length)
          ? '文档没有可书写内容，请输入文字。'
          : '',
  }
}
