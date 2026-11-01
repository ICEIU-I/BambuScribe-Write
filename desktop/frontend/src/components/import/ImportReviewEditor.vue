<script setup lang="ts">
import { ref } from 'vue'
import type { Extraction, RecognizedCell } from '../../services/tableImportTypes'
import type { ImportSource } from '../../state/useBatchTableImport'
const props = defineProps<{
  extraction: Extraction
  selected: number
  origins: number[]
  sources: ImportSource[]
  disabled: boolean
}>()
const emit = defineEmits<{ select: [index: number]; change: [] }>()
const enlarged = ref(false)
function editTitle(e: Event) {
  table()!.title = (e.target as HTMLInputElement).value
  emit('change')
}
function editCell(c: RecognizedCell, e: Event) {
  c.text = (e.target as HTMLTextAreaElement).value
  c.uncertain = false
  emit('change')
}
function editNotes(e: Event) {
  table()!.notes = (e.target as HTMLTextAreaElement).value.split('\n').filter(Boolean)
  emit('change')
}
const table = () => props.extraction.tables[props.selected]
const source = () => props.sources[props.origins[props.selected]!]
function rows() {
  return Array.from({ length: table()?.rows ?? 0 }, (_, row) =>
    table()!
      .cells.filter((c) => c.row === row)
      .sort((a, b) => a.col - b.col),
  )
}
</script>
<template>
  <section class="import-review-editor">
    <label class="import-field"
      >核对表格<select
        :value="selected"
        aria-label="核对表格"
        @change="emit('select', Number(($event.target as HTMLSelectElement).value))"
      >
        <option v-for="(t, i) in extraction.tables" :key="i" :value="i">
          {{ sources[origins[i]!]!.name }} · {{ t.title || `表格 ${i + 1}` }}
        </option>
      </select></label
    >
    <template v-if="table()">
      <button
        class="import-original-button"
        :class="{ enlarged }"
        :aria-expanded="enlarged"
        aria-label="放大或缩小原图"
        @click="enlarged = !enlarged"
      >
        <img class="import-original" :src="source()?.data" alt="当前表格原图" />
      </button>
      <label class="import-field"
        >表名<input
          :value="table()!.title"
          aria-label="识别表名"
          :disabled="disabled"
          @input="editTitle"
      /></label>
      <div class="import-cell-scroll">
        <table class="import-cell-editor">
          <tbody>
            <tr v-for="(row, ri) in rows()" :key="ri">
              <td
                v-for="c in row"
                :key="c.col"
                :rowspan="c.rowSpan"
                :colspan="c.colSpan"
                :class="{ uncertain: c.uncertain }"
              >
                <textarea
                  :aria-label="`第 ${c.row + 1} 行第 ${c.col + 1} 列`"
                  :value="c.text"
                  :disabled="disabled"
                  :title="c.uncertain ? '需对照原图核对' : ''"
                  @input="editCell(c, $event)"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <label class="import-field"
        >说明<textarea
          :value="table()!.notes.join('\n')"
          aria-label="识别说明"
          :disabled="disabled"
          @input="editNotes"
        />
      </label>
    </template>
  </section>
</template>
