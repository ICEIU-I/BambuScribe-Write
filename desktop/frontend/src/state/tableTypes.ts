import type { Point, MissingGlyph } from '../types'

export type TableStyle = 'grid' | 'three-line' | 'horizontal' | 'none'
export type TableAlign = 'left' | 'center' | 'right'
export type TableVerticalAlign = 'top' | 'middle' | 'bottom'
export type TableBorders = Partial<Record<'top' | 'right' | 'bottom' | 'left', boolean>>
export interface TableCell {
  row: number
  col: number
  rowSpan: number
  colSpan: number
  text: string
  align?: TableAlign
  verticalAlign?: TableVerticalAlign
  borders?: TableBorders
}
export interface TableAsset {
  groupId?: string
  id: string
  kind?: 'textbox'
  page: number
  x: number
  y: number
  columnWidths: number[]
  rowHeights: number[]
  cells: TableCell[]
  style: TableStyle
  fontSize: number
  padding: number
  align: TableAlign
  verticalAlign: TableVerticalAlign
}
export interface TableCellPlacement extends TableCell {
  fontSize?: number
  x: number
  y: number
  width: number
  height: number
  textWidth?: number
  textHeight?: number
  textSpacing?: {
    lines: { width: number; gap: number }[]
    strokeGaps: [number, number][]
  }
  strokes: Point[][]
  missing: MissingGlyph[]
}
export interface TablePlacement {
  style?: TableAsset['style']
  id: string
  kind?: 'textbox'
  fontSize?: number
  maxFontSize?: number
  padding?: number
  x: number
  y: number
  width: number
  height: number
  columnWidths: number[]
  rowHeights: number[]
  cells: TableCellPlacement[]
  strokes: Point[][]
  issues: string[]
}
export interface TableSelection {
  r0: number
  c0: number
  r1: number
  c1: number
}
export interface TableInsertOptions {
  rows: number
  cols: number
  width: number
  style: TableStyle
}
