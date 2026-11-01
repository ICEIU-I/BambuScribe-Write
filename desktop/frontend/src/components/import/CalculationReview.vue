<script setup lang="ts">
import type { CalculationPlan, CalculationVariable } from '../../services/calculationTypes'
const props = defineProps<{ plan: CalculationPlan; disabled: boolean }>()
const emit = defineEmits<{ change: [] }>()
function confirmIssue(index: number) {
  props.plan.issues.splice(index, 1)
  emit('change')
}
function confirmValue(variable: CalculationVariable) {
  variable.uncertain = false
  emit('change')
}
</script>
<template>
  <div class="calculation-review">
    <div v-if="plan.issues.length" class="calculation-issues">
      <label v-for="(issue, i) in plan.issues" :key="issue"
        ><input type="checkbox" :disabled="disabled" @change="confirmIssue(i)" />{{
          issue
        }}（核对后勾选）</label
      >
    </div>
    <h3>读数与参数</h3>
    <div class="calculation-values">
      <label
        v-for="v in plan.variables"
        :key="v.id"
        :class="{ uncertain: v.uncertain || !v.value }"
      >
        <span
          >{{ v.label }}<small>{{ v.source }}</small></span
        >
        <input
          v-model="v.value"
          :aria-label="v.label"
          :disabled="disabled"
          inputmode="decimal"
          @input="confirmValue(v)"
        />
        <span>{{ v.unit }}</span>
        <button
          v-if="v.uncertain"
          class="text-button"
          :disabled="disabled"
          @click="confirmValue(v)"
        >
          确认读数
        </button>
      </label>
    </div>
    <details class="calculation-rules">
      <summary>计算步骤与精度</summary>
      <section v-for="(section, si) in plan.sections" :key="si">
        <input
          v-model="section.title"
          :aria-label="`第${si + 1}组标题`"
          :disabled="disabled"
          @input="emit('change')"
        />
        <details v-for="step in section.steps" :key="step.id">
          <summary>
            {{ step.label }} · {{ step.format === 'decimal' ? '小数位' : '有效数字' }}
            {{ step.precision }}
          </summary>
          <label
            >计算式<input
              v-model="step.expression"
              :disabled="disabled"
              :aria-label="`${step.id}计算式`"
              @input="emit('change')"
          /></label>
          <small
            >可引用读数编号：{{
              plan.variables.map((v) => `${v.id}（${v.label}）`).join('、')
            }}</small
          >
          <label
            >结果单位<input v-model="step.unit" :disabled="disabled" @input="emit('change')"
          /></label>
          <label
            >精度<select v-model="step.format" :disabled="disabled" @change="emit('change')">
              <option value="decimal">小数位</option>
              <option value="significant">有效数字</option></select
            ><input
              v-model.number="step.precision"
              type="number"
              min="0"
              max="12"
              :disabled="disabled"
              @input="emit('change')"
          /></label>
          <label
            ><input
              v-model="step.useRounded"
              type="checkbox"
              :disabled="disabled"
              @change="emit('change')"
            />后续计算使用舍入后的值</label
          >
          <label v-for="(c, ci) in step.conversions" :key="ci"
            >换算为{{ c.unit
            }}<input
              v-model="c.factor"
              :disabled="disabled"
              aria-label="换算系数"
              @input="emit('change')" /><select
              v-model="c.format"
              :disabled="disabled"
              @change="emit('change')"
            >
              <option value="decimal">小数位</option>
              <option value="significant">有效数字</option></select
            ><input
              v-model.number="c.precision"
              type="number"
              min="0"
              max="12"
              :disabled="disabled"
              @input="emit('change')"
          /></label>
        </details>
      </section>
    </details>
  </div>
</template>
