<script setup lang="ts">
import { ref } from 'vue'
import type { LayoutSettings } from '../../types'
import type { TableInsertOptions, TableStyle } from '../../state/tableTypes'
const props = defineProps<{ settings: LayoutSettings }>()
const emit = defineEmits<{ insert: [TableInsertOptions] }>()
const rows = ref(4),
  cols = ref(3),
  width = ref(
    Math.round(
      Math.min(
        120,
        props.settings.paperWidth - props.settings.marginLeft - props.settings.marginRight,
      ),
    ),
  ),
  style = ref<TableStyle>('grid'),
  error = ref('')
function insert() {
  if (
    rows.value < 1 ||
    rows.value > 50 ||
    cols.value < 1 ||
    cols.value > 20 ||
    !Number.isInteger(rows.value) ||
    !Number.isInteger(cols.value) ||
    width.value < cols.value * 3 ||
    width.value > props.settings.paperWidth
  ) {
    error.value = '请输入 1–50 行、1–20 列，宽度至少每列 3 mm 且不超过纸张。'
    return
  }
  emit('insert', { rows: rows.value, cols: cols.value, width: width.value, style: style.value })
}
</script>
<template>
  <form class="table-form" @submit.prevent="insert">
    <label
      >样式<select v-model="style" aria-label="表格样式">
        <option value="grid">普通网格表</option>
        <option value="three-line">三线表</option>
        <option value="horizontal">仅横线表</option>
      </select></label
    >
    <div class="table-form-row">
      <label
        >行数<input
          v-model.number="rows"
          type="number"
          min="1"
          max="50"
          step="1"
          aria-label="表格行数"
          required /></label
      ><label
        >列数<input
          v-model.number="cols"
          type="number"
          min="1"
          max="20"
          step="1"
          aria-label="表格列数"
          required
      /></label>
    </div>
    <label
      >宽度（mm）<input
        v-model.number="width"
        type="number"
        :min="cols * 3"
        :max="settings.paperWidth"
        step="0.5"
        aria-label="表格宽度"
        required
    /></label>
    <p v-if="error" class="table-error" role="alert">{{ error }}</p>
    <button class="button primary" type="submit">插入</button>
  </form>
</template>
