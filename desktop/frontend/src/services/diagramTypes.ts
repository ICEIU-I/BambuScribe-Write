import type { Point } from '../types'

export interface RecognizedDiagram {
  data: string
  scene: DiagramScene
}

export interface DiagramScene {
  width: number
  height: number
  strokes: { id: string; role: string; points: Point[]; dashed: boolean }[]
  circles: { id: string; cx: number; cy: number; r: number; dashed: boolean }[]
  labels: { text: string; x: number; y: number; width: number; height: number }[]
  uncertainties: string[]
}
