import type { Document } from '../types'
export interface CalculationVariable {
  id: string
  label: string
  value: string
  unit: string
  source: string
  uncertain: boolean
}
export interface Conversion {
  factor: string
  unit: string
  precision: number
  format: 'decimal' | 'significant'
}
export interface CalculationStep {
  id: string
  label: string
  symbolic: string
  expression: string
  unit: string
  precision: number
  format: 'decimal' | 'significant'
  useRounded: boolean
  conversions: Conversion[]
}
export interface CalculationPlan {
  variables: CalculationVariable[]
  sections: { title: string; steps: CalculationStep[] }[]
  issues: string[]
}
export interface CalculationRecord {
  id: string
  text: string
  plan: CalculationPlan
}
export interface CalculationSources {
  data: string[]
  references: string[]
  instruction: string
}
export interface CalculationPreview {
  text: string
  values: Record<string, string>
  document: Document
}
export interface SourceImage {
  id: string
  name: string
  data: string
  size: number
}
