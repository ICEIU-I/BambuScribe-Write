<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { useCalculationImport } from '../../state/useCalculationImport'
import { errorText } from '../../services/bridge'
import ImportModelConfig from './ImportModelConfig.vue'
import CalculationImages from './CalculationImages.vue'
import CalculationImageViewer from './CalculationImageViewer.vue'
import type { SourceImage } from '../../services/calculationTypes'
import CalculationPaper from './CalculationPaper.vue'
import CalculationReview from './CalculationReview.vue'
import './calculation-import.css'
const props = defineProps<{ task: ReturnType<typeof useCalculationImport> }>()
const sourceImage = ref<SourceImage>()
const dialog = ref<HTMLDialogElement>(),
  model = ref<InstanceType<typeof ImportModelConfig>>()
watch(
  () => props.task.open.value,
  async (open) => {
    if (!open) {
      sourceImage.value = undefined
      dialog.value?.close()
      return
    }
    await nextTick()
    dialog.value?.showModal()
    if (!props.task.editing.value) await model.value?.load()
  },
)
async function run() {
  try {
    await model.value?.ensure()
    await props.task.recognize()
  } catch (e) {
    props.task.error.value = errorText(e)
  }
}
</script>
<template>
  <dialog
    ref="dialog"
    class="calculation-dialog"
    :class="{ 'has-result': task.plan.value }"
    aria-label="数据处理"
    @cancel.prevent="!task.saving.value && !task.reading.value && task.close()"
  >
    <CalculationImageViewer :image="sourceImage" @close="sourceImage = undefined" />
    <div class="calculation-heading">
      <h2>{{ task.editing.value ? '编辑数据处理' : '数据处理' }}</h2>
      <button
        class="icon-button"
        aria-label="关闭数据处理"
        :disabled="task.saving.value || task.reading.value"
        @click="task.close()"
      >
        ×
      </button>
    </div>
    <div v-if="!task.plan.value" class="calculation-inputs">
      <CalculationImages
        title="原始数据"
        :images="task.data.value"
        :disabled="task.working.value || task.reading.value"
        @add="task.add('data', $event)"
        @view="sourceImage = $event"
        @change="task.data.value = $event"
      />
      <CalculationImages
        title="参考样例"
        :images="task.references.value"
        :disabled="task.working.value || task.reading.value"
        @add="task.add('references', $event)"
        @view="sourceImage = $event"
        @change="task.references.value = $event"
      />
      <label class="calculation-instruction"
        >补充要求（可选）<textarea
          v-model="task.instruction.value"
          rows="2"
          maxlength="2000"
          :disabled="task.working.value"
          placeholder="例如：灵敏度用242，按参考步骤计算"
        />
      </label>
      <ImportModelConfig
        ref="model"
        :disabled="task.working.value"
        @error="task.error.value = $event"
      />
    </div>
    <div v-else class="calculation-result">
      <div class="calculation-preview-pane">
        <label class="calculation-scale"
          >分式字号比例<input
            v-model.number="task.fractionScale.value"
            aria-label="分式字号比例"
            type="number"
            min="50"
            max="100"
            :disabled="task.saving.value"
            @input="task.changed()"
          />%</label
        >
        <CalculationPaper v-if="task.preview.value" :document="task.preview.value.document" />
        <div v-else class="calculation-empty">
          {{ task.planning.value ? '正在排版…' : '核对右侧数据后显示预览' }}
        </div>
      </div>
      <div class="calculation-review-pane">
        <details v-if="task.data.value.length || task.references.value.length">
          <summary>查看原图</summary>
          <div class="calculation-reference-images">
            <button
              v-for="im in [...task.data.value, ...task.references.value]"
              :key="im.id"
              class="calculation-image-open"
              :aria-label="`查看${im.name}`"
              @click="sourceImage = im"
            >
              <img :src="im.data" :alt="im.name" />
            </button>
          </div>
        </details>
        <CalculationReview
          :plan="task.plan.value"
          :disabled="task.saving.value"
          @change="task.changed()"
        />
      </div>
    </div>
    <p v-if="task.error.value" class="import-error" role="alert">{{ task.error.value }}</p>
    <div class="calculation-footer">
      <span>{{
        task.working.value
          ? '正在结合图片识别计算步骤…'
          : task.reading.value
            ? '正在读取图片…'
            : task.plan.value
              ? '双击插入后的公式，可再次修改数据。'
              : '同一组计算会连续排版，按当前纸张自动分页。'
      }}</span>
      <button
        v-if="task.plan.value && !task.editing.value"
        class="button"
        :disabled="task.saving.value"
        @click="task.plan.value = undefined"
      >
        返回图片
      </button>
      <button v-if="task.working.value" class="button" @click="task.close()">取消</button>
      <button
        v-else-if="!task.plan.value"
        class="button primary"
        :disabled="task.reading.value || !task.data.value.length || !task.references.value.length"
        @click="run"
      >
        生成预览
      </button>
      <button
        v-else
        class="button primary"
        :disabled="task.saving.value || task.planning.value || !task.preview.value"
        @click="task.insert()"
      >
        {{ task.saving.value ? '正在应用…' : task.editing.value ? '更新正文' : '插入正文' }}
      </button>
    </div>
  </dialog>
</template>
