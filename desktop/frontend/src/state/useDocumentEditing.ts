import type { CalculationRecord } from '../services/calculationTypes'
import { moveImportedTable } from './importedTableGroup'
import { targetErasure } from './erasureTargets'
import { rebaseBodyCuts, resetChangedObjectCuts } from './inkCutHistory'
import { nextTick, ref, watch, toRaw, type Ref } from 'vue'
import type {
  Document,
  ImageAsset,
  ImageTextRegion,
  Point,
  TableAsset,
  PageErasure,
  PageLayout,
} from '../types'
import { t } from '../i18n'
import { createEditorHistory, type EditorSnapshot } from './editorHistory'
import { insertedRange, tidySpacing } from './pasteTidy'
import { isTextBox } from './textBox'
import { displayRegions } from './imageEditing'
import { deleteDocumentPage } from './deleteDocumentPage'
import { insertDocumentPage } from './insertDocumentPage'
import { moveDocumentPage } from './moveDocumentPage'
import type { PageAction } from './documentPages'
import { useObjectClipboard, type ImageClipboardPlacement } from './useObjectClipboard'

// How much text one input event has to carry before it can only have arrived
// from a paste or a drop rather than from typing.
const bulkInsertCharacters = 8

interface EditingProps {
  calculations?: CalculationRecord[]
  pageLayouts?: PageLayout[]
  modelValue: string
  images: ImageAsset[]
  tables?: TableAsset[]
  erasures?: PageErasure[]
  document: Document | null
  disabled: boolean
  active?: boolean
}

export function useDocumentEditing(
  props: EditingProps,
  input: Ref<HTMLTextAreaElement | undefined>,
  output: (
    text: string,
    images: ImageAsset[],
    tables: TableAsset[],
    erasures: PageErasure[],
    pageLayouts?: PageLayout[],
    calculations?: CalculationRecord[],
  ) => void,
  restoreSelection: (a: number, b: number) => void,
  notice: (message: string) => void = () => {},
  placement?: ImageClipboardPlacement,
) {
  const tableId = ref(''),
    imageId = ref(''),
    regionId = ref('')
  let pasteAnchor: Point | undefined
  let pasted = false
  const history = createEditorHistory()
  let pendingOutput:
    | {
        text: string
        images: ImageAsset[]
        tables: TableAsset[]
        erasures: PageErasure[]
        pageLayouts?: PageLayout[]
        calculations?: CalculationRecord[]
      }
    | undefined
  let calculations = props.calculations ?? []
  let text = props.modelValue
  let images = props.images
  let tables = props.tables ?? []
  let erasures = props.erasures ?? []
  let pageLayouts = props.pageLayouts
  let selection = { start: 0, end: 0 }

  const current = (): EditorSnapshot => ({
    calculations,
    pageLayouts,
    tables,
    erasures,
    tableId: tableId.value,
    text,
    images,
    imageId: imageId.value,
    regionId: regionId.value,
    ...selection,
  })

  watch(
    () =>
      [
        props.modelValue,
        props.images,
        props.tables,
        props.erasures,
        props.pageLayouts,
        props.calculations,
      ] as const,
    ([t, i, ts, es, ps, cs]) => {
      if (
        pendingOutput &&
        t === pendingOutput.text &&
        sameImages(i, pendingOutput.images) &&
        (ts === undefined || toRaw(ts) === toRaw(pendingOutput.tables)) &&
        (es === undefined || toRaw(es) === toRaw(pendingOutput.erasures)) &&
        toRaw(ps) === toRaw(pendingOutput.pageLayouts) &&
        toRaw(cs) === toRaw(pendingOutput.calculations)
      ) {
        text = t
        images = i
        tables = ts ?? []
        erasures = es ?? []
        pageLayouts = ps
        calculations = cs ?? []
        pendingOutput = undefined
        return
      }
      if (
        t !== text ||
        toRaw(i) !== toRaw(images) ||
        (ts !== undefined && toRaw(ts) !== toRaw(tables)) ||
        (es !== undefined && toRaw(es) !== toRaw(erasures)) ||
        toRaw(ps) !== toRaw(pageLayouts) ||
        toRaw(cs) !== toRaw(calculations)
      ) {
        text = t
        images = i
        tables = ts ?? []
        erasures = es ?? []
        pageLayouts = ps
        calculations = cs ?? []
        tableId.value = ''
        history.reset()
      }
    },
    // A mixed paste updates image and table props in the same render. Wait
    // for the complete batch before reconciling it with the history entry.
    { flush: 'pre' },
  )

  function commit(next: Partial<EditorSnapshot>, group?: string) {
    const before = current()
    const after = { ...before, ...next }
    if (!next.calculations && after.text !== before.text)
      after.calculations = calculations.filter((r) => after.text.includes(r.text))
    if (!next.erasures) {
      const frozen = erasures.some((e) => !e.cuts)
        ? erasures.map((e) => targetErasure(e, props.document?.pages[e.page]))
        : erasures
      after.erasures = resetChangedObjectCuts(
        rebaseBodyCuts(frozen, before.text, after.text),
        before.tables ?? [],
        after.tables ?? [],
        before.images,
        after.images,
      )
    }
    history.record(before, after, group)
    calculations = after.calculations ?? []
    text = after.text
    images = after.images
    tables = after.tables ?? []
    erasures = after.erasures ?? []
    pageLayouts = after.pageLayouts
    tableId.value = after.tableId ?? ''
    imageId.value = after.imageId
    regionId.value = after.regionId
    selection = { start: after.start, end: after.end }
    pendingOutput = { text, images: after.images, tables, erasures, pageLayouts, calculations }
    output(text, images, tables, erasures, pageLayouts, calculations)
  }

  function beforeInput(e: InputEvent) {
    if (e.inputType === 'historyUndo' || e.inputType === 'historyRedo') {
      e.preventDefault()
      travel(e.inputType === 'historyUndo' ? 'undo' : 'redo')
      return
    }
    if (e.inputType === 'insertFromPaste') pasted = true
    if (input.value)
      selection = { start: input.value.selectionStart, end: input.value.selectionEnd }
  }

  // The beforeinput above is the precise signal but not a universal one: the
  // paste event is what every engine reports, so the paper announces its own
  // pastes too. Either flag is consumed by the input event that follows.
  function notePaste() {
    pasted = true
  }

  function changeText(next: string) {
    const previous = text,
      fromPaste = pasted
    pasted = false
    commit(
      {
        text: next,
        imageId: '',
        tableId: '',
        regionId: '',
        start: input.value?.selectionStart ?? 0,
        end: input.value?.selectionEnd ?? 0,
      },
      'typing',
    )
    tidyPaste(next, previous, fromPaste)
  }

  // Clipboard text whose block boundaries were flattened into runs of spaces is
  // repaired as its own history step: one undo brings the pasted text back as
  // the clipboard had it, a second removes the paste altogether.
  async function tidyPaste(after: string, before: string, pasted: boolean) {
    const [start, end] = insertedRange(before, after)
    // A block of text arrives only from a paste or a drop; no keystroke or IME
    // commit inserts this much in one event. Engines disagree about how they
    // report a paste, so the size of the insert is the signal that always holds.
    if (!pasted && end - start < bulkInsertCharacters) return
    const tidied = tidySpacing(after.slice(start, end))
    if (!tidied.count) return
    const caret = start + tidied.text.length
    commit(
      {
        text: after.slice(0, start) + tidied.text + after.slice(end),
        imageId: '',
        tableId: '',
        regionId: '',
        start: caret,
        end: caret,
      },
      'paste',
    )
    notice(t('editor.tidyPaste', { count: tidied.count }))
    await nextTick()
    restoreSelection(caret, caret)
  }

  function select(id: string, point?: Point) {
    tableId.value = ''
    regionId.value = ''
    pasteAnchor = point
    history.breakGroup()
    imageId.value = id
    input.value?.blur()
  }

  function selectRegion(id: string, rid: string) {
    select(id)
    regionId.value = rid
  }

  function deselect() {
    tableId.value = ''
    regionId.value = ''
    pasteAnchor = undefined
    history.breakGroup()
    imageId.value = ''
  }

  function selectTable(id: string) {
    deselect()
    tableId.value = id
    input.value?.blur()
  }

  function changeTable(table: TableAsset, wholeGroup = false) {
    let next: TableAsset[]
    try {
      next =
        moveImportedTable(table, tables, props.document, wholeGroup) ??
        tables.map((t) => (t.id === table.id ? table : t))
    } catch (e) {
      notice(e instanceof Error ? e.message : String(e))
      return
    }
    commit({
      tables: next,
      tableId: table.id,
      imageId: '',
      regionId: '',
    })
  }
  function addTable(table: TableAsset) {
    commit({ tables: [...tables, table], tableId: table.id, imageId: '', regionId: '' })
  }
  function removeTable(id: string) {
    commit({ tables: tables.filter((t) => t.id !== id), tableId: '' })
  }

  function changeImage(image: ImageAsset) {
    commit({
      images: images.map((item) => (item.id === image.id ? image : item)),
      imageId: image.id,
      tableId: '',
      regionId: '',
    })
  }

  function addImage(image: ImageAsset) {
    commit({ images: [...images, image], imageId: image.id, tableId: '', regionId: '' })
  }

  function removeImage(id: string) {
    commit({ images: images.filter((item) => item.id !== id), imageId: '', regionId: '' })
  }

  function editRegionText(id: string, region: ImageTextRegion) {
    if (!isTextBox(region)) return
    const image = images.find((item) => item.id === id)
    if (!image) return
    const regions = (image.textRegions ?? []).map((item) =>
      item.id === region.id && isTextBox(item)
        ? {
            ...item,
            text: region.text,
            x: region.x,
            y: region.y,
            width: region.width,
            height: region.height,
          }
        : item,
    )
    commit({
      images: images.map((item) => (item.id === id ? { ...item, textRegions: regions } : item)),
      imageId: id,
      regionId: region.id,
    })
  }

  function addRegion(id: string, region: ImageTextRegion) {
    const image = images.find((item) => item.id === id)
    if (!image) return
    const preview = props.document?.pages
      .flatMap((page) => page.images ?? [])
      .find((item) => item.id === id)
    const regions = displayRegions(image, preview)
    if (regions.some((item) => item.id === region.id)) return
    commit({
      images: images.map((item) =>
        item.id === id
          ? { ...item, textRegions: [...regions, { ...region, source: 'manual' as const }] }
          : item,
      ),
      imageId: id,
      regionId: isTextBox(region) ? region.id : '',
    })
  }

  async function travel(direction: 'undo' | 'redo') {
    if (props.disabled || !props.active) return
    const state = history[direction]()
    if (!state) return
    const erasureChange = state.erasures !== erasures
    calculations = state.calculations ?? []
    text = state.text
    images = state.images
    tables = state.tables ?? []
    erasures = state.erasures ?? []
    pageLayouts = state.pageLayouts
    tableId.value = state.tableId ?? ''
    selection = { start: state.start, end: state.end }
    imageId.value = state.imageId
    regionId.value = state.regionId
    pasteAnchor = undefined
    pendingOutput = { text, images, tables, erasures, pageLayouts, calculations }
    output(text, images, tables, erasures, pageLayouts, calculations)
    await nextTick()
    if (!state.imageId && !state.tableId && !erasureChange) restoreSelection(state.start, state.end)
  }

  function removeRegion(id: string, rid: string) {
    const image = images.find((item) => item.id === id)
    if (!image?.textRegions?.some((r) => r.id === rid && isTextBox(r))) return
    commit({
      images: images.map((item) =>
        item.id === id
          ? { ...item, textRegions: item.textRegions?.filter((r) => r.id !== rid) }
          : item,
      ),
      imageId: id,
      regionId: '',
    })
  }

  const { copy, cut, paste } = useObjectClipboard(
    props,
    {
      images: () => images,
      selection: () => ({ imageId: imageId.value, regionId: regionId.value, point: pasteAnchor }),
      paperInput: () => input.value,
      advanceText: (point) => {
        pasteAnchor = point
      },
      addImage,
      removeImage,
      addRegion,
      removeRegion,
    },
    placement,
  )

  function globalKeydown(e: KeyboardEvent) {
    if (props.active === false || props.disabled || e.isComposing || e.defaultPrevented) return
    const target = e.target as HTMLElement | null
    if (target?.closest('dialog,[role="dialog"],.table-overlay')) return
    if (target?.closest('input,textarea,[contenteditable="true"]') && target !== input.value) return
    const key = e.key.toLowerCase()
    if ((e.ctrlKey || e.metaKey) && !e.altKey && (key === 'z' || key === 'y')) {
      e.preventDefault()
      e.stopPropagation()
      travel(key === 'y' || e.shiftKey ? 'redo' : 'undo')
      return
    }
    if (key === 'escape') {
      deselect()
      return
    }
    if (tableId.value && (key === 'backspace' || key === 'delete')) {
      e.preventDefault()
      removeTable(tableId.value)
      return
    }
    if (imageId.value && (key === 'backspace' || key === 'delete')) {
      e.preventDefault()
      e.stopPropagation()
      if (regionId.value) removeRegion(imageId.value, regionId.value)
      else removeImage(imageId.value)
    }
  }

  return {
    replaceGenerated: (from: number, to: number, inserted: string, record: CalculationRecord) => {
      const nextText = text.slice(0, from) + inserted + text.slice(to)
      const retained = calculations.filter(
        (r) => nextText.includes(r.text) && (r.id !== record.id || r.text !== record.text),
      )
      if (retained.some((r) => r.id === record.id)) record = { ...record, id: crypto.randomUUID() }
      commit({
        text: nextText,
        calculations: [...retained, record],
        start: from,
        end: from,
        tableId: '',
        imageId: '',
        regionId: '',
      })
    },
    replaceFormula: (from: number, to: number, value: string) =>
      commit({ text: text.slice(0, from) + value + text.slice(to), start: from, end: from }),
    changeObjects: (changedImages: ImageAsset[], changedTables: TableAsset[]) => {
      const imageMap = new Map(changedImages.map((im) => [im.id, im])),
        tableMap = new Map(changedTables.map((table) => [table.id, table]))
      commit({
        images: images.map((im) => imageMap.get(im.id) ?? im),
        tables: tables.map((table) => tableMap.get(table.id) ?? table),
        tableId: '',
        imageId: '',
        regionId: '',
      })
    },
    changePage: (index: number, action: PageAction, destination?: number) => {
      if (props.disabled || !props.document) return false
      const contents = {
        text,
        images,
        tables,
        erasures: erasures.map((e) => targetErasure(e, props.document?.pages[e.page])),
      }
      const next =
        action === 'move'
          ? moveDocumentPage(contents, props.document, index, destination)
          : action === 'delete'
            ? deleteDocumentPage(contents, props.document, index)
            : insertDocumentPage(contents, props.document, index, action)
      if (!next) return false
      commit({ ...next, tableId: '', imageId: '', regionId: '' })
      pasteAnchor = undefined
      return true
    },
    addObjects: (additions: ImageAsset[], addedTables: TableAsset[]) =>
      commit({
        images: [...images, ...additions],
        tables: [...tables, ...addedTables],
        tableId: addedTables.at(-1)?.id ?? '',
        imageId: addedTables.length ? '' : (additions.at(-1)?.id ?? ''),
        regionId: '',
      }),
    addErasure: (erasure: PageErasure) =>
      commit({ erasures: [...erasures, erasure], tableId: '', imageId: '', regionId: '' }),
    tableId,
    selectTable,
    changeTable,
    addTables: (additions: TableAsset[]) =>
      commit({
        tables: [...tables, ...additions],
        tableId: additions.at(-1)?.id ?? '',
        imageId: '',
        regionId: '',
      }),
    addTable,
    removeTable,
    imageId,
    regionId,
    selectRegion,
    removeRegion,
    copy,
    cut,
    paste,
    notePaste,
    beforeInput,
    changeText,
    select,
    deselect,
    changeImage,
    addImage,
    removeImage,
    editRegionText,
    addRegion,
    globalKeydown,
    undo: () => travel('undo'),
    redo: () => travel('redo'),
  }
}

function sameImages(a: ImageAsset[], b: ImageAsset[]) {
  const left = toRaw(a) as ImageAsset[]
  const right = toRaw(b) as ImageAsset[]
  if (left === right) return true
  if (!Array.isArray(left) || !Array.isArray(right) || left.length !== right.length) return false
  return left.every((image, index) => {
    const other = toRaw(right[index]) as ImageAsset
    return (
      image === other ||
      (image?.id !== undefined &&
        image.id === other?.id &&
        image.data === other.data &&
        JSON.stringify(image.diagram) === JSON.stringify(other.diagram))
    )
  })
}
