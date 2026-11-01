<script setup lang="ts">
import { ref } from 'vue'
import { native, errorText } from '../../services/bridge'
import type { AgentConfig } from '../../services/tableImportTypes'
defineProps<{ disabled: boolean }>()
const emit = defineEmits<{ error: [message: string] }>()
const config = ref<AgentConfig>({
  baseURL: 'https://api.openai.com/v1',
  model: '',
  protocol: 'responses',
  apiKey: '',
})
const settings = ref(false),
  saving = ref(false)
async function load() {
  try {
    config.value = { ...(await native().AgentConfig()), apiKey: '' }
    settings.value = !config.value.hasKey || !config.value.model
  } catch (e) {
    emit('error', errorText(e))
  }
}
async function save() {
  saving.value = true
  try {
    await native().SaveAgentConfig(config.value)
    config.value = { ...(await native().AgentConfig()), apiKey: '' }
    settings.value = false
  } finally {
    saving.value = false
  }
}
async function clickSave() {
  try {
    await save()
  } catch (e) {
    emit('error', errorText(e))
  }
}
async function ensure() {
  if (config.value.apiKey || settings.value) await save()
}
defineExpose({ load, ensure })
</script>
<template>
  <button class="import-config-toggle" :disabled="disabled" @click="settings = !settings">
    模型连接 {{ settings ? '▴' : '▾'
    }}<span v-if="!settings && config.model"> · {{ config.model }}</span>
  </button>
  <div v-if="settings" class="import-config">
    <label class="import-field"
      >API 地址<input v-model="config.baseURL" aria-label="模型接口地址" :disabled="disabled"
    /></label>
    <label class="import-field"
      >模型<input v-model="config.model" aria-label="模型名称" :disabled="disabled"
    /></label>
    <label class="import-field"
      >协议<select v-model="config.protocol" aria-label="模型接口协议" :disabled="disabled">
        <option value="responses">Responses</option>
        <option value="chat-completions">Chat Completions</option>
      </select></label
    >
    <label class="import-field"
      >API Key<input
        v-model="config.apiKey"
        type="password"
        autocomplete="off"
        aria-label="模型 API Key"
        :placeholder="config.hasKey ? '已保存，留空保留' : '填写 API Key'"
        :disabled="disabled"
    /></label>
    <button class="button" :disabled="disabled || saving" @click="clickSave">保存连接</button>
  </div>
</template>
