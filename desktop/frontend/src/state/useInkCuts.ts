import { inject, type InjectionKey, type Ref } from 'vue'
import type { InkLayer, Point } from '../types'
import { pageErasuresKey } from './pageEraser'
import { cutInk } from './inkCuts'
export const pageInkKey: InjectionKey<Ref<InkLayer[]>> = Symbol('page-ink')
export function useInkCuts(id: () => string) {
  const layers = inject(pageInkKey, undefined),
    erasures = inject(pageErasuresKey, undefined)
  return (strokes: Point[][], part: string) =>
    cutInk(
      strokes,
      layers?.value.find((l) => l.objectId === id() && l.part === part),
      erasures?.value ?? [],
    )
}
