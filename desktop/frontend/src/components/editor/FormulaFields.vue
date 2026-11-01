<script setup lang="ts">
import type { FormulaField } from '../../state/formulaFields'
defineProps<{ nodes: FormulaField[]; label: string; disabled: boolean }>()
const emit = defineEmits<{ change: [] }>()
</script>
<template>
  <span class="formula-fields">
    <span v-for="(node, i) in nodes" :key="i" :class="['formula-field', `formula-${node.kind}`]">
      <input
        v-if="node.kind === 'text'"
        v-model="node.text"
        :aria-label="`${label}${i + 1}`"
        :size="Math.max(1, Math.min(40, [...(node.text ?? '')].length))"
        :disabled="disabled"
        @input="emit('change')"
      />
      <template v-else-if="node.kind === 'fraction'"
        ><FormulaFields
          :nodes="node.children!"
          label="分子"
          :disabled="disabled"
          @change="emit('change')" /><FormulaFields
          :nodes="node.denominator!"
          label="分母"
          :disabled="disabled"
          @change="emit('change')"
      /></template>
      <template v-else
        ><span v-if="node.kind === 'sqrt'">√</span
        ><FormulaFields
          :nodes="node.children!"
          :label="node.kind === 'sub' ? '下标' : node.kind === 'super' ? '上标' : label"
          :disabled="disabled"
          @change="emit('change')"
      /></template>
    </span>
  </span>
</template>
<style>
.formula-fields {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  flex-wrap: wrap;
}
.formula-field {
  display: inline-flex;
  align-items: center;
  min-width: 0;
}
.formula-field input {
  font-size: 18px;
  min-width: 22px;
  max-width: 100%;
  padding: 4px 3px;
}
.formula-fraction {
  display: inline-flex;
  flex-direction: column;
  align-items: stretch;
  gap: 3px;
}
.formula-fraction > .formula-fields {
  justify-content: center;
  padding: 3px;
}
.formula-fraction > .formula-fields:first-child {
  border-bottom: 1px solid var(--text);
}
.formula-sub {
  align-self: flex-end;
  transform: translateY(6px);
}
.formula-super {
  align-self: flex-start;
  transform: translateY(-6px);
}
.formula-sub input,
.formula-super input {
  font-size: 13px;
}
.formula-bar {
  border-top: 1px solid var(--text);
}
</style>
