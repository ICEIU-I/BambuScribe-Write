import type { LayoutSettings } from '../types'
import type { TableAsset } from './tableTypes'
import { newTableId } from './tableOperations'
import type { PaperPosition } from './paperGeometry'

// A text box uses one borderless cell in the shared vector object pipeline.
// The kind retains its own editor and controls across projects and clipboard.
export const isFreeTextBox = (asset: TableAsset) => asset.kind === 'textbox'
export function createFreeTextBox(settings: LayoutSettings, position: PaperPosition): TableAsset {
  const width = Math.min(70, settings.paperWidth) / 2,
    fontSize = settings.fontSize,
    height = Math.max(8, fontSize * 1.5 + 1) / 2
  return {
    id: newTableId(),
    kind: 'textbox',
    page: position.page,
    x: Math.max(0, Math.min(position.x, settings.paperWidth - width)),
    y: Math.max(0, Math.min(position.y, settings.paperHeight - height)),
    columnWidths: [width],
    rowHeights: [height],
    cells: [{ row: 0, col: 0, rowSpan: 1, colSpan: 1, text: '' }],
    style: 'none',
    fontSize,
    padding: 0.5,
    align: 'left',
    verticalAlign: 'middle',
  }
}
